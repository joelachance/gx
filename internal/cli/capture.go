package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/satoricorp/gx/internal/capture"
	"github.com/satoricorp/gx/internal/capture/claudehooks"
	"github.com/satoricorp/gx/internal/capture/orchestrator"
	"github.com/satoricorp/gx/internal/capture/reparse"
	"github.com/satoricorp/gx/internal/hooks"
	"github.com/satoricorp/gx/internal/storage"
)

func newCaptureCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "capture",
		Short:  "Capture agent sessions for pushed commits",
		Hidden: true,
	}
	cmd.AddCommand(newCapturePushCommand(ctx))
	cmd.AddCommand(newCaptureReparseCommand(ctx))
	cmd.AddCommand(newCaptureTranscriptCommand(ctx))
	return cmd
}

// runPushHook is a seam for tests: the real hook runner needs a git repo and a
// live staging database, neither of which is the subject of the command's own
// error reporting.
var runPushHook = hooks.RunPush

func newCapturePushCommand(ctx context.Context) *cobra.Command {
	var (
		repoRoot string
		refRange string
		base     string
		head     string
		tools    string
		// Passed by the pre-push hook scripts already installed in users'
		// repositories, and no longer used. They stay accepted, hidden: an
		// unknown flag would fail the hook, and its `|| true` would hide that
		// capture had stopped.
		remote, localRef, headSHA string
	)
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Run capture pipeline for a pushed ref range",
		RunE: func(cmd *cobra.Command, args []string) error {
			toolList := splitCaptureTools(tools)
			outcome, err := runPushHook(ctx, hooks.PushOptions{
				RepoRoot: repoRoot,
				RefRange: refRange,
				Base:     base,
				Head:     head,
				Tools:    toolList,
			})
			// The command still exits 0 — it runs inside a git hook and must
			// never block a push — but a failure is printed rather than
			// discarded. Swallowing it left an empty `extract=` as the only
			// evidence that anything went wrong, which reads exactly like a
			// clean run that found nothing.
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: capture push failed: %v\n", err)
			}
			result := outcome.Result
			// A run that deliberately did nothing says so. Printing the
			// staging line here would emit the byte-for-byte signature of a
			// failed run (`extract=` empty) for a paused capture or a repo
			// that opted out, which is how one line came to mean four things.
			if outcome.SkipReason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "capture skipped: %s\n", outcome.SkipReason)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "capture staged extract=%s sessions=%d coverage=%.1f%% ref=%s attached=%d\n",
					result.StagedExtractID,
					result.StagedSessions,
					result.HunkCoverage*100,
					result.RefRange,
					outcome.AttachedSessions,
				)
				// Zero coverage with sessions on disk and work in the range is
				// not a hand-written change; it is capture reading a
				// transcript the agent had not finished writing. Say so, or
				// the staging line reads as a verdict on the change.
				if result.AttributionSuspect {
					fmt.Fprintln(cmd.ErrOrStderr(), labelWarningValue("Warning", fmt.Sprintf(
						"found agent sessions but linked none of %d changed hunk(s); session data was likely still being written. Re-run once the session settles: gx capture push --repo %s --ref-range %s",
						result.EligibleHunks, repoRoot, result.RefRange,
					)))
				}
			}
			if outcome.CaptureError != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: capture failed, so this push has no session context: %s\n", outcome.CaptureError)
			}
			if outcome.RecoveryError != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: repair gx revision metadata: %s\n", outcome.RecoveryError)
			}
			if outcome.AttachError != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: this push's sessions were not linked to its commits: %s\n", outcome.AttachError)
			}
			// One tool being unavailable is survivable but never silent: it
			// means this push carries less session context than it could.
			for _, problem := range result.DiscoveryProblems {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: capture skipped a tool: %s\n", problem)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo", "", "repository root (default: current directory)")
	cmd.Flags().StringVar(&refRange, "ref-range", "", "git ref range (e.g. abc..def)")
	cmd.Flags().StringVar(&base, "base", "", "base ref when ref-range omitted")
	cmd.Flags().StringVar(&head, "head", "", "head ref when ref-range omitted")
	cmd.Flags().StringVar(&tools, "tools", "claude,codex,cursor", "comma-separated capture tools")
	cmd.Flags().StringVar(&remote, "remote", "", "ignored")
	cmd.Flags().StringVar(&localRef, "local-ref", "", "ignored")
	cmd.Flags().StringVar(&headSHA, "head-sha", "", "ignored")
	for _, name := range []string{"remote", "local-ref", "head-sha"} {
		_ = cmd.Flags().MarkHidden(name)
	}
	return cmd
}

