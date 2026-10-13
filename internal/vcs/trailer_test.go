package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Commits made before gx stopped writing trailers keep the revision theirs
// names.
func TestParseRevisionIDsFromLegacyTrailers(t *testing.T) {
	id := "klpxswmqonvr"
	message := "add feature\n\n" + legacyRevisionTrailerPrefix + id
	got := ParseRevisionIDsFromMessage(message)
	if len(got) != 1 || got[0] != id {
		t.Fatalf("ParseRevisionIDsFromMessage() = %v, want [%s]", got, id)
	}
	dup := message + "\n\n" + legacyRevisionTrailerPrefix + id
	if got := ParseRevisionIDsFromMessage(dup); len(got) != 1 {
		t.Fatalf("ParseRevisionIDsFromMessage() duplicated = %v, want one id", got)
	}
	if got := ParseRevisionIDsFromMessage("add feature"); len(got) != 0 {
		t.Fatalf("ParseRevisionIDsFromMessage() on a plain message = %v, want none", got)
	}
}

func TestCommitsInGitRange(t *testing.T) {
	repo := t.TempDir()
	initTrailerTestRepo(t, repo)
	first := commitFile(t, repo, "first")
	second := commitFile(t, repo, "second")
	base := strings.TrimSpace(gitTestOutput(t, repo, "rev-parse", "HEAD~2"))

	got, err := CommitsInGitRange(context.Background(), repo, base+".."+second)
	if err != nil {
		t.Fatalf("CommitsInGitRange() error = %v", err)
	}
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("CommitsInGitRange() = %v, want [%s %s] oldest first", got, first, second)
	}
	single, err := CommitsInGitRange(context.Background(), repo, second+"~1.."+second)
	if err != nil {
		t.Fatalf("CommitsInGitRange(single) error = %v", err)
	}
	if len(single) != 1 || single[0] != second {
		t.Fatalf("CommitsInGitRange(single) = %v, want [%s]", single, second)
	}
}

func initTrailerTestRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
	} {
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

// commitFile commits one new file and returns the commit's SHA.
func commitFile(t *testing.T, repo, subject string) string {
	t.Helper()
	file := filepath.Join(repo, subject+".txt")
	if err := os.WriteFile(file, []byte(subject+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", filepath.Base(file)}, {"commit", "-m", subject}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", args[0], err, out)
		}
	}
	return strings.TrimSpace(gitTestOutput(t, repo, "rev-parse", "HEAD"))
}

func gitTestOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(out)
}
