package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/satoricorp/gx/internal/capture/orchestrator"
	"github.com/satoricorp/gx/internal/hooks"
)

// TestCapturePushSurfacesHookError pins the diagnostic contract of the push
// command: a failed run is reported on stderr, and the staging line still
// prints so its empty `extract=` keeps meaning "the run errored" rather than
// "the run found nothing".
func TestCapturePushSurfacesHookError(t *testing.T) {
	restore := runPushHook
	runPushHook = func(context.Context, hooks.PushOptions) (hooks.PushOutcome, error) {
		return hooks.PushOutcome{}, errors.New("open capture stager: disk is full")
	}
	t.Cleanup(func() { runPushHook = restore })

	var stdout, stderr bytes.Buffer
	cmd := newCapturePushCommand(context.Background())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, want nil so the hook never blocks the push", err)
	}

	if !strings.Contains(stderr.String(), "disk is full") {
		t.Fatalf("stderr = %q, want the hook error surfaced", stderr.String())
	}
	if !strings.Contains(stdout.String(), "capture staged extract= ") {
		t.Fatalf("stdout = %q, want the staging line with an empty extract id", stdout.String())
	}
}

// TestCapturePushWarnsAboutSkippedTools proves a tool that could not be
// discovered reaches the operator instead of quietly shrinking the capture.
func TestCapturePushWarnsAboutSkippedTools(t *testing.T) {
	restore := runPushHook
	runPushHook = func(context.Context, hooks.PushOptions) (hooks.PushOutcome, error) {
		return hooks.PushOutcome{Result: orchestrator.Result{
			StagedExtractID:   "ext-1",
			DiscoveryProblems: []string{"claude: read /home/u/.claude/projects: permission denied"},
		}}, nil
	}
	t.Cleanup(func() { runPushHook = restore })

	var stdout, stderr bytes.Buffer
	cmd := newCapturePushCommand(context.Background())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "permission denied") {
		t.Fatalf("stderr = %q, want the skipped tool reported", stderr.String())
	}
}

// TestCapturePushDistinguishesSkipsFromFailures pins the meaning of the
// staging line. RunPush returns an empty outcome on four separate paths, and
// the command printed the staging line unconditionally, so a paused capture and
// a repo opted out with `git config gx.enabled false` emitted the byte-for-byte
// signature of an errored run — `capture staged extract= sessions=0 …` — with
// nothing on stderr. An empty extract id has to mean exactly one thing.
func TestCapturePushDistinguishesSkipsFromFailures(t *testing.T) {
	restore := runPushHook
	runPushHook = func(context.Context, hooks.PushOptions) (hooks.PushOutcome, error) {
		return hooks.PushOutcome{
			SkipReason: "this repository opted out (git config gx.enabled false)",
		}, nil
	}
	t.Cleanup(func() { runPushHook = restore })

	var stdout, stderr bytes.Buffer
	cmd := newCapturePushCommand(context.Background())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "capture staged extract=") {
		t.Fatalf("stdout = %q, want a deliberate skip not rendered as an errored staging line", stdout.String())
	}
	if !strings.Contains(stdout.String(), "capture skipped: this repository opted out") {
		t.Fatalf("stdout = %q, want the skip named", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want an opt-out to warn about nothing", stderr.String())
	}
}
