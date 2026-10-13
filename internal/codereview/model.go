package codereview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// gx calls models with the reviewer's own API key, on one of two wires: the
// Anthropic API for Claude models and the OpenAI API for everything else.
// There is no server in between.

// modelTransport is how a review leg's request reaches its provider.
//
// Both legs, the judge, the duplicate adjudicator and the rule sorter share one
// request shape and one response parser (modelReviewer.completeJSON); only the
// wire differs. Keeping that difference behind this interface is what lets a
// Claude leg and a GPT leg sit in the same panel without a second
// implementation of prompt construction, parsing, or failure reporting.
type modelTransport interface {
	// complete performs one model call and returns the model's reply.
	complete(ctx context.Context, model, system, input string, maxOutputTokens int) (modelCompletion, error)
	// detail is the human phrase naming this wire, reported on every review so
	// that "slow" and "denied" stay answerable after the fact.
	detail() string
}

// modelCompletion is one model reply: the text, and why the model stopped.
//
// The stop reason is carried because a reply cut off at the output cap and a
// reply the model formatted badly are the same symptom at the parse site — JSON
// that will not decode — and completely different problems. Every wire reports
// it in the Anthropic dialect ("end_turn", "max_tokens"), so the distinction
// survives whichever one a leg is on.
type modelCompletion struct {
	Text       string
	StopReason string
}

// truncated reports whether the model ran out of output budget mid-reply.
func (c modelCompletion) truncated() bool {
	return strings.EqualFold(strings.TrimSpace(c.StopReason), "max_tokens")
}

// modelRetryBackoffs is how long a wire waits before each retry of a throttled
// or failed call. A model call is minutes of work, so waiting seconds to save
// one is always the right trade.
var modelRetryBackoffs = []time.Duration{2 * time.Second, 8 * time.Second, 20 * time.Second}

// modelCallTimeout bounds one model call. Flagship models reading a large
// brief routinely take minutes.
const modelCallTimeout = 10 * time.Minute

// Model name prefixes from when Claude and GPT models shared a namespace. They
// are still accepted, and stripped before the name reaches a provider.
const (
	anthropicModelPrefix = "anthropic:"
	openAIModelPrefix    = "openai:"
)

// modelUsesOpenAI reports whether a model runs on the OpenAI API. The provider
// follows the model's own name: Claude models run on the Anthropic API, and
// everything else on OpenAI's.
func modelUsesOpenAI(model string) bool {
	model = strings.TrimSpace(model)
	if strings.HasPrefix(model, openAIModelPrefix) {
		return true
	}
	return !strings.HasPrefix(strings.TrimPrefix(model, anthropicModelPrefix), "claude")
}

// apiModelName is the model's name as its provider knows it.
func apiModelName(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, anthropicModelPrefix)
	return strings.TrimSpace(strings.TrimPrefix(model, openAIModelPrefix))
}

// modelProviderName is the short provider label a leg's attribution carries.
func modelProviderName(model string) string {
	if modelUsesOpenAI(model) {
		return "OpenAI"
	}
	return "Anthropic"
}

// transportForModel builds the wire one model runs on, or says which key it is
// missing. Each caller gets its own transport, so the two reviewers and the
// judge do not share an HTTP client's connection budget.
//
// Errors are returned, never swallowed: a review with no reviewer must read as
// "nothing reviewed this", never as a clean review.
func transportForModel(model string) (modelTransport, error) {
	if modelUsesOpenAI(model) {
		return newOpenAITransport(model)
	}
	cred, ok := anthropicCredentialFromEnv()
	if !ok {
		if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
			return nil, fmt.Errorf("no model API key is set: set ANTHROPIC_API_KEY to review with Claude, or OPENAI_API_KEY to review with OpenAI models")
		}
		return nil, fmt.Errorf("%s runs on the Anthropic API, which needs ANTHROPIC_API_KEY; unset GX_REVIEW_MODELS and the per-leg model overrides to review with OpenAI models instead", apiModelName(model))
	}
	return newAnthropicTransport(cred), nil
}

