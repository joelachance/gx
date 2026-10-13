package cli

import (
	"strings"
	"testing"
)

// noReviewerEnv is an environment with nothing to review with: no model key and
// no model overrides.
func noReviewerEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GX_HOME", t.TempDir())
	for _, name := range []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY", "GX_REVIEW_AI",
		"GX_REVIEW_MODELS", "GX_REVIEW_MODEL_A", "GX_REVIEW_MODEL_B",
	} {
		t.Setenv(name, "")
	}
}

// A review with no reviewer produced a full report, a verdict, and a findings
// file with no model having read a line of the change. Refusing is not a new
// restriction — there was no reviewer either way — it is the difference
// between saying so and shipping something that reads like a review.
func TestReviewRefusesWithoutAReviewer(t *testing.T) {
	noReviewerEnv(t)

	err := requireReviewer()
	if err == nil {
		t.Fatal("requireReviewer() = nil with no model key set, want a refusal")
	}
	// The message has to name the fixes, or it reads like the DEGRADED line it
	// replaces.
	for _, want := range []string{"gx review cannot run", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GX_REVIEW_AI=0"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want it to mention %q", err, want)
		}
	}
}

// Either provider's key is the whole of a review's setup.
func TestReviewRunsOnEitherKey(t *testing.T) {
	noReviewerEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	if err := requireReviewer(); err != nil {
		t.Fatalf("requireReviewer() = %v with ANTHROPIC_API_KEY set, want nil", err)
	}

	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")
	if err := requireReviewer(); err != nil {
		t.Fatalf("requireReviewer() = %v with OPENAI_API_KEY set, want nil", err)
	}
}

// Turning the AI panel off still runs a real review: the deterministic rules.
func TestReviewAllowsTheAIPanelOff(t *testing.T) {
	noReviewerEnv(t)
	t.Setenv("GX_REVIEW_AI", "0")
	if err := requireReviewer(); err != nil {
		t.Fatalf("requireReviewer() = %v with the AI panel off, want nil", err)
	}
}
