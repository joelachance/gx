package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/satoricorp/gx/internal/codereview"
	"time"

	"github.com/satoricorp/gx/internal/gxtest"
)

// staleGitStatCache backdates the working tree's mtimes so the index's stat
// cache no longer matches it. This is the ordinary state of a fresh clone or
// checkout — the CI case gx review is built for — and it is the condition
// under which git status and git diff rewrite .git/index. Without it the
// cache is already current, git has nothing to refresh, and an index-guard
// regression would sail past this test.
func staleGitStatCache(t *testing.T, root string) {
	t.Helper()
	backdated := time.Now().Add(-time.Hour)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		return os.Chtimes(path, backdated, backdated)
	})
	if err != nil {
		t.Fatalf("backdate working tree: %v", err)
	}
}

func gitIndexDigest(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("read .git/index: %v", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// `gx review` has to be usable as a CI gate and on a checkout the reviewer
// does not own, so it must not auto-initialize anything: no git hooks, no gx
// home, no repo state. This test reviews a repo that has never run `gx init`
// and asserts the repo and the machine come out untouched.
func TestReviewNeverInitializesTheRepoOrTheMachine(t *testing.T) {
	root := newReviewGateRepo(t)
	commitOnBranch(t, root)
	t.Chdir(root)
	setReviewGateEnv(t)

	// Every other review test runs without credentials, which would make this
	// one vacuous: the biggest thing review can leave on a machine is the code
	// index manifest, and the refresh that writes it does not even start
	// without a key. So this test alone is given credentials and a backend that
	// answers successfully. A dead endpoint would not do — the test would then
	// pass because the network failed rather than because review kept its hands
	// off the machine, which is exactly how this assertion came to be satisfied
	// by accident.
	backend := gxtest.NewIndexBackend(t)
	backend.Use(t)

	// Point GX_HOME and HOME at paths that do not exist yet: anything that
	// opens the store or writes config has to create them, which makes the
	// mutation visible instead of silently landing in an existing directory.
	sandbox := t.TempDir()
	gxHome := filepath.Join(sandbox, "gx-home")
	fakeHome := filepath.Join(sandbox, "home")
	t.Setenv("GX_HOME", gxHome)
	t.Setenv("HOME", fakeHome)

	hooksDir := filepath.Join(root, ".git", "hooks")
	before := hookDirEntries(t, hooksDir)
	staleGitStatCache(t, root)
	// Reviewing runs git status and git diff, which refresh the stat cache and
	// rewrite .git/index unless review keeps them off it.
	indexBefore := gitIndexDigest(t, root)

	out, err := runReviewCommand(t, "--base", "main")
	if err != nil {
		t.Fatalf("gx review error = %v\n%s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatalf("gx review produced no report in an uninitialized repo")
	}
	// Indexing is gx Cloud's job, done on merge from the GitHub App. A review on
	// a machine with no gx home reads that index; it must not write one, which
	// would mean both re-embedding the whole checkout and leaving a manifest
	// behind.
	if backend.Upserted() {
		t.Fatalf("gx review indexed from a checkout with no gx home.\nrequests: %v", backend.Requests())
	}

	if after := hookDirEntries(t, hooksDir); !equalStrings(before, after) {
		t.Fatalf("gx review changed .git/hooks:\nbefore: %v\nafter:  %v", before, after)
	}
	for _, name := range before {
		data, readErr := os.ReadFile(filepath.Join(hooksDir, name))
		if readErr != nil {
			t.Fatalf("read hook %s: %v", name, readErr)
		}
		if strings.Contains(string(data), "gx ") || strings.Contains(string(data), "GX_") {
			t.Fatalf("gx review left a gx marker in .git/hooks/%s:\n%s", name, data)
		}
	}
	for _, path := range []string{gxHome, fakeHome, filepath.Join(root, ".gx")} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("gx review created %s (stat error = %v), want it untouched", path, statErr)
		}
	}
	if after := gitIndexDigest(t, root); after != indexBefore {
		t.Fatalf("gx review rewrote .git/index:\nbefore: %s\nafter:  %s", indexBefore, after)
	}
}

// A gate that fails is the tool working, not a crash. `gx review --fail-on`
// returning findings must not ship log tails and repo identity to gx Cloud,
// and must not mint a machine ID — a CI job that fails the gate on every run
// would otherwise upload on every run, from a machine that never ran gx.
func TestReviewGateFailureNeverWritesGxHome(t *testing.T) {
	root := newReviewGateRepo(t)
	writeTestFile(t, root, "package.json", "{\n  \"name\": \"example\"\n}\n")
	gitAddTestFiles(t, root, "package.json")
	runGitTest(t, root, "commit", "-m", "add package.json")
	t.Chdir(root)
	setReviewGateEnv(t)

	sandbox := t.TempDir()
	gxHome := filepath.Join(sandbox, "gx-home")
	fakeHome := filepath.Join(sandbox, "home")
	t.Setenv("GX_HOME", gxHome)
	t.Setenv("HOME", fakeHome)

	staleGitStatCache(t, root)
	indexBefore := gitIndexDigest(t, root)

	out, err := runReviewCommand(t, "--scope", "dependencies", "--fail-on", "strong", "--no-publish")
	if err == nil {
		t.Fatalf("gx review --fail-on strong exited 0 with findings:\n%s", out)
	}
	if code := ExitCode(err); code != reviewFindingsExitCode {
		t.Fatalf("ExitCode() = %d, want %d (error: %v)", code, reviewFindingsExitCode, err)
	}

	for _, path := range []string{gxHome, fakeHome, filepath.Join(root, ".gx")} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("a failing gate created %s (stat error = %v), want it untouched", path, statErr)
		}
	}
	if after := gitIndexDigest(t, root); after != indexBefore {
		t.Fatalf("a failing gate rewrote .git/index:\nbefore: %s\nafter:  %s", indexBefore, after)
	}
}