// modelReviewer is one review leg: a model plus the wire it is reached over. It
// owns prompt construction and response parsing; getting the bytes to the
// provider is the transport's problem.
type modelReviewer struct {
	model     string
	transport modelTransport
}

func newModelReviewer(transport modelTransport, model string) *modelReviewer {
	return &modelReviewer{model: model, transport: transport}
}

// legLabel names a leg by its model and provider, so a merged finding's
// "Reviewer" evidence says which model actually raised it. "Reviewer A" alone
// is a slot, not an attribution.
func legLabel(slot, model string) string {
	if friendly := friendlyModelName(model); friendly != "" {
		return slot + " (" + friendly + " via " + modelProviderName(model) + ")"
	}
	return slot
}

// friendlyModelName turns a model ID into the name a person uses:
// claude-haiku-4-5 → "Claude Haiku 4.5", claude-sonnet-4-6 → "Claude Sonnet
// 4.6", gpt-5.6-luna → "GPT-5.6 Luna". The report ledger and run details show
// this; the JSON keeps the exact ID, because "which snapshot" is a question
// that comes up the moment a review is slow or wrong.
func friendlyModelName(model string) string {
	short := apiModelName(model)
	if short == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(short, "gpt-"); ok {
		// gpt-5.6-luna → "GPT-5.6 Luna"; gpt-5.2 → "GPT-5.2"
		if i := strings.IndexByte(rest, '-'); i > 0 && i < len(rest)-1 && !isAllDigits(rest[i+1:]) {
			return "GPT-" + rest[:i] + " " + strings.ToUpper(rest[i+1:i+2]) + rest[i+2:]
		}
		return "GPT-" + rest
	}
	parts := strings.Split(short, "-")
	if len(parts) < 2 || parts[0] != "claude" {
		return short
	}
	family := strings.ToUpper(parts[1][:1]) + parts[1][1:]
	// Version is the run of numeric segments after the family, joined with a
	// dot; a trailing 8-digit date is a snapshot stamp, not part of the version.
	var version []string
	for _, seg := range parts[2:] {
		if len(seg) == 8 && isAllDigits(seg) {
			break
		}
		if isAllDigits(seg) {
			version = append(version, seg)
			continue
		}
		break
	}
	name := "Claude " + family
	if len(version) > 0 {
		name += " " + strings.Join(version, ".")
	}
	return name
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// resolveReviewModels picks the model for each leg: env > the active panel.
//
// Which model reviews is the operator's call: a REVIEW.md line used to win over
// both, which let the repository under review choose the reviewer that judged
// it — including choosing a weaker one.
func resolveReviewModels() (string, string) {
	preset := activeModelPreset()
	modelA := firstNonEmpty(os.Getenv("GX_REVIEW_MODEL_A"), preset.ReviewerA)
	modelB := firstNonEmpty(os.Getenv("GX_REVIEW_MODEL_B"), preset.ReviewerB)
	if legDisabled(modelA) {
		modelA = ""
	}
	if legDisabled(modelB) {
		modelB = ""
	}
	return strings.TrimSpace(modelA), strings.TrimSpace(modelB)
}

// legDisabled reports whether a leg was explicitly turned off, which is how a
// single-model review is requested: GX_REVIEW_MODEL_B=off.
//
// A panel is two models so their misses are uncorrelated, which is worth its
// cost on a pull request. It is not always worth its latency: an interactive
// review a developer is waiting on is a different product than a review nobody
// is watching, and the second leg adds its slowest call to the critical path.
func legDisabled(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "", "0", "off", "none", "disabled":
		return true
	}
	return false
}

func (r *modelReviewer) Available() bool { return r != nil && r.transport != nil }

func (r *modelReviewer) Review(ctx context.Context, brief ReviewBrief) ([]Finding, error) {
	summary, err := r.ReviewForSummary(ctx, brief)
	return summary.Findings, err
}

