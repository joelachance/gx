package orchestrator_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joelachance/gx/internal/capture/matcher"
	"github.com/joelachance/gx/internal/capture/orchestrator"
)

const reviewedCode = "func settleLedgerBatch(entries []LedgerEntry) error {\n\tfor _, entry := range entries {\n\t\tif entry.AmountMinor < 0 {\n\t\t\treturn errNegativeSettlement\n\t\t}\n\t}\n\treturn nil\n}"

// claudeLine is one Claude Code transcript line in the shape the real CLI
// writes: a line type, a message with a role, and the cwd that binds it to the
// repository.
func claudeLine(t *testing.T, lineType, role, cwd, session string, ts time.Time, content string) string {
	t.Helper()
	model := ""
	if role == "assistant" {
		model = `,"model":"claude-sonnet-4-6"`
	}
	return fmt.Sprintf(`{"type":%q,"cwd":%q,"sessionId":%q,"timestamp":%q,"message":{"role":%q%s,"content":[%s]}}`,
		lineType, filepath.ToSlash(cwd), session, ts.UTC().Format("2006-01-02T15:04:05.000Z"), role, model, content)
}

func textBlock(text string) string { return fmt.Sprintf(`{"type":"text","text":%q}`, text) }

func writeBlock(path, content string) string {
	return fmt.Sprintf(`{"type":"tool_use","id":"toolu_1","name":"Write","input":{"file_path":%q,"content":%q}}`, filepath.ToSlash(path), content)
}

func writeClaudeSession(t *testing.T, claudeDir, cwd, session string, lines ...string) {
	t.Helper()
	dir := filepath.Join(claudeDir, strings.ReplaceAll(filepath.ToSlash(filepath.Clean(cwd)), "/", "-"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, session+".jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func reviewHunk(file, code string) matcher.HunkRef {
	return matcher.HunkRef{Index: 0, FilePath: file, AddedLines: strings.Split(code, "\n"), CommitTime: time.Now().UnixMilli()}
}

func TestSessionsForReviewFindsTheSessionThatWroteTheLines(t *testing.T) {
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	claudeDir := t.TempDir()
	now := time.Now()
	file := "internal/ledger/settle.go"

	writeClaudeSession(t, claudeDir, repo, "author-session",
		claudeLine(t, "user", "user", repo, "author-session", now.Add(-3*time.Minute),
			textBlock("<system-reminder>You are operating in a worktree.</system-reminder>\nReject negative settlements, the ledger is append-only. Token AKIAIOSFODNN7EXAMPLE is in .env")),
		claudeLine(t, "assistant", "assistant", repo, "author-session", now.Add(-2*time.Minute), textBlock("I'll add a guard in settleLedgerBatch.")),
		claudeLine(t, "api-request-blob", "user", repo, "author-session", now.Add(-2*time.Minute), textBlock("You are Claude Code, Anthropic's official CLI")),
		claudeLine(t, "assistant", "assistant", repo, "author-session", now.Add(-time.Minute), writeBlock(filepath.Join(repo, file), reviewedCode)),
		claudeLine(t, "user", "user", repo, "author-session", now, textBlock("use minor units, not floats")),
	)
	writeClaudeSession(t, claudeDir, repo, "bystander-session",
		claudeLine(t, "user", "user", repo, "bystander-session", now, textBlock("rename the README heading")),
		claudeLine(t, "assistant", "assistant", repo, "bystander-session", now, writeBlock(filepath.Join(repo, "README.md"), "# A completely different heading\n\nNothing to do with settlements at all.")),
	)

	sessions, problems, err := orchestrator.SessionsForReview(context.Background(), orchestrator.ReviewSessionOptions{
		RepoRoot:    repo,
		Hunks:       []matcher.HunkRef{reviewHunk(file, reviewedCode)},
		Since:       now.Add(-time.Hour),
		Until:       now.Add(time.Hour),
		HomeDir:     t.TempDir(),
		ClaudeDir:   claudeDir,
		CodexDir:    filepath.Join(t.TempDir(), "codex"),
		CursorVSCDB: filepath.Join(t.TempDir(), "state.vscdb"),
	})
	if err != nil {
		t.Fatalf("SessionsForReview: %v (problems %v)", err, problems)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v, want only the session that wrote the lines", sessions)
	}
	got := sessions[0]
	if got.SessionID != "author-session" || got.Tool != "claude" || got.MatchedHunks != 1 || strings.Join(got.Files, ",") != file {
		t.Fatalf("session = %+v", got)
	}
	// Only the user's two turns: no system reminder, no assistant text, and no
	// API-request copy, with the secret redacted before it reaches a model.
	if len(got.Prompts) != 2 ||
		!strings.HasPrefix(got.Prompts[0], "Reject negative settlements, the ledger is append-only. Token [REDACTED:aws-key:") ||
		strings.Contains(got.Prompts[0], "AKIAIOSFODNN7EXAMPLE") ||
		got.Prompts[1] != "use minor units, not floats" {
		t.Fatalf("prompts = %q", got.Prompts)
	}
}

func TestSessionsForReviewWithNothingToAttribute(t *testing.T) {
	sessions, _, err := orchestrator.SessionsForReview(context.Background(), orchestrator.ReviewSessionOptions{RepoRoot: t.TempDir()})
	if err != nil || sessions != nil {
		t.Fatalf("no hunks: sessions = %v, err = %v; want nothing and no error", sessions, err)
	}
}
