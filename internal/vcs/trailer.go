package vcs

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var revisionTrailerLinePattern = regexp.MustCompile(
	`(?m)^` + regexp.QuoteMeta(legacyRevisionTrailerPrefix) + `(\S+)\s*$`,
)

// ParseRevisionIDsFromMessage returns the revision IDs in a commit message's
// legacy gx trailers. Only commits made before gx stopped writing them have any.
func ParseRevisionIDsFromMessage(message string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, match := range revisionTrailerLinePattern.FindAllStringSubmatch(message, -1) {
		if len(match) < 2 {
			continue
		}
		id := strings.TrimSpace(match[1])
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// CommitsInGitRange lists the commits in a ref range, oldest first.
func CommitsInGitRange(ctx context.Context, repoRoot, refRange string) ([]string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	refRange = strings.TrimSpace(refRange)
	if repoRoot == "" {
		return nil, fmt.Errorf("repo root required")
	}
	if refRange == "" {
		return nil, nil
	}
	args := []string{"-C", repoRoot, "rev-list", "--reverse", refRange}
	if !strings.Contains(refRange, "..") {
		// A bare rev means a new-branch push with no remote base; without a
		// bound the walk collects the whole history. Only commits that are not
		// already on a remote are being pushed.
		args = append(args, "--not", "--remotes")
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git rev-list %s: %w\n%s", refRange, err, strings.TrimSpace(string(out)))
	}
	return splitLines(strings.TrimSpace(string(out))), nil
}
