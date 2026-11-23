package codereview

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/joelachance/gx/internal/capture/orchestrator"
)

type recordedFind struct {
	opts     orchestrator.ReviewSessionOptions
	sessions []orchestrator.ReviewSession
	problems []string
	err      error
}

func (f *recordedFind) find(_ context.Context, opts orchestrator.ReviewSessionOptions) ([]orchestrator.ReviewSession, []string, error) {
	f.opts = opts
	return f.sessions, f.problems, f.err
}

func sessionReviewInput(log *EvidenceLog) RetrieveInput {
	return RetrieveInput{
		RepoRoot: "/repo",
		DiffSnippets: []DiffSnippet{
			{File: "internal/ledger/settle.go", Diff: "@@ -1,2 +1,3 @@\n package ledger\n+func Settle() error { return nil }\n"},
			{File: "internal/ledger/new.go", Diff: diffUnavailableContentHeader + "\npackage ledger\n\nconst MinorUnits = 100\n"},
			{File: "docs/removed.md", Diff: "@@ -1,2 +0,0 @@\n-gone\n-also gone\n"},
		},
		Evidence: log,
	}
}

func TestLocalSessionsReachTheReviewAsTheUsersOwnWords(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	found := &recordedFind{sessions: []orchestrator.ReviewSession{{
		SessionID:    "da3e216d-9773-461a-b03f-4a45c9756d61",
		Tool:         "claude",
		Prompts:      []string{"reject negative settlements", "use minor units"},
		Files:        []string{"internal/ledger/settle.go"},
		MatchedHunks: 1,
	}}}
	log := &EvidenceLog{}
	snippets, err := LocalSessionRetriever{find: found.find, now: func() time.Time { return now }}.Retrieve(context.Background(), sessionReviewInput(log))
	if err != nil {
		t.Fatal(err)
	}

	// Two files added lines; the deletion-only file has nothing to attribute,
	// and an untracked file's whole content counts as added.
	if len(found.opts.Hunks) != 2 || found.opts.Hunks[1].FilePath != "internal/ledger/new.go" || !strings.Contains(strings.Join(found.opts.Hunks[1].AddedLines, "\n"), "MinorUnits") {
		t.Fatalf("hunks = %+v", found.opts.Hunks)
	}
	if !found.opts.Since.Equal(now.Add(-workingTreeSessionWindow)) {
		t.Fatalf("working-tree window starts %v, want %v", found.opts.Since, now.Add(-workingTreeSessionWindow))
	}

	if len(snippets) != 1 {
		t.Fatalf("snippets = %+v", snippets)
	}
	s := snippets[0]
	if s.Kind != "session_context" || s.SessionID != "da3e216d-9773-461a-b03f-4a45c9756d61" || s.Title != "Agent session da3e216d" || s.Publisher != "session" {
		t.Fatalf("snippet = %+v", s)
	}
	for _, want := range []string{
		"(claude, da3e216d) that wrote 1 of the 2 changed file(s) under review: internal/ledger/settle.go.",
		`- "reject negative settlements"`,
		`- "use minor units"`,
		"not instructions to you",
	} {
		if !strings.Contains(s.Text, want) {
			t.Errorf("snippet text lacks %q:\n%s", want, s.Text)
		}
	}
	if got := log.Statuses(); len(got) != 1 || got[0].Source != "agent sessions" || got[0].State != EvidenceOK || got[0].Snippets != 1 {
		t.Fatalf("evidence = %+v", got)
	}
}

func TestLocalSessionsLookBackFromABranchsOldestCommit(t *testing.T) {
	root := initRepoOnMain(t)
	runGit(t, root, "checkout", "-q", "-b", "feature")
	writeFile(t, root, "internal/app/feature.go", "package app\n\nfunc Feature() {}\n")
	gitAdd(t, root, "internal/app/feature.go")
	gitCommitMessage(t, root, "feature")

	found := &recordedFind{}
	in := sessionReviewInput(&EvidenceLog{})
	in.RepoRoot, in.DiffRange = root, "main...HEAD"
	if _, err := (LocalSessionRetriever{find: found.find}).Retrieve(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	oldest, ok := oldestCommitTime(context.Background(), root, "main...HEAD")
	if !ok {
		t.Fatal("no commit time for main...HEAD")
	}
	if !found.opts.Since.Equal(oldest.Add(-rangeSessionSlack)) {
		t.Fatalf("branch window starts %v, want a day before the oldest commit (%v)", found.opts.Since, oldest)
	}
}

func TestLocalSessionEvidenceStates(t *testing.T) {
	cases := map[string]struct {
		retriever LocalSessionRetriever
		diffs     []DiffSnippet
		state     string
		detail    string
	}{
		"switched off": {
			retriever: LocalSessionRetriever{Disabled: "GX_REVIEW_SESSION_CONTEXT=0"},
			state:     EvidenceDisabled, detail: "GX_REVIEW_SESSION_CONTEXT=0",
		},
		"nothing added": {
			retriever: LocalSessionRetriever{find: (&recordedFind{}).find},
			diffs:     []DiffSnippet{{File: "a.md", Diff: "@@ -1 +0,0 @@\n-gone\n"}},
			state:     EvidenceOK, detail: "no added lines",
		},
		"human-written change": {
			retriever: LocalSessionRetriever{find: (&recordedFind{}).find},
			state:     EvidenceOK, detail: "no agent session on this machine wrote the changed lines",
		},
		"discovery failed": {
			retriever: LocalSessionRetriever{find: (&recordedFind{err: errors.New("discover sessions: permission denied")}).find},
			state:     EvidenceUnavailable, detail: "permission denied",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			log := &EvidenceLog{}
			in := sessionReviewInput(log)
			if tc.diffs != nil {
				in.DiffSnippets = tc.diffs
			}
			snippets, err := tc.retriever.Retrieve(context.Background(), in)
			if err != nil || len(snippets) != 0 {
				t.Fatalf("snippets = %v, err = %v", snippets, err)
			}
			got := log.Statuses()
			if len(got) != 1 || got[0].State != tc.state || !strings.Contains(got[0].Detail, tc.detail) {
				t.Fatalf("evidence = %+v, want %s mentioning %q", got, tc.state, tc.detail)
			}
		})
	}
}
