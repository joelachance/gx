package hooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/satoricorp/gx/internal/capture"
	"github.com/satoricorp/gx/internal/capture/matcher"
	"github.com/satoricorp/gx/internal/capture/orchestrator"
	"github.com/satoricorp/gx/internal/vcs"
)

// runCapture is a seam for tests that need one specific orchestrator outcome —
// above all a partial failure, which returns a populated Result *and* an error
// and is otherwise hard to produce on demand.
var runCapture = orchestrator.Run

// commitsInGitRange is a seam for tests: a git invocation that fails mid-push
// is real, used to abort the rest of the hook, and is not reachable on demand
// without corrupting a repo mid-run.
var commitsInGitRange = vcs.CommitsInGitRange

// PushOptions configures a hook-triggered capture run.
type PushOptions struct {
	RepoRoot    string
	RefRange    string
	Base        string
	Head        string
	HomeDir     string
	Tools       []string
	SkipCapture bool
}

// PushOutcome summarizes a non-blocking pre-push hook run.
//
// SkipReason is set whenever the hook deliberately did no work. Without it a
// paused capture, a repo opted out with `git config gx.enabled false`, and a
// failure to resolve the working directory all returned the same zero outcome,
// so all three rendered as the identical `capture staged extract= sessions=0`
// line — the same line a genuine error prints. An empty extract id has to mean
// one thing.
//
// AttachError reports a failure to link this push's sessions to their change
// rows. It is never returned: a push must not fail because SQLite was busy.
// But it must not be silent either — sessions that silently fail to attach
// hid the commit-time matcher's self-referential gate for 174 pushes.
type PushOutcome struct {
	Result orchestrator.Result
	// RecoveredCommits counts pushed commits ~/.gx had no revision for, made
	// while gx's hooks were not running, and recorded now.
	RecoveredCommits int
	SkipReason       string
	CaptureError     string
	RecoveryError    string
	AttachError      string
	AttachedSessions int
}

// RunPush captures the coding sessions behind a push and links them to the
// commits they wrote, all in ~/.gx. It always returns a nil error so git
// pre-push hooks never block a push.
func RunPush(ctx context.Context, opts PushOptions) (PushOutcome, error) {
	outcome := PushOutcome{}
	homeDir := opts.HomeDir
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}
	if capture.IsPaused(homeDir) {
		outcome.SkipReason = "capture is paused (GX_CAPTURE_PAUSED or ~/.gx/pause-capture)"
		return outcome, nil
	}

	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			// Not an opt-out — something is wrong. It is named as a failure so
			// it cannot hide behind a reason a user would read as intentional.
			outcome.SkipReason = fmt.Sprintf("resolve working directory: %v", err)
			outcome.CaptureError = outcome.SkipReason
			return outcome, nil
		}
		repoRoot = cwd
	}
	repoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		outcome.SkipReason = fmt.Sprintf("resolve repo path %q: %v", repoRoot, err)
		outcome.CaptureError = outcome.SkipReason
		return outcome, nil
	}
	// Per-repo opt-out: `git config gx.enabled false` excludes one repository
	// from gx, including from a machine-wide `gx init --global` install.
	if !EnabledForRepo(ctx, repoRoot) {
		outcome.SkipReason = "this repository opted out (git config gx.enabled false)"
		return outcome, nil
	}

	base, head := parseRefRange(opts.RefRange, opts.Base, opts.Head)
	tools := opts.Tools
	if len(tools) == 0 {
		tools = []string{capture.ToolClaude, capture.ToolCodex, capture.ToolCursor}
	}

	if !opts.SkipCapture {
		result, err := runCapture(ctx, orchestrator.RunOptions{
			RepoRoot:  repoRoot,
			Base:      base,
			Head:      head,
			HomeDir:   opts.HomeDir,
			Tools:     tools,
			StageOnly: true,
		})
		// The orchestrator returns a populated Result alongside its error: a run
		// that staged the extract and then failed on one session has produced
		// real rows and hunk links. Keeping the Result on both branches is what
		// lets the steps below still attach those sessions. Dropping it made a
		// partial failure look like a total no-op, which is the "silent
		// staging" symptom.
		outcome.Result = result
		if err != nil {
			// The hook must never block a push, but a failed capture run must
			// not read as "captured nothing" either — that hid a staging bug
			// behind a plausible sessions=0 for a long time.
			outcome.CaptureError = err.Error()
		}
	}

	refRange := strings.TrimSpace(opts.RefRange)
	if refRange == "" {
		refRange = strings.TrimSpace(outcome.Result.RefRange)
	}
	if refRange == "" && base != "" && head != "" {
		refRange = base + ".." + head
	}
	commits, err := commitsInGitRange(ctx, repoRoot, refRange)
	if err != nil {
		// Recovery is a backlog convenience, not a prerequisite. Carry on and
		// report the scan failure.
		outcome.RecoveryError = fmt.Sprintf("list pushed commits: %v", err)
		commits = nil
	}
	if len(commits) > 0 {
		if repo, repoErr := vcs.NewService().ResolveGxRepoAtPath(ctx, repoRoot); repoErr == nil {
			recovered, recoverErr := vcs.NewService().RecoverMissingCommits(ctx, repo, commits)
			outcome.RecoveredCommits = recovered
			if recoverErr != nil {
				outcome.RecoveryError = recoverErr.Error()
			}
		} else {
			outcome.RecoveryError = repoErr.Error()
		}
	}

	// Placement is load-bearing: RecoverMissingCommits above guarantees the
	// `changes` rows this links to exist.
	if attached, err := attachPushSessions(ctx, repoRoot, outcome.Result.HunkLinks); err != nil {
		outcome.AttachError = err.Error()
	} else {
		outcome.AttachedSessions = attached
	}

	return outcome, nil
}

// attachPushSessions links the sessions this push's hunk links name to the
// change rows they edited.
func attachPushSessions(ctx context.Context, repoRoot string, links []matcher.HunkLink) (int, error) {
	if len(links) == 0 {
		return 0, nil
	}
	service := vcs.NewService()
	repo, err := service.ResolveGxRepoAtPath(ctx, repoRoot)
	if err != nil {
		return 0, err
	}
	return service.AttachSessionsFromHunkLinks(ctx, repo, links)
}

func parseRefRange(refRange, base, head string) (string, string) {
	refRange = strings.TrimSpace(refRange)
	if base != "" || head != "" {
		if head == "" {
			head = "HEAD"
		}
		return base, head
	}
	if refRange == "" {
		return "", "HEAD"
	}
	if strings.Contains(refRange, "..") {
		parts := strings.SplitN(refRange, "..", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	// Single SHA from a new pushed ref. Let the orchestrator choose the best
	// available base instead of assuming "<sha>~20" exists.
	return "", refRange
}

// ParsePrePushLine parses one stdin line from git pre-push.
func ParsePrePushLine(line string) (localRef, localSHA, remoteRef, remoteSHA string, err error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) != 4 {
		return "", "", "", "", fmt.Errorf("expected 4 fields, got %d", len(fields))
	}
	return fields[0], fields[1], fields[2], fields[3], nil
}

// RefRangeForPush computes the capture ref range for one push update.
func RefRangeForPush(localSHA, remoteSHA string) string {
	zero := "0000000000000000000000000000000000000000"
	if localSHA == zero {
		return ""
	}
	if remoteSHA == zero {
		return localSHA
	}
	return remoteSHA + ".." + localSHA
}
