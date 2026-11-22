package cli

import (
	"strings"
	"testing"
)

func TestInitReviewModelStatusSaysWhatReviewsRunOn(t *testing.T) {
	t.Setenv("GX_HOME", t.TempDir())
	t.Setenv("GX_REVIEW_CLAUDE_CODE", "0")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	if got := initReviewModelStatus(); !strings.Contains(got, "sign in to Claude Code") {
		t.Fatalf("with nothing set, status = %q, want it to say how to get a model", got)
	}
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	if got := initReviewModelStatus(); !strings.Contains(got, "anthropic API key (ANTHROPIC_API_KEY)") {
		t.Fatalf("with an Anthropic key, status = %q", got)
	}
}
