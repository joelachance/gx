package codereview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicCredential is how a review reaches the Anthropic API directly: an
// API key, or an OAuth access token (`ant auth print-credentials
// --access-token`) for callers who sign in rather than mint keys.
type anthropicCredential struct {
	apiKey    string
	authToken string
}

// anthropicCredentialFromEnv reads the credential, or reports false when the
// environment carries none.
func anthropicCredentialFromEnv() (anthropicCredential, bool) {
	if key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); key != "" {
		return anthropicCredential{apiKey: key}, true
	}
	if token := strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN")); token != "" {
		return anthropicCredential{authToken: token}, true
	}
	return anthropicCredential{}, false
}

// anthropicTransport implements modelTransport against the Anthropic Messages
// API, on the reviewer's own Anthropic account.
type anthropicTransport struct {
	client anthropic.Client
}

func newAnthropicTransport(cred anthropicCredential, opts ...option.RequestOption) *anthropicTransport {
	base := []option.RequestOption{
		// The SDK's own retries cover what the OpenAI wire's retry loop does
		// (429, 5xx, connection errors), and honour retry-after on top.
		option.WithMaxRetries(len(modelRetryBackoffs)),
		option.WithRequestTimeout(modelCallTimeout),
	}
	if cred.apiKey != "" {
		base = append(base, option.WithAPIKey(cred.apiKey))
	} else {
		base = append(base, option.WithAuthToken(cred.authToken))
	}
	return &anthropicTransport{client: anthropic.NewClient(append(base, opts...)...)}
}

func (t *anthropicTransport) detail() string { return "Anthropic API (api.anthropic.com)" }

func (t *anthropicTransport) complete(ctx context.Context, model, system, input string, maxOutputTokens int) (modelCompletion, error) {
	if t == nil {
		return modelCompletion{}, fmt.Errorf("Anthropic reviewer has no transport")
	}
	id := apiModelName(model)
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultReviewMaxOutputTokens
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(id),
		MaxTokens: int64(maxOutputTokens),
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(input))},
	}
	if strings.TrimSpace(system) != "" {
		params.System = []anthropic.TextBlockParam{{Text: system}}
	}
	resp, err := t.client.Messages.New(ctx, params)
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) {
			return modelCompletion{}, fmt.Errorf("Anthropic model %s returned %d: %s", id, apiErr.StatusCode, firstLine([]byte(apiErr.RawJSON())))
		}
		return modelCompletion{}, fmt.Errorf("call Anthropic model %s: %w", id, err)
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if block, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(block.Text)
		}
	}
	reply := strings.TrimSpace(text.String())
	if resp.StopReason == anthropic.StopReasonRefusal {
		// A declined review is not a review that found nothing; the leg reports
		// it as a failure so the run reads as degraded.
		return modelCompletion{}, fmt.Errorf("Anthropic model %s declined to review this change (stop_reason refusal)", id)
	}
	if reply == "" {
		return modelCompletion{}, fmt.Errorf("Anthropic model %s returned no content (stop_reason %s)", id, resp.StopReason)
	}
	return modelCompletion{Text: reply, StopReason: string(resp.StopReason)}, nil
}
