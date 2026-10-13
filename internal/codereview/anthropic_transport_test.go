package codereview

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicTestServer answers /v1/messages with the given replies in order,
// recording each request body.
func anthropicTestServer(t *testing.T, replies ...func(w http.ResponseWriter)) (*httptest.Server, *[]map[string]any, *atomic.Int32) {
	t.Helper()
	var bodies []map[string]any
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1)) - 1
		if !strings.HasSuffix(r.URL.Path, "/v1/messages") {
			t.Errorf("request path = %s, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("X-Api-Key"); got != "sk-ant-test" {
			t.Errorf("x-api-key = %q, want the configured key", got)
		}
		if r.Header.Get("Anthropic-Version") == "" {
			t.Error("request carries no anthropic-version header")
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		bodies = append(bodies, body)
		if n >= len(replies) {
			t.Errorf("unexpected request %d", n+1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		replies[n](w)
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies, &calls
}

func anthropicReply(status int, body string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		// Keeps the SDK's retry backoff at a millisecond in tests.
		w.Header().Set("Retry-After-Ms", "1")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func anthropicMessage(stopReason, text string) string {
	reply, _ := json.Marshal(map[string]any{
		"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-sonnet-4-6",
		"content":     []map[string]any{{"type": "text", "text": text}},
		"stop_reason": stopReason,
		"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
	})
	return string(reply)
}

func testAnthropicTransport(srv *httptest.Server) *anthropicTransport {
	return newAnthropicTransport(anthropicCredential{apiKey: "sk-ant-test"}, option.WithBaseURL(srv.URL))
}

// The prompt has to arrive as the system prompt plus one user turn, under the
// model's own API name, with the output cap the caller asked for.
func TestAnthropicTransportSendsTheReviewAsOneMessagesCall(t *testing.T) {
	srv, bodies, _ := anthropicTestServer(t, anthropicReply(http.StatusOK, anthropicMessage("end_turn", `{"findings":[]}`)))

	completion, err := testAnthropicTransport(srv).complete(context.Background(), "anthropic:claude-sonnet-4-6", "review rules", "the diff", 6000)
	if err != nil {
		t.Fatalf("complete() error = %v", err)
	}
	if completion.Text != `{"findings":[]}` || completion.StopReason != "end_turn" {
		t.Fatalf("completion = %+v, want the reply text and its stop reason", completion)
	}
	body := (*bodies)[0]
	if body["model"] != "claude-sonnet-4-6" {
		t.Fatalf("model = %v, want claude-sonnet-4-6 with its prefix stripped", body["model"])
	}
	if body["max_tokens"] != float64(6000) {
		t.Fatalf("max_tokens = %v, want 6000", body["max_tokens"])
	}
	if system, _ := json.Marshal(body["system"]); !strings.Contains(string(system), "review rules") {
		t.Fatalf("system = %s, want the review prompt", system)
	}
	messages, _ := body["messages"].([]any)
	if len(messages) != 1 || !strings.Contains(mustJSON(messages[0]), "the diff") {
		t.Fatalf("messages = %v, want one user turn carrying the input", body["messages"])
	}
}

// A reply cut off at the cap has to read as truncated, as it does on every
// other wire, or the parse site blames the model's formatting.
func TestAnthropicTransportReportsATruncatedReply(t *testing.T) {
	srv, _, _ := anthropicTestServer(t, anthropicReply(http.StatusOK, anthropicMessage("max_tokens", `{"findings":[`)))

	completion, err := testAnthropicTransport(srv).complete(context.Background(), "claude-haiku-4-5", "", "x", 100)
	if err != nil {
		t.Fatalf("complete() error = %v", err)
	}
	if !completion.truncated() {
		t.Fatalf("completion = %+v, want it reported as truncated", completion)
	}
}

// A declined review must fail the leg, never pass for a review with nothing in it.
func TestAnthropicTransportFailsALegTheModelDeclined(t *testing.T) {
	srv, _, _ := anthropicTestServer(t, anthropicReply(http.StatusOK, anthropicMessage("refusal", "")))

	if _, err := testAnthropicTransport(srv).complete(context.Background(), "claude-sonnet-4-6", "", "x", 100); err == nil || !strings.Contains(err.Error(), "refusal") {
		t.Fatalf("complete() error = %v, want the refusal reported", err)
	}
}

// Throttles are retried; a request the API rejects is not.
func TestAnthropicTransportRetriesThrottlesButNotRejections(t *testing.T) {
	srv, _, calls := anthropicTestServer(t,
		anthropicReply(http.StatusTooManyRequests, `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`),
		anthropicReply(http.StatusOK, anthropicMessage("end_turn", "ok")),
	)
	if _, err := testAnthropicTransport(srv).complete(context.Background(), "claude-sonnet-4-6", "", "x", 100); err != nil {
		t.Fatalf("complete() after one 429 = %v, want the retry to succeed", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want the throttle retried once", calls.Load())
	}

	srv, _, calls = anthropicTestServer(t,
		anthropicReply(http.StatusBadRequest, `{"type":"error","error":{"type":"invalid_request_error","message":"bad model"}}`),
	)
	_, err := testAnthropicTransport(srv).complete(context.Background(), "claude-sonnet-4-6", "", "x", 100)
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad model") {
		t.Fatalf("complete() on a 400 = %v, want the status and the API's reason", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want a rejected request sent once", calls.Load())
	}
}

// The provider follows the model's own name, so a panel can mix the two and a
// key only ever reaches the provider it belongs to.
func TestModelsRouteToTheirProvider(t *testing.T) {
	for model, openAI := range map[string]bool{
		"claude-haiku-4-5":            false,
		"claude-opus-5-5":             false,
		"anthropic:claude-sonnet-4-6": false,
		"gpt-5.6-luna":                true,
		"openai:gpt-5.2":              true,
		"o3":                          true,
	} {
		if got := modelUsesOpenAI(model); got != openAI {
			t.Errorf("modelUsesOpenAI(%q) = %v, want %v", model, got, openAI)
		}
	}
	if got := apiModelName("openai:gpt-5.2"); got != "gpt-5.2" {
		t.Fatalf("apiModelName(openai:gpt-5.2) = %q, want the prefix stripped", got)
	}
}

// modelKeyEnv clears every key and model override a review could find.
func modelKeyEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY", "GX_REVIEW_AI",
		"GX_REVIEW_MODELS", "GX_REVIEW_MODEL_A", "GX_REVIEW_MODEL_B", "GX_REVIEW_JUDGE_MODEL",
	} {
		t.Setenv(name, "")
	}
}

// Either key on its own is enough for a review, and the panel follows it.
func TestThePanelFollowsTheKeyThatIsSet(t *testing.T) {
	modelKeyEnv(t)
	if reason := NoReviewerReason(); !strings.Contains(reason, "ANTHROPIC_API_KEY") || !strings.Contains(reason, "OPENAI_API_KEY") {
		t.Fatalf("NoReviewerReason() with no key = %q, want both keys named as the fix", reason)
	}

	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	if a, b := resolveReviewModels(); a != defaultReviewModelA || b != defaultReviewModelB {
		t.Fatalf("panel with ANTHROPIC_API_KEY = %q, %q; want the Claude panel", a, b)
	}
	if reason := NoReviewerReason(); reason != "" {
		t.Fatalf("NoReviewerReason() with ANTHROPIC_API_KEY = %q, want a reviewer", reason)
	}

	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")
	if a, b := resolveReviewModels(); a != defaultOpenAIReviewModelA || b != defaultOpenAIReviewModelB {
		t.Fatalf("panel with OPENAI_API_KEY alone = %q, %q; want the OpenAI panel", a, b)
	}
	if judge := resolveJudgeModel(); judge != defaultOpenAIJudgeModel {
		t.Fatalf("judge with OPENAI_API_KEY alone = %q, want %q", judge, defaultOpenAIJudgeModel)
	}
	if reason := NoReviewerReason(); reason != "" {
		t.Fatalf("NoReviewerReason() with OPENAI_API_KEY = %q, want a reviewer", reason)
	}

	// Both keys: Claude, unless GX_REVIEW_MODELS says otherwise.
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	if a, _ := resolveReviewModels(); a != defaultReviewModelA {
		t.Fatalf("leg A with both keys = %q, want the Claude panel", a)
	}
	t.Setenv("GX_REVIEW_MODELS", "openai")
	if a, _ := resolveReviewModels(); a != defaultOpenAIReviewModelA {
		t.Fatalf("leg A with GX_REVIEW_MODELS=openai = %q, want the OpenAI panel", a)
	}
}

// A leg whose provider has no key is unavailable with that key named, and does
// not take the other leg down with it.
func TestALegWithoutItsKeyNamesTheKey(t *testing.T) {
	modelKeyEnv(t)
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")
	t.Setenv("GX_REVIEW_MODEL_A", "claude-sonnet-4-6")
	t.Setenv("GX_REVIEW_MODEL_B", "gpt-5.6-terra")

	if reason := NoReviewerReason(); reason != "" {
		t.Fatalf("NoReviewerReason() = %q with the OpenAI leg available, want a reviewer", reason)
	}
	if reason := ReviewerUnavailableReason(reviewerFromEnv()); !strings.Contains(reason, "claude-sonnet-4-6 runs on the Anthropic API, which needs ANTHROPIC_API_KEY") {
		t.Fatalf("ReviewerUnavailableReason() = %q, want the Claude leg's missing key named", reason)
	}
}

// The Anthropic credential can be an API key or an OAuth access token.
func TestAnthropicCredentialAcceptsAKeyOrAToken(t *testing.T) {
	modelKeyEnv(t)
	if _, ok := anthropicCredentialFromEnv(); ok {
		t.Fatal("anthropicCredentialFromEnv() found a credential in an empty environment")
	}
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "oauth-token")
	if cred, ok := anthropicCredentialFromEnv(); !ok || cred.authToken != "oauth-token" {
		t.Fatalf("anthropicCredentialFromEnv() = %+v, %v; want the access token", cred, ok)
	}
	transport, err := transportForModel("claude-haiku-4-5")
	if err != nil || !strings.Contains(transport.detail(), "Anthropic API") {
		t.Fatalf("transportForModel(claude-haiku-4-5) = %v, %v; want the Anthropic wire", transport, err)
	}
}
