package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/joelachance/gx/internal/capture"
	"github.com/joelachance/gx/internal/capture/matcher"
	"github.com/joelachance/gx/internal/capture/parsers"
	"github.com/joelachance/gx/internal/capture/parsers/claude"
	"github.com/joelachance/gx/internal/capture/parsers/codex"
	cursorparser "github.com/joelachance/gx/internal/capture/parsers/cursor"
	"github.com/joelachance/gx/internal/capture/redact"
)

// ReviewSessionOptions says which added lines a review wants the authors of,
// and the window to look for their sessions in.
type ReviewSessionOptions struct {
	RepoRoot string
	Hunks    []matcher.HunkRef
	Since    time.Time
	Until    time.Time

	// Overrides for tests; empty means the tools' real locations.
	HomeDir     string
	ClaudeDir   string
	CodexDir    string
	CursorVSCDB string
}

// ReviewSession is one agent session that wrote some of the lines under
// review: what the user asked it, in their own words, and which files its
// matched edits landed in.
type ReviewSession struct {
	SessionID    string
	Tool         string
	Prompts      []string
	Files        []string
	MatchedHunks int
}

// maxReviewPrompts bounds the prompts kept per session. The first ones carry
// the original ask, the last ones the corrections that shaped the final code.
const (
	maxReviewPrompts  = 8
	keptFirstPrompts  = 2
	reviewPromptLimit = 512
)

// SessionsForReview reads the agents' transcripts the way pre-push capture
// does and returns the sessions whose edits wrote the given lines, most
// matched hunks first. It reads live transcripts rather than ~/.gx because a
// review usually runs before the push that would capture them.
//
// Problems are discovery legs that could not be read; they are returned for
// the caller to report, not as an error, so one unreadable tool does not hide
// the others.
func SessionsForReview(ctx context.Context, opts ReviewSessionOptions) ([]ReviewSession, []string, error) {
	if len(opts.Hunks) == 0 {
		return nil, nil, nil
	}
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("repo path: %w", err)
	}
	home := opts.HomeDir
	if home == "" {
		if home, err = os.UserHomeDir(); err != nil {
			return nil, nil, fmt.Errorf("home dir: %w", err)
		}
	}
	tools := normalizeToolsList(nil)
	discovery, err := parsers.Discover(parsers.DiscoverOptions{
		HomeDir:     home,
		RepoRoot:    repoRoot,
		Since:       opts.Since,
		Until:       opts.Until,
		ClaudeDir:   opts.ClaudeDir,
		CodexDir:    opts.CodexDir,
		CursorVSCDB: opts.CursorVSCDB,
		Tools:       tools,
	})
	if err != nil {
		return nil, discovery.ProblemStrings(), fmt.Errorf("discover sessions: %w", err)
	}
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}
	active := buildToolParsers(tools,
		&claude.Parser{},
		&codex.Parser{},
		&cursorparser.Parser{Since: opts.Since, Until: opts.Until},
	)
	perSource, err := parsers.ParseAllPerSource(discovery.Sessions, repoRoot, active)
	if err != nil {
		return nil, discovery.ProblemStrings(), fmt.Errorf("parse sessions: %w", err)
	}
	var events []capture.SessionEvent
	for _, source := range perSource {
		events = append(events, source.Events...)
	}

	result := matcher.Match(events, opts.Hunks, matcher.DefaultConfig())
	type found struct {
		tool  string
		hunks map[int]struct{}
		files map[string]struct{}
	}
	bySession := map[string]*found{}
	for _, outcome := range result.Outcomes {
		if outcome.Tier != matcher.TierExact && outcome.Tier != matcher.TierFuzzy && outcome.Tier != matcher.TierCommand {
			continue
		}
		if outcome.EventIndex < 0 || outcome.EventIndex >= len(events) || outcome.HunkIndex < 0 || outcome.HunkIndex >= len(opts.Hunks) {
			continue
		}
		ev := events[outcome.EventIndex]
		f := bySession[ev.SessionID]
		if f == nil {
			f = &found{tool: ev.Tool, hunks: map[int]struct{}{}, files: map[string]struct{}{}}
			bySession[ev.SessionID] = f
		}
		f.hunks[outcome.HunkIndex] = struct{}{}
		f.files[opts.Hunks[outcome.HunkIndex].FilePath] = struct{}{}
	}

	prompts := map[string][]string{}
	sortedEvents := append([]capture.SessionEvent(nil), events...)
	sort.SliceStable(sortedEvents, func(i, j int) bool { return sortedEvents[i].TS < sortedEvents[j].TS })
	for _, ev := range sortedEvents {
		if _, ok := bySession[ev.SessionID]; !ok || ev.Kind != capture.KindMessage || ev.Role != "user" {
			continue
		}
		if prompt := cleanReviewPrompt(ev.PromptContext); prompt != "" {
			prompts[ev.SessionID] = append(prompts[ev.SessionID], prompt)
		}
	}

	out := make([]ReviewSession, 0, len(bySession))
	for id, f := range bySession {
		out = append(out, ReviewSession{
			SessionID:    id,
			Tool:         f.tool,
			Prompts:      boundPrompts(prompts[id]),
			Files:        sortedKeys(f.files),
			MatchedHunks: len(f.hunks),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MatchedHunks != out[j].MatchedHunks {
			return out[i].MatchedHunks > out[j].MatchedHunks
		}
		return out[i].SessionID < out[j].SessionID
	})
	return out, discovery.ProblemStrings(), nil
}

// injectedBlock matches the markup the agent harness adds to a user turn
// (system reminders, slash-command wrappers); none of it is the user's words.
var injectedBlock = regexp.MustCompile(`(?s)<(system-reminder|command-[a-z-]+|local-command-[a-z-]+)>.*?</(system-reminder|command-[a-z-]+|local-command-[a-z-]+)>`)

func cleanReviewPrompt(text string) string {
	text = injectedBlock.ReplaceAllString(text, "")
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}
	text = redact.Redact(text)
	if len(text) > reviewPromptLimit {
		text = text[:reviewPromptLimit] + "…"
	}
	return text
}

func boundPrompts(prompts []string) []string {
	if len(prompts) <= maxReviewPrompts {
		return prompts
	}
	out := append([]string(nil), prompts[:keptFirstPrompts]...)
	return append(out, prompts[len(prompts)-(maxReviewPrompts-keptFirstPrompts):]...)
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
