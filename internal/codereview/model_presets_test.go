package codereview

import (
	"strings"
	"testing"
)

// presetEnv clears every key and per-slot override, leaving only what a test sets.
func presetEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY", "GX_REVIEW_MODELS",
		"GX_REVIEW_MODEL_A", "GX_REVIEW_MODEL_B", "GX_REVIEW_JUDGE_MODEL", ruleSortModelEnvVar,
	} {
		t.Setenv(name, "")
	}
}

// A preset fills every slot, so switching provider is one variable.
func TestModelPresetFillsEverySlot(t *testing.T) {
	presetEnv(t)
	t.Setenv("GX_REVIEW_MODELS", "openai")
	a, b := resolveReviewModels()
	if a != defaultOpenAIReviewModelA || b != defaultOpenAIReviewModelB {
		t.Fatalf("openai panel legs = %q, %q", a, b)
	}
	if j := resolveJudgeModel(); j != defaultOpenAIJudgeModel {
		t.Fatalf("openai panel judge = %q", j)
	}
	if r := resolveRuleSortModel(); r != defaultOpenAIRuleSortModel {
		t.Fatalf("openai panel rule sorter = %q", r)
	}
}

func TestExplicitEnvBeatsPreset(t *testing.T) {
	presetEnv(t)
	t.Setenv("GX_REVIEW_MODELS", "openai")
	t.Setenv("GX_REVIEW_JUDGE_MODEL", "claude-sonnet-4-6")
	if j := resolveJudgeModel(); j != "claude-sonnet-4-6" {
		t.Fatalf("explicit GX_REVIEW_JUDGE_MODEL must beat the preset, got %q", j)
	}
	if _, b := resolveReviewModels(); b != defaultOpenAIReviewModelB {
		t.Fatalf("slots without an explicit env still take the preset, got %q", b)
	}
}

func TestUnknownPresetIsIgnoredByResolversAndCaughtByValidate(t *testing.T) {
	presetEnv(t)
	t.Setenv("GX_REVIEW_MODELS", "opneai")
	if _, b := resolveReviewModels(); b != defaultReviewModelB {
		t.Fatalf("unknown preset must leave the default panel alone, got %q", b)
	}
	err := ValidateModelPresetEnv()
	if err == nil || !strings.Contains(err.Error(), "openai") || !strings.Contains(err.Error(), "opneai") {
		t.Fatalf("validate should name the bad value and the valid presets, got %v", err)
	}
	t.Setenv("GX_REVIEW_MODELS", "")
	if err := ValidateModelPresetEnv(); err != nil {
		t.Fatalf("unset is fine, got %v", err)
	}
	t.Setenv("GX_REVIEW_MODELS", "Anthropic")
	if err := ValidateModelPresetEnv(); err != nil {
		t.Fatalf("preset names are case-insensitive, got %v", err)
	}
}

func TestFriendlyModelNames(t *testing.T) {
	for in, want := range map[string]string{
		"claude-haiku-4-5":          "Claude Haiku 4.5",
		"claude-sonnet-4-6":         "Claude Sonnet 4.6",
		"claude-haiku-4-5-20251001": "Claude Haiku 4.5",
		"claude-opus-5-5":           "Claude Opus 5.5",
		"gpt-5.6-luna":              "GPT-5.6 Luna",
		"openai:gpt-5.2":            "GPT-5.2",
	} {
		if got := friendlyModelName(in); got != want {
			t.Errorf("friendlyModelName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The shipped judge: Haiku 4.5. Verification is the per-finding call, and
// judge batches are byte-bounded well inside Haiku's 200K context.
func TestShippedJudgeDefaultIsHaiku(t *testing.T) {
	presetEnv(t)
	if j := resolveJudgeModel(); j != "claude-haiku-4-5" {
		t.Fatalf("default judge = %q, want Haiku 4.5", j)
	}
}