func splitCaptureTools(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func installCaptureHook(cmd *cobra.Command, repoRoot string) error {
	return installCaptureHookWithOutput(cmd, repoRoot, true)
}

func installCaptureHookQuiet(cmd *cobra.Command, repoRoot string) error {
	return installCaptureHookWithOutput(cmd, repoRoot, false)
}

func installCaptureHookWithOutput(cmd *cobra.Command, repoRoot string, printSuccess bool) error {
	if repoRoot == "" {
		var err error
		repoRoot, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	gxPath, err := os.Executable()
	if err != nil {
		return err
	}
	if err := hooks.Install(hooks.InstallOptions{RepoRoot: repoRoot, GxPath: gxPath}); err != nil {
		return err
	}
	if printSuccess {
		fmt.Fprintln(cmd.OutOrStdout(), labelValue("gx lifecycle hooks", success("ok")+": commit metadata and push capture"))
	}
	return nil
}

func installClaudeCaptureHooks(cmd *cobra.Command, repoRoot string, quiet bool) error {
	gxPath, err := os.Executable()
	if err != nil {
		return err
	}
	if err := claudehooks.MergeSettings(repoRoot, gxPath); err != nil {
		return err
	}
	if !quiet {
		fmt.Fprintln(cmd.OutOrStdout(), labelValue("Claude hooks", success("ok")+": Stop/SessionEnd transcript capture"))
	}
	return nil
}

func newCaptureReparseCommand(ctx context.Context) *cobra.Command {
	var repoRoot string
	cmd := &cobra.Command{
		Use:   "reparse",
		Short: "Re-run normalization over stored raw session blobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoRoot == "" {
				var err error
				repoRoot, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			stager, err := storage.OpenCaptureStager(ctx)
			if err != nil {
				return err
			}
			result, err := reparse.Run(ctx, stager, repoRoot)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "capture reparse: sessions=%d updated=%d errors=%d\n",
				result.Sessions, result.Updated, result.Errors)
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo", "", "repository root (default: current directory)")
	return cmd
}

func newCaptureTranscriptCommand(ctx context.Context) *cobra.Command {
	var (
		path     string
		repoRoot string
		tool     string
	)
	cmd := &cobra.Command{
		Use:   "transcript",
		Short: "Ingest a Claude/Codex/Cursor transcript path from hooks or --path",
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				resolved, err := transcriptPathFromStdin(cmd.InOrStdin())
				if err != nil {
					return err
				}
				path = resolved
			}
			if strings.TrimSpace(path) == "" {
				return nil
			}
			if repoRoot == "" {
				var err error
				repoRoot, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if tool == "" {
				tool = inferTranscriptTool(path)
			}
			stager, err := storage.OpenCaptureStager(ctx)
			if err != nil {
				return err
			}
			session, badLines, err := orchestrator.IngestRawSession(ctx, stager, tool, path, raw, repoRoot, nil)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "capture transcript: tool=%s session=%s bad_lines=%d events=%d\n",
				tool, session.SessionID, badLines, len(session.Events))
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "transcript JSONL path")
	cmd.Flags().StringVar(&repoRoot, "repo", "", "repository root (default: current directory)")
	cmd.Flags().StringVar(&tool, "tool", "", "agent tool (claude, codex, cursor)")
	return cmd
}

func transcriptPathFromStdin(in io.Reader) (string, error) {
	if in == nil {
		return "", nil
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	payload := strings.TrimSpace(string(data))
	if payload == "" {
		return "", nil
	}
	var hook map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &hook); err != nil {
		return "", nil
	}
	var path string
	if err := json.Unmarshal(hook["transcript_path"], &path); err != nil {
		return "", nil
	}
	return strings.TrimSpace(path), nil
}

func inferTranscriptTool(path string) string {
	lower := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lower, "/.claude/"):
		return capture.ToolClaude
	case strings.Contains(lower, "/.codex/"):
		return capture.ToolCodex
	case strings.Contains(lower, "/.cursor/"):
		return capture.ToolCursor
	default:
		return capture.ToolClaude
	}
}
