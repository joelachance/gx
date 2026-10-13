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

func TestInstallLifecycleHooksUsesGitHooksDir(t *testing.T) {
	repo := t.TempDir()
	runGitInRepo(t, repo, "init")
	runGitInRepo(t, repo, "config", "user.email", "dev@example.com")
	runGitInRepo(t, repo, "config", "user.name", "Dev")
	gxPath := buildGxBinary(t)
	if err := hooks.Install(hooks.InstallOptions{RepoRoot: repo, GxPath: gxPath}); err != nil {
		t.Fatal(err)
	}
	hooksDir, err := hooks.ResolveHooksDir(t.Context(), repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"post-commit", "post-rewrite", "pre-push"} {
		data, err := os.ReadFile(filepath.Join(hooksDir, name))
		if err != nil {
			t.Fatalf("read %s hook: %v", name, err)
		}
		if !strings.Contains(string(data), "# gx lifecycle hooks") {
			t.Fatalf("%s hook missing marker:\n%s", name, data)
		}
	}
}

// Hooks installed by an older gx still call prepare-commit-msg, and without
// `|| true`, so it must succeed and leave the message as the author wrote it.
func TestPrepareCommitMsgLeavesTheMessageAlone(t *testing.T) {
	repo := t.TempDir()
	runGitInRepo(t, repo, "init")
	messagePath := filepath.Join(repo, "COMMIT_EDITMSG")
	if err := os.WriteFile(messagePath, []byte("add feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := hooks.PrepareCommitMsg(hooks.PrepareCommitMsgOptions{
		RepoRoot:    repo,
		MessagePath: messagePath,
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(messagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "add feature\n" {
		t.Fatalf("message = %q, want it untouched", string(data))
	}
}

func TestPlainGitCommitWithInstalledHooks(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("GX_HOME", t.TempDir())
	runGitInRepo(t, repo, "init", "-b", "main")
	runGitInRepo(t, repo, "config", "user.email", "dev@example.com")
	runGitInRepo(t, repo, "config", "user.name", "Dev")
	gxPath := buildGxBinary(t)
	if err := hooks.Install(hooks.InstallOptions{RepoRoot: repo, GxPath: gxPath}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInRepo(t, repo, "add", "feature.txt")
	runGitInRepo(t, repo, "commit", "-m", "add feature")
	if message := gitOutput(t, repo, "log", "-1", "--format=%B"); strings.TrimSpace(message) != "add feature" {
		t.Fatalf("commit message = %q, want it untouched", message)
	}
}

func TestInstallRefusesForeignHook(t *testing.T) {
	repo := t.TempDir()
	runGitInRepo(t, repo, "init")
	hooksDir, err := hooks.ResolveHooksDir(t.Context(), repo)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho foreign\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	err = hooks.Install(hooks.InstallOptions{RepoRoot: repo, GxPath: "/bin/gx"})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("Install() error = %v, want refusal", err)
	}
}

func TestInstallUpgradesLegacyGxPrePushHook(t *testing.T) {
	repo := t.TempDir()
	runGitInRepo(t, repo, "init")
	hooksDir, err := hooks.ResolveHooksDir(t.Context(), repo)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(hooksDir, "pre-push")
	legacy := "#!/bin/sh\n# gx capture pre-push hook\ntt capture push || true\n"
	if err := os.WriteFile(path, []byte(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := hooks.Install(hooks.InstallOptions{RepoRoot: repo, GxPath: "/bin/gx"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# gx lifecycle hooks") || strings.Contains(string(data), "# gx capture pre-push hook") {
		t.Fatalf("legacy pre-push hook was not upgraded:\n%s", data)
	}
}

func TestPostLifecycleHooksWarnWithoutBlocking(t *testing.T) {
	repo := t.TempDir()
	runGitInRepo(t, repo, "init")
	if err := hooks.Install(hooks.InstallOptions{RepoRoot: repo, GxPath: "/missing/gx"}); err != nil {
		t.Fatal(err)
	}
	hooksDir, err := hooks.ResolveHooksDir(t.Context(), repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"post-commit", "post-rewrite"} {
		data, err := os.ReadFile(filepath.Join(hooksDir, name))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if !strings.Contains(content, "failed") || !strings.Contains(content, "exit 0") {
			t.Fatalf("%s does not warn and continue:\n%s", name, content)
		}
	}
}

func TestBootstrapFromLinkedWorktree(t *testing.T) {
	primary := t.TempDir()
	gxHome := t.TempDir()
	t.Setenv("GX_HOME", gxHome)
	t.Setenv("GX_DISABLE_BACKGROUND_WORKERS", "1")
	runGitInRepo(t, primary, "init", "-b", "main")
	runGitInRepo(t, primary, "config", "user.email", "dev@example.com")
	runGitInRepo(t, primary, "config", "user.name", "Dev")
	if err := os.WriteFile(filepath.Join(primary, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInRepo(t, primary, "add", "README.md")
	runGitInRepo(t, primary, "commit", "-m", "base")

	linked := filepath.Join(t.TempDir(), "linked")
	runGitInRepo(t, primary, "worktree", "add", "-b", "linked", linked)
	if err := os.WriteFile(filepath.Join(primary, "primary.txt"), []byte("primary staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInRepo(t, primary, "add", "primary.txt")
	primaryHead := strings.TrimSpace(gitOutput(t, primary, "rev-parse", "HEAD"))
	primaryIndex := gitOutput(t, primary, "diff", "--cached", "--binary")

	if err := os.WriteFile(filepath.Join(linked, "linked.txt"), []byte("linked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInRepo(t, linked, "add", "linked.txt")
	// Any gx command run inside the linked worktree bootstraps the repo and
	// installs the lifecycle hooks; the commit itself is plain git.
	gxPath := buildGxBinary(t)
	cmd := exec.Command(gxPath, "doctor")
	cmd.Dir = linked
	cmd.Env = append(os.Environ(), "GX_HOME="+gxHome, "GX_DISABLE_BACKGROUND_WORKERS=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gx doctor: %v\n%s", err, out)
	}

	if !hooks.IsInstalled(linked) {
		t.Fatal("bootstrap did not install lifecycle hooks")
	}
	runGitInRepo(t, linked, "commit", "-m", "linked worktree commit")
	if message := gitOutput(t, linked, "log", "-1", "--format=%B"); strings.TrimSpace(message) != "linked worktree commit" {
		t.Fatalf("linked commit message = %q, want it untouched", message)
	}
	linkedOID := strings.TrimSpace(gitOutput(t, linked, "rev-parse", "HEAD"))

	service := vcs.NewService()
	repo, err := service.ResolveGxRepoAtPath(context.Background(), linked)
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
	initialized, err := store.IsInitializedRepo(context.Background(), repoRow.RootPath)
	if err != nil || !initialized {
		t.Fatalf("IsInitializedRepo() = %v, %v", initialized, err)
	}
	change, err := store.FindChangeByCommitID(context.Background(), repoRow.ID, linkedOID)
	if err != nil || change == nil || !vcs.ValidRevisionID(change.JJChangeID) {
		t.Fatalf("FindChangeByCommitID() = %+v, %v", change, err)
	}

	if err := os.WriteFile(filepath.Join(linked, "plain.txt"), []byte("plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInRepo(t, linked, "add", "plain.txt")
	runGitInRepo(t, linked, "commit", "-m", "plain linked commit")
	plainOID := strings.TrimSpace(gitOutput(t, linked, "rev-parse", "HEAD"))
	plainChange, err := store.FindChangeByCommitID(context.Background(), repoRow.ID, plainOID)
	if err != nil || plainChange == nil {
		t.Fatalf("plain commit was not recorded: %+v, %v", plainChange, err)
	}
	plainRevision := plainChange.JJChangeID

	if err := os.WriteFile(filepath.Join(linked, "plain.txt"), []byte("plain amended\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInRepo(t, linked, "add", "plain.txt")
	runGitInRepo(t, linked, "commit", "--amend", "--no-edit")
	amendedOID := strings.TrimSpace(gitOutput(t, linked, "rev-parse", "HEAD"))
	if amendedOID == plainOID {
		t.Fatal("amend did not change the Git commit OID")
	}
	amendedChange, err := store.FindChangeByJJChangeID(context.Background(), repoRow.ID, plainRevision)
	if err != nil || amendedChange == nil || amendedChange.CurrentCommitID != amendedOID {
		t.Fatalf("amended change = %+v, %v; want revision %s at commit %s", amendedChange, err, plainRevision, amendedOID)
	}

	hooksDir, err := hooks.ResolveHooksDir(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"post-commit", "post-rewrite"} {
		if err := os.Rename(filepath.Join(hooksDir, name), filepath.Join(hooksDir, name+".disabled")); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(linked, "plain.txt"), []byte("plain missed rewrite\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInRepo(t, linked, "add", "plain.txt")
	runGitInRepo(t, linked, "commit", "--amend", "--no-edit")
	missedOID := strings.TrimSpace(gitOutput(t, linked, "rev-parse", "HEAD"))
	staleChange, err := store.FindChangeByJJChangeID(context.Background(), repoRow.ID, plainRevision)
	if err != nil || staleChange == nil || staleChange.CurrentCommitID != amendedOID {
		t.Fatalf("change was not stale before recovery: %+v, %v", staleChange, err)
	}

	outcome, err := hooks.RunPush(context.Background(), hooks.PushOptions{
		RepoRoot:    linked,
		RefRange:    missedOID,
		SkipCapture: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.RecoveryError != "" {
		t.Fatalf("pre-push recovery error = %q", outcome.RecoveryError)
	}
	// With post-rewrite disabled nothing linked the rewrite to its revision,
	// so the push records the missed commit as a revision of its own.
	recoveredChange, err := store.FindChangeByCommitID(context.Background(), repoRow.ID, missedOID)
	if err != nil || recoveredChange == nil || recoveredChange.JJChangeID == plainRevision {
		t.Fatalf("recovered change = %+v, %v; want a new revision for commit %s", recoveredChange, err, missedOID)
	}
	if got := strings.TrimSpace(gitOutput(t, primary, "rev-parse", "HEAD")); got != primaryHead {
		t.Fatalf("primary HEAD moved: got %s want %s", got, primaryHead)
	}
	if got := gitOutput(t, primary, "diff", "--cached", "--binary"); got != primaryIndex {
		t.Fatalf("primary index changed:\n%s", got)
	}
}

func runGitInRepo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func buildGxBinary(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "gx")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/gx")
	cmd.Dir = mustRepoRoot(t)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build gx: %v\n%s", err, combined)
	}
	return out
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("go.mod not found")
		}
		wd = parent
	}
}
