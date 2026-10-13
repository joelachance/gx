package codereview

import (
	"os"
	"sort"
	"strings"
)

// Model presets.
//
// A review fills four model slots: reviewer A, reviewer B, the judge that
// verifies their findings, and the rule sorter. A preset fills all four, and
// there is one per provider gx speaks. With GX_REVIEW_MODELS unset, the panel
// follows the key that is set: ANTHROPIC_API_KEY picks the Claude panel, and
// OPENAI_API_KEY on its own picks the OpenAI one. GX_REVIEW_MODELS=openai
// chooses OpenAI even when both keys are set.
//
// Precedence per slot is explicit env var > panel, so GX_REVIEW_MODELS=openai
// GX_REVIEW_JUDGE_MODEL=claude-sonnet-4-6 swaps everything but the judge.

// The Claude panel: two reviewers across tiers, so the second reviewer's misses
// are less correlated with the first's, and Sonnet's 1M context is the repo-mode
// safety net. The judge is the per-finding call, which is where cost
// concentrates, so it runs on Haiku; judge batches are byte-bounded
// (maxJudgeBatchBytes + maxJudgeDiffBatchBytes, windowed around each finding),
// which keeps them inside Haiku's 200K context. Sharing Haiku with leg A means
// leg B's findings are still judged by a model that did not write them, and
// GX_REVIEW_JUDGE_MODEL gives the judge its own model when wanted.
const (
	defaultReviewModelA  = "claude-haiku-4-5"
	defaultReviewModelB  = "claude-sonnet-4-6"
	defaultJudgeModel    = "claude-haiku-4-5"
	defaultRuleSortModel = "claude-haiku-4-5"
)

// The rule sorter is the cheap tier on purpose in both panels: sorting an
// existing, already-verified finding into a named bucket is a much smaller job
// than finding it or checking it, and it runs off the critical path of both.

// The OpenAI panel mirrors the Claude one tier for tier: the low-cost model as
// leg A, judge, and rule sorter, the mid tier as leg B.
const (
	defaultOpenAIReviewModelA  = "gpt-5.6-luna"
	defaultOpenAIReviewModelB  = "gpt-5.6-terra"
	defaultOpenAIJudgeModel    = "gpt-5.6-luna"
	defaultOpenAIRuleSortModel = "gpt-5.6-luna"
)

// modelPreset is one named panel.
type modelPreset struct {
	// Why is one line for the operator: what this panel is and what it needs.
	Why       string
	ReviewerA string
	ReviewerB string
	Judge     string
	RuleSort  string
}

var modelPresets = map[string]modelPreset{
	"anthropic": {
		Why:       "Claude Haiku 4.5 and Claude Sonnet 4.6 review, Claude Haiku 4.5 verifies; needs ANTHROPIC_API_KEY",
		ReviewerA: defaultReviewModelA,
		ReviewerB: defaultReviewModelB,
		Judge:     defaultJudgeModel,
		RuleSort:  defaultRuleSortModel,
	},
	"openai": {
		Why:       "GPT-5.6 Luna and GPT-5.6 Terra review, GPT-5.6 Luna verifies; needs OPENAI_API_KEY",
		ReviewerA: defaultOpenAIReviewModelA,
		ReviewerB: defaultOpenAIReviewModelB,
		Judge:     defaultOpenAIJudgeModel,
		RuleSort:  defaultOpenAIRuleSortModel,
	},
}

// PresetNames lists the presets in a stable order, for help text and errors.
func PresetNames() []string {
	names := make([]string, 0, len(modelPresets))
	for name := range modelPresets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// activeModelPreset is the panel this review runs: the one GX_REVIEW_MODELS
// names, otherwise the one whose key is set, preferring Claude when both are.
// An unknown GX_REVIEW_MODELS is not an error here — the callers are model
// resolvers deep in the review path, and the CLI validates the name up front
// where a message can reach the user.
func activeModelPreset() modelPreset {
	if preset, ok := modelPresets[strings.ToLower(strings.TrimSpace(os.Getenv("GX_REVIEW_MODELS")))]; ok {
		return preset
	}
	if _, ok := anthropicCredentialFromEnv(); !ok && strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" {
		return modelPresets["openai"]
	}
	return modelPresets["anthropic"]
}

// ValidateModelPresetEnv reports an error naming the valid presets when
// GX_REVIEW_MODELS is set to something that is not one. Called from the CLI
// before a review starts.
func ValidateModelPresetEnv() error {
	name := strings.ToLower(strings.TrimSpace(os.Getenv("GX_REVIEW_MODELS")))
	if name == "" {
		return nil
	}
	if _, ok := modelPresets[name]; ok {
		return nil
	}
	return &unknownPresetError{name: name}
}

type unknownPresetError struct{ name string }

func (e *unknownPresetError) Error() string {
	return "GX_REVIEW_MODELS=" + e.name + " is not a preset; valid: " + strings.Join(PresetNames(), ", ")
}
