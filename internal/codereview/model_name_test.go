package codereview

import "testing"

func TestFriendlyModelName(t *testing.T) {
	cases := map[string]string{
		"claude-haiku-4-5-20251001":   "Claude Haiku 4.5",
		"claude-sonnet-4-6":           "Claude Sonnet 4.6",
		"claude-opus-4-7":             "Claude Opus 4.7",
		"anthropic:claude-sonnet-5-5": "Claude Sonnet 5.5",
		"claude-haiku-4-5":            "Claude Haiku 4.5",
		"":                            "",
		"some-other-model":            "some-other-model",
	}
	for in, want := range cases {
		if got := friendlyModelName(in); got != want {
			t.Errorf("friendlyModelName(%q) = %q, want %q", in, got, want)
		}
	}
}
