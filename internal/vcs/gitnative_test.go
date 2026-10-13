package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/satoricorp/gx/internal/storage"
)

func TestGenerateRevisionID(t *testing.T) {
	id, err := GenerateRevisionID()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidRevisionID(id) {
		t.Fatalf("GenerateRevisionID() = %q, invalid format", id)
	}
	second, err := GenerateRevisionID()
	if err != nil {
		t.Fatal(err)
	}
	if id == second {
		t.Fatalf("GenerateRevisionID() returned duplicate ids")
	}
}

func TestParsePostRewriteMappings(t *testing.T) {
	mappings := ParsePostRewriteMappings("aaa111 bbb222\nccc333 ddd444\n")
	if len(mappings) != 2 {
		t.Fatalf("len = %d, want 2", len(mappings))
	}
	if mappings[0].OldOID != "aaa111" || mappings[0].NewOID != "bbb222" {
		t.Fatalf("first mapping = %+v", mappings[0])
	}
}

func TestNormalizeGitCommonDir(t *testing.T) {
	worktree := "/tmp/repo"
	got := NormalizeGitCommonDir(worktree, ".git")
	want := "/tmp/repo/.git"
	if got != want {
		t.Fatalf("NormalizeGitCommonDir() = %q, want %q", got, want)
	}
}

func TestPendingContextIsolatedByLinkedWorktree(t *testing.T) {
	repo := initGitNativeTestRepo(t)
	worktree := filepath.Join(t.TempDir(), "linked")
	runGitNativeTestGit(t, repo, "worktree", "add", "-b", "linked", worktree)
	service := NewService()
	mainPaths, err := service.ResolveGitPaths(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	linkedPaths, err := service.ResolveGitPaths(context.Background(), worktree)
	if err != nil {
		t.Fatal(err)
	}
	if mainPaths.CommonDir != linkedPaths.CommonDir {
		t.Fatalf("common dirs differ: %q != %q", mainPaths.CommonDir, linkedPaths.CommonDir)
	}
	if mainPaths.GitDir == linkedPaths.GitDir {
		t.Fatalf("git dirs unexpectedly equal: %q", mainPaths.GitDir)
	}
	mainPending := PendingCommitContext{WorktreeRoot: repo}
	linkedPending := PendingCommitContext{WorktreeRoot: worktree}
	if err := writePendingCommitContext(mainPaths.GitDir, mainPending); err != nil {
		t.Fatal(err)
	}
	if err := writePendingCommitContext(linkedPaths.GitDir, linkedPending); err != nil {
		t.Fatal(err)
	}
	mainGot, err := readPendingCommitContext(mainPaths.GitDir)
	if err != nil {
		t.Fatal(err)
	}
	linkedGot, err := readPendingCommitContext(linkedPaths.GitDir)
	if err != nil {
		t.Fatal(err)
	}
	if mainGot.WorktreeRoot != repo || linkedGot.WorktreeRoot != worktree {
		t.Fatalf("pending contexts crossed: main=%q linked=%q", mainGot.WorktreeRoot, linkedGot.WorktreeRoot)
	}
	info, err := os.Stat(pendingCommitContextPath(mainPaths.GitDir))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("pending context permissions = %o, want 600", info.Mode().Perm())
	}
}

func initGitNativeTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitNativeTestGit(t, repo, "init", "-b", "main")
	runGitNativeTestGit(t, repo, "config", "user.name", "gx Test")
	runGitNativeTestGit(t, repo, "config", "user.email", "gx@example.com")
	writeGitNativeTestFile(t, repo, "README.md", "base\n")
	runGitNativeTestGit(t, repo, "add", "README.md")
	runGitNativeTestGit(t, repo, "commit", "-m", "base")
	return repo
}

