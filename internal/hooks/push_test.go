package hooks_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/satoricorp/gx/internal/hooks"
	"github.com/satoricorp/gx/internal/storage"
	"github.com/satoricorp/gx/internal/vcs"
)

func TestRunPushAlwaysSucceedsWithNetworkUnavailable(t *testing.T) {
	repo := initPushHookRepo(t)
	base := gitRev(t, repo, "HEAD~1")
	head := gitRev(t, repo, "HEAD")
	refRange := base + ".." + head

	gxHome := t.TempDir()
	t.Setenv("GX_HOME", gxHome)

	outcome, err := hooks.RunPush(context.Background(), hooks.PushOptions{
		RepoRoot:    repo,
		RefRange:    refRange,
		SkipCapture: true,
	})
	if err != nil {
		t.Fatalf("RunPush() error = %v, want nil so hook never blocks", err)
	}
	if len(outcome.RevisionIDs) != 1 || outcome.RevisionIDs[0] != "revpushaaaaa" {
		t.Fatalf("RevisionIDs = %v, want [revpushaaaaa]", outcome.RevisionIDs)
	}
}

func TestRunPushIdempotentCaptureRows(t *testing.T) {
	ctx := context.Background()
	gxHome := t.TempDir()
	t.Setenv("GX_HOME", gxHome)
	t.Setenv("GX_DISABLE_BACKGROUND_WORKERS", "1")

	stager, err := storage.OpenCaptureStager(ctx)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"sessionID":"s1"}`)
	row := storage.StagedSession{
		SessionID:   "s1",
		Tool:        "cursor",
		PayloadJSON: payload,
		RevisionID:  "revpushaaaaa",
	}
	if err := stager.StageSession(ctx, row); err != nil {
		t.Fatal(err)
	}
	firstID := storage.CaptureRowID(row.RevisionID, storage.PayloadContentHash(payload))
	if err := stager.StageSession(ctx, row); err != nil {
		t.Fatal(err)
	}
	sessions, err := stager.PendingSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != firstID {
		t.Fatalf("sessions = %+v, want one idempotent row %s", sessions, firstID)
	}
}

func initPushHookRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GX_DISABLE_BACKGROUND_WORKERS", "1")
	repo := t.TempDir()
	for _, args := range [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "feature.txt")
	message := "feature\n\n" + vcs.RevisionTrailerLine("revpushaaaaa")
	runGit(t, repo, "commit", "-m", message)
	return repo
}

func gitRev(t *testing.T, repo, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "rev-parse", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse %s: %v: %s", ref, err, out)
	}
	return strings.TrimSpace(string(out))
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}
