package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/satoricorp/gx/internal/capture/orchestrator"
	"github.com/satoricorp/gx/internal/storage"
	"github.com/satoricorp/gx/internal/vcs"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "README.md")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
}

func TestRunPushRespectsPauseFlag(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	initGitRepo(t, repo)

	gxHome := filepath.Join(home, ".gx")
	t.Setenv("GX_HOME", gxHome)
	if err := os.MkdirAll(gxHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gxHome, "pause-capture"), []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	outcome, err := RunPush(context.Background(), PushOptions{
		RepoRoot: repo,
		HomeDir:  home,
	})
	if err != nil {
		t.Fatalf("RunPush() error = %v", err)
	}
	if outcome.Result.RefRange != "" || outcome.Result.EligibleHunks != 0 {
		t.Fatalf("RunPush() = %+v, want empty result when paused", outcome)
	}
}

func TestParseRefRangeSingleSHALeavesBaseForOrchestrator(t *testing.T) {
	base, head := parseRefRange("abc123", "", "")
	if base != "" || head != "abc123" {
		t.Fatalf("parseRefRange() = (%q, %q), want empty base and abc123 head", base, head)
	}
}

func TestRunPushRecoversMissingRevision(t *testing.T) {
	repoRoot := t.TempDir()
	gxHome := t.TempDir()
	t.Setenv("GX_HOME", gxHome)
	t.Setenv("GX_DISABLE_BACKGROUND_WORKERS", "1")
	initGitRepo(t, repoRoot)
	service := vcs.NewService()
	if _, err := service.InitAtPath(context.Background(), repoRoot, vcs.InitOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInHookTest(t, repoRoot, "add", "feature.txt")
	runGitInHookTest(t, repoRoot, "commit", "--no-verify", "-m", "feature")
	commitOID := strings.TrimSpace(gitOutputInHookTest(t, repoRoot, "rev-parse", "HEAD"))

	outcome, err := RunPush(context.Background(), PushOptions{
		RepoRoot:    repoRoot,
		RefRange:    commitOID,
		SkipCapture: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.RecoveryError != "" {
		t.Fatalf("recovery error = %q", outcome.RecoveryError)
	}
	// Nothing has been pushed yet, so the initial commit is recorded too.
	if outcome.RecoveredCommits != 2 {
		t.Fatalf("recovered commits = %d, want 2", outcome.RecoveredCommits)
	}
	repo, err := service.ResolveGxRepoAtPath(context.Background(), repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := storage.NewStore(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	repoRow, err := store.FindRepoByIdentity(context.Background(), repo.GitCommonDir, repo.RootPath)
	if err != nil || repoRow == nil {
		t.Fatalf("FindRepoByIdentity() = %+v, %v", repoRow, err)
	}
	change, err := store.FindChangeByCommitID(context.Background(), repoRow.ID, commitOID)
	if err != nil || change == nil {
		t.Fatalf("FindChangeByCommitID() = %+v, %v", change, err)
	}
}

func TestRunPushReportsRevisionScanFailure(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("GX_HOME", t.TempDir())
	initGitRepo(t, repoRoot)

	outcome, err := RunPush(context.Background(), PushOptions{
		RepoRoot:    repoRoot,
		RefRange:    "missing-ref..HEAD",
		SkipCapture: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(outcome.RecoveryError, "list pushed commits") {
		t.Fatalf("recovery error = %q, want revision scan failure", outcome.RecoveryError)
	}
}

func runGitInHookTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitOutputInHookTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// TestRunPushKeepsResultFromPartialCaptureFailure covers the orchestrator's
// partial-failure contract: it returns a populated Result alongside its error
// because the rows it already wrote are real. Dropping that Result made a run
// that staged an extract and then failed look identical to a run that staged
// nothing, and left the staged rows unmarked and therefore unuploadable.
func TestRunPushKeepsResultFromPartialCaptureFailure(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("GX_HOME", t.TempDir())
	t.Setenv("GX_DISABLE_BACKGROUND_WORKERS", "1")
	initGitRepo(t, repoRoot)

	ctx := context.Background()
	stager, err := storage.OpenCaptureStager(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Rows with no revision id at all, exactly as a plain-Git commit stages them.
	if err := stager.StageExtract(ctx, storage.StagedExtract{
		ID: "ext-partial", RepoRoot: repoRoot, RefRange: "HEAD~1..HEAD", PayloadJSON: []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := stager.StageSession(ctx, storage.StagedSession{
		ID: "sess-partial", SessionID: "s1", Tool: "claude", PayloadJSON: []byte(`{"sessionID":"s1"}`),
	}); err != nil {
		t.Fatal(err)
	}

	restore := runCapture
	runCapture = func(context.Context, orchestrator.RunOptions) (orchestrator.Result, error) {
		return orchestrator.Result{
			RefRange:         "HEAD~1..HEAD",
			StagedExtractID:  "ext-partial",
			StagedExtractIDs: []string{"ext-partial"},
			StagedSessionIDs: []string{"sess-partial"},
			StagedSessions:   1,
			EligibleHunks:    3,
		}, fmt.Errorf("stage session s2: database is locked")
	}
	t.Cleanup(func() { runCapture = restore })

	outcome, err := RunPush(ctx, PushOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("RunPush() error = %v, want nil so the hook never blocks", err)
	}
	if outcome.CaptureError == "" {
		t.Fatal("CaptureError = \"\", want the partial failure reported")
	}
	if outcome.Result.StagedSessions != 1 || outcome.Result.EligibleHunks != 3 {
		t.Fatalf("Result = %+v, want the work the failed run actually produced", outcome.Result)
	}
	if len(outcome.Result.StagedExtractIDs) != 1 || len(outcome.Result.StagedSessionIDs) != 1 {
		t.Fatalf("staged ids = %v / %v, want the ids of the rows that reached SQLite",
			outcome.Result.StagedExtractIDs, outcome.Result.StagedSessionIDs)
	}

}

// TestRunPushNamesEveryReasonItDidNothing is the anti-ambiguity test. A paused
// capture and a repo opted out with `git config gx.enabled false` both returned
// the same zero outcome as a genuine error, so all three printed the identical
// `capture staged extract= sessions=0` line and only one of them was a problem.
func TestRunPushNamesEveryReasonItDidNothing(t *testing.T) {
	t.Run("paused", func(t *testing.T) {
		home := t.TempDir()
		repo := t.TempDir()
		initGitRepo(t, repo)
		gxHome := filepath.Join(home, ".gx")
		t.Setenv("GX_HOME", gxHome)
		if err := os.MkdirAll(gxHome, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(gxHome, "pause-capture"), []byte("1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		outcome, err := RunPush(context.Background(), PushOptions{RepoRoot: repo, HomeDir: home})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(outcome.SkipReason, "paused") {
			t.Fatalf("SkipReason = %q, want it to name the pause", outcome.SkipReason)
		}
		if outcome.CaptureError != "" {
			t.Fatalf("CaptureError = %q, want a deliberate pause not reported as a failure", outcome.CaptureError)
		}
	})

	t.Run("repo opted out", func(t *testing.T) {
		home := t.TempDir()
		repo := t.TempDir()
		initGitRepo(t, repo)
		t.Setenv("GX_HOME", t.TempDir())
		runGitInHookTest(t, repo, "config", "gx.enabled", "false")

		outcome, err := RunPush(context.Background(), PushOptions{RepoRoot: repo, HomeDir: home})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(outcome.SkipReason, "gx.enabled") {
			t.Fatalf("SkipReason = %q, want it to name the opt-out", outcome.SkipReason)
		}
	})
}