func writeGitNativeTestFile(t *testing.T, repo, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGitNativeTestGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// recordedChangeForCommit reads the change ~/.gx recorded for a commit.
func recordedChangeForCommit(t *testing.T, repo RepoInfo, commitOID string) *storage.Change {
	t.Helper()
	store, err := openStore(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	repoRow, err := store.FindRepoByIdentity(context.Background(), repo.GitCommonDir, repo.RootPath)
	if err != nil || repoRow == nil {
		t.Fatalf("FindRepoByIdentity() = %+v, %v", repoRow, err)
	}
	change, err := store.FindChangeByCommitID(context.Background(), repoRow.ID, commitOID)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

// A plain `git commit` is recorded as a revision in ~/.gx, and gx leaves its
// message exactly as written.
func TestPostCommitHookRecordsPlainGitCommit(t *testing.T) {
	repoRoot := initGitNativeTestRepo(t)
	t.Setenv("GX_HOME", t.TempDir())
	service := NewService()
	if _, err := service.InitAtPath(context.Background(), repoRoot, InitOptions{}); err != nil {
		t.Fatal(err)
	}
	writeGitNativeTestFile(t, repoRoot, "feature.txt", "feature\n")
	runGitNativeTestGit(t, repoRoot, "add", "feature.txt")
	runGitNativeTestGit(t, repoRoot, "commit", "-m", "add feature")

	// The post-commit hook runs without a pending context file; it must fall
	// back to the repo it was handed and still record the revision.
	if err := service.RunPostCommitHook(context.Background(), repoRoot); err != nil {
		t.Fatal(err)
	}

	if message := strings.TrimSpace(runGitNativeTestGit(t, repoRoot, "log", "-1", "--format=%B")); message != "add feature" {
		t.Fatalf("commit message = %q, want it untouched", message)
	}
	repo, err := service.ResolveGxRepoAtPath(context.Background(), repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	headOID := strings.TrimSpace(runGitNativeTestGit(t, repoRoot, "rev-parse", "HEAD"))
	change := recordedChangeForCommit(t, repo, headOID)
	if change == nil || !ValidRevisionID(change.JJChangeID) {
		t.Fatalf("recorded change for HEAD = %+v, want one with a valid revision id", change)
	}
}

// An amend keeps its revision. Git runs post-commit for the new commit before
// post-rewrite maps the old SHA to it, so post-commit must not record the
// amended commit as a revision of its own.
func TestAmendKeepsTheRevision(t *testing.T) {
	repoRoot := initGitNativeTestRepo(t)
	t.Setenv("GX_HOME", t.TempDir())
	service := NewService()
	ctx := context.Background()
	if _, err := service.InitAtPath(ctx, repoRoot, InitOptions{}); err != nil {
		t.Fatal(err)
	}
	repo, err := service.ResolveGxRepoAtPath(ctx, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	writeGitNativeTestFile(t, repoRoot, "feature.txt", "one\n")
	runGitNativeTestGit(t, repoRoot, "add", "feature.txt")
	runGitNativeTestGit(t, repoRoot, "commit", "-m", "feature")
	if err := service.RunPostCommitHook(ctx, repoRoot); err != nil {
		t.Fatal(err)
	}
	oldOID := strings.TrimSpace(runGitNativeTestGit(t, repoRoot, "rev-parse", "HEAD"))
	original := recordedChangeForCommit(t, repo, oldOID)
	if original == nil {
		t.Fatal("first commit was not recorded")
	}

	writeGitNativeTestFile(t, repoRoot, "feature.txt", "two\n")
	runGitNativeTestGit(t, repoRoot, "add", "feature.txt")
	runGitNativeTestGit(t, repoRoot, "commit", "--amend", "-m", "feature")
	newOID := strings.TrimSpace(runGitNativeTestGit(t, repoRoot, "rev-parse", "HEAD"))
	if err := service.RunPostCommitHook(ctx, repoRoot); err != nil {
		t.Fatal(err)
	}
	if early := recordedChangeForCommit(t, repo, newOID); early != nil {
		t.Fatalf("post-commit recorded the amended commit as revision %s; post-rewrite owns rewrites", early.JJChangeID)
	}
	if err := service.RunPostRewriteHook(ctx, repoRoot, []CommitOIDMapping{{OldOID: oldOID, NewOID: newOID}}); err != nil {
		t.Fatal(err)
	}
	amended := recordedChangeForCommit(t, repo, newOID)
	if amended == nil || amended.JJChangeID != original.JJChangeID {
		t.Fatalf("amended commit's revision = %+v, want the original %s", amended, original.JJChangeID)
	}
}

// A commit made before gx stopped writing trailers keeps the revision its
// trailer names, so existing history stays linked.
func TestRecordGitCommitKeepsALegacyTrailerRevision(t *testing.T) {
	repoRoot := initGitNativeTestRepo(t)
	t.Setenv("GX_HOME", t.TempDir())
	service := NewService()
	ctx := context.Background()
	if _, err := service.InitAtPath(ctx, repoRoot, InitOptions{}); err != nil {
		t.Fatal(err)
	}
	repo, err := service.ResolveGxRepoAtPath(ctx, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	revisionID, err := GenerateRevisionID()
	if err != nil {
		t.Fatal(err)
	}
	writeGitNativeTestFile(t, repoRoot, "feature.txt", "one\n")
	runGitNativeTestGit(t, repoRoot, "add", "feature.txt")
	runGitNativeTestGit(t, repoRoot, "commit", "-m", "feature\n\n"+legacyRevisionTrailerPrefix+revisionID)
	headOID := strings.TrimSpace(runGitNativeTestGit(t, repoRoot, "rev-parse", "HEAD"))
	if _, err := service.RecordGitCommit(ctx, repo, headOID, PendingCommitContext{WorktreeRoot: repo.RootPath, GitCommonDir: repo.GitCommonDir}); err != nil {
		t.Fatal(err)
	}
	if change := recordedChangeForCommit(t, repo, headOID); change == nil || change.JJChangeID != revisionID {
		t.Fatalf("recorded change = %+v, want the trailer's revision %s", change, revisionID)
	}
}