func (r *modelReviewer) ReviewForSummary(ctx context.Context, brief ReviewBrief) (PRSummaryReview, error) {
	brief = compactReviewBriefForAI(brief)
	system := reviewDeveloperPrompt(brief)
	input := mustJSON(brief)
	// One retry, for malformed replies only. A leg sometimes answers in prose
	// with no JSON at all ("I reviewed the change…"), which extractJSONObject
	// cannot recover; observed live on a binary-only diff, and it turned a
	// healthy panel into a DEGRADED run. The retry restates the JSON-only
	// requirement at the top of the system prompt, which is enough for a model
	// that drifted once. A truncated reply is not retried: the fix for that is
	// the output cap, not another attempt at the same cap.
	var lastParseErr error
	for attempt := 0; attempt < 2; attempt++ {
		prompt := system
		if attempt > 0 {
			prompt = reviewJSONOnlyReminder + "\n" + system
		}
		completion, err := r.completeJSON(ctx, prompt, input, defaultReviewMaxOutputTokens)
		if err != nil {
			return PRSummaryReview{}, err
		}
		output, parseErr := parseAIReviewOutput(completion.Text, brief)
		if parseErr == nil {
			return aiReviewOutputToPRSummaryReview(output), nil
		}
		if errors.Is(parseErr, errTruncatedButSalvaged) {
			// The cap cut the reply, but complete findings were recovered.
			// Return them; the truncation is reported through the summary's
			// Truncated field so the report can say the leg was cut short
			// rather than pretending it answered in full.
			review := aiReviewOutputToPRSummaryReview(output)
			review.Truncated = describeTruncatedCompletion("AI review", defaultReviewMaxOutputTokens, parseErr).Error()
			return review, nil
		}
		if completion.truncated() {
			return PRSummaryReview{}, describeTruncatedCompletion("AI review", defaultReviewMaxOutputTokens, parseErr)
		}
		lastParseErr = parseErr
	}
	return PRSummaryReview{}, lastParseErr
}

// reviewJSONOnlyReminder is prepended to the system prompt on the one retry
// after a reply that carried no decodable JSON. It is deliberately blunt: the
// first prompt already says "Return JSON only", and the model ignored it.
const reviewJSONOnlyReminder = "Your previous reply was not JSON and could not be used. Reply with the JSON object only: no prose before it, no markdown headings, no summary after it. If there is nothing to report, reply exactly {\"recommendations\":[]}."

// completeJSON is the single model call. Both reviewer legs and the judge go
// through it, so the request shape and the response parsing live in one place.
func (r *modelReviewer) completeJSON(ctx context.Context, system string, input string, maxOutputTokens int) (modelCompletion, error) {
	if r == nil || r.transport == nil {
		return modelCompletion{}, fmt.Errorf("reviewer has no model transport")
	}
	completion, err := r.transport.complete(ctx, r.model, system, input, maxOutputTokens)
	dumpReviewExchange(r.model, system, input, completion, err)
	return completion, err
}

// dumpReviewExchange writes what a leg actually sent and received to the
// directory named by GX_REVIEW_DUMP_DIR.
//
// A review that returns no findings is indistinguishable, from the outside,
// between "the model read the change and found nothing", "the prompt talked it
// out of reporting", and "the reply did not parse". Those need completely
// different fixes, and the only way to tell them apart is to read the exchange.
// Off unless the variable is set; it writes prompts verbatim, so it is a
// debugging tool and not something to leave on.
func dumpReviewExchange(model, system, input string, completion modelCompletion, callErr error) {
	dir := strings.TrimSpace(os.Getenv("GX_REVIEW_DUMP_DIR"))
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	safeModel := strings.NewReplacer("/", "_", ":", "_", ".", "-").Replace(model)
	stamp := time.Now().UTC().Format("150405.000")
	body := strings.Join([]string{
		"=== MODEL: " + model,
		"=== SYSTEM PROMPT (" + strconv.Itoa(len(system)) + " bytes):",
		system,
		"=== INPUT BRIEF (" + strconv.Itoa(len(input)) + " bytes):",
		input,
		"=== STOP REASON: " + completion.StopReason,
		"=== ERROR: " + fmt.Sprint(callErr),
		"=== REPLY (" + strconv.Itoa(len(completion.Text)) + " bytes):",
		completion.Text,
	}, "\n")
	_ = os.WriteFile(filepath.Join(dir, safeModel+"-"+stamp+".txt"), []byte(body), 0o644)
}