// `gx version` answers one question about the binary. A Dockerfile or CI step
// that runs it to check what it installed should not thereby acquire a machine
// ID and a $GX_HOME — the install event is not worth creating state on a
// machine that has not yet decided to use gx.
func TestVersionNeverWritesGxState(t *testing.T) {
	root := newReviewGateRepo(t)
	t.Chdir(root)
	setReviewGateEnv(t)

	sandbox := t.TempDir()
	gxHome := filepath.Join(sandbox, "gx-home")
	fakeHome := filepath.Join(sandbox, "home")
	t.Setenv("GX_HOME", gxHome)
	t.Setenv("HOME", fakeHome)

	cmd := NewRoot(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("gx version error = %v", err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatalf("gx version printed nothing")
	}
	for _, path := range []string{gxHome, fakeHome} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("gx version created %s (stat error = %v), want it untouched", path, statErr)
		}
	}
}

// The other gate outcome: nothing was reviewed. Same rule — a verdict, not a
// crash, so nothing gets uploaded.
func TestReviewNothingToReviewGateExits(t *testing.T) {
	root := newReviewGateRepo(t)
	t.Chdir(root)
	setReviewGateEnv(t)

	sandbox := t.TempDir()
	t.Setenv("GX_HOME", filepath.Join(sandbox, "gx-home"))
	t.Setenv("HOME", filepath.Join(sandbox, "home"))

	out, err := runReviewCommand(t, "--fail-on", "any", "--no-publish")
	if err == nil {
		t.Fatalf("gx review --fail-on any exited 0 without reviewing anything:\n%s", out)
	}
	if code := ExitCode(err); code != reviewNothingToReviewExitCode {
		t.Fatalf("ExitCode() = %d, want %d (error: %v)", code, reviewNothingToReviewExitCode, err)
	}
}

func hookDirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A review whose rule labeling failed still answers the gate's question: the
// findings and their severities are unchanged, only their names are missing.
// This pins the two channels apart, because folding labeling into
// DegradedReasons turned a failed label call into a failed gate.
func TestRuleLabelingFailureDoesNotDegradeTheGate(t *testing.T) {
	report := codereview.Report{
		RuleLabeling: []string{"1 of 1 rule-labeling batch(es) failed: model unreachable"},
	}
	if reason := gateDegradedReason(report); reason != "" {
		t.Errorf("gateDegradedReason() = %q, want \"\" — a missing rule name is not a degraded review", reason)
	}
	report.DegradedReasons = []string{"one reviewer did not run"}
	if reason := gateDegradedReason(report); reason == "" {
		t.Error("gateDegradedReason() = \"\", want the reviewer failure — real degradation must still gate")
	}
}
