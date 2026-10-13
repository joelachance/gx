package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/satoricorp/gx/internal/codereview"
	"github.com/satoricorp/gx/internal/vcs"
	"github.com/spf13/cobra"
)

// Exit codes for `gx review` as an automated gate. Both are distinct from the
// generic failure exit so a CI step can tell a policy failure from a crash.
const (
	// reviewFindingsExitCode means findings at or above --fail-on survived.
	reviewFindingsExitCode = 3
	// reviewNothingToReviewExitCode means the review inspected no code at all.
	// A gate that passes without looking is the false pass --fail-on exists to
	// prevent, so an explicit gate treats "never looked" as a failure too.
	reviewNothingToReviewExitCode = 4
	// reviewDegradedExitCode means code was read but the review that read it was
	// incomplete — no model ran, or only part of the subject reached one. Same
	// reasoning as above: "no findings at or above X" is a claim about what was
	// inspected, and a gate must not make it on a review that did not run.
	reviewDegradedExitCode = 5
)

func newReviewCommand(ctx context.Context) *cobra.Command {
	var scope string
	var focus string
	var base string
	var wholeRepo bool
	var deep bool
	var verbose bool
	var jsonOut bool
	var markdownOut bool
	var failOn string
	var noPublish bool
	var noComment bool
	var fast bool
	var maxFindings int
	var clientOverride string
	cmd := &cobra.Command{
		Use:     "review [intent]",
		Aliases: []string{"gxr"},
		Short:   "Review changes based on codebase & session context, along with independent resources",
		Long: `Review the current change — the working tree by default, a commit range with
--base, or the whole repository with --repo.

The optional INTENT is one sentence saying what the change was supposed to do,
in your words: "make checkout survive gateway blips", "fix the auth timeout".
It is not a question and not a filter. The reviewer holds the diff against it:
a change that quietly does more than the intent says is a finding
(scope-matches-intent), and the story lane's "asked" line quotes it back. With
no intent, gx derives one from the branch name and recent commit subjects and
says so in the report.

Findings land in two lanes. BLOCKING stops the change and exits 3; ADVISORY is
reported and does not. A finding blocks only when both reviewers raised it and
the verification model confirmed it — one model's opinion is advisory. Under a
terminal the report opens as a menu; piped or in CI it prints in full. --json
and --md are the machine and PR-comment shapes.

gx review needs ANTHROPIC_API_KEY or OPENAI_API_KEY, and the panel follows the
key: Claude Haiku 4.5 and Claude Sonnet 4.6 with an Anthropic key, GPT-5.6 Luna
and GPT-5.6 Terra with only an OpenAI key. GX_REVIEW_MODELS=anthropic or
=openai picks one when both are set. Any single slot's env var
(GX_REVIEW_MODEL_A/_B, GX_REVIEW_JUDGE_MODEL, GX_REVIEW_RULE_SORT_MODEL) wins
over the panel; GX_REVIEW_MODEL_B=off reviews with one model.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			startedAt := time.Now()
			failOnLevel, err := codereview.ParseFailOnLevel(failOn)
			if err != nil {
				return err
			}
			// A mistyped GX_REVIEW_MODELS would silently review with the
			// defaults; say so before spending a model call.
			if err := codereview.ValidateModelPresetEnv(); err != nil {
				return err
			}
			if err := requireReviewer(); err != nil {
				return err
			}
			reviewScope := ""
			scopeExplicit := cmd.Flags().Changed("scope")
			if scopeExplicit {
				reviewScope = scope
			}
			prompt := ""
			if len(args) > 0 {
				prompt = strings.TrimSpace(args[0])
			}
			// Store-free on purpose: review must leave no gx state behind in a
			// repo (or on a machine) that has never run `gx init`.
			repo, err := vcs.NewService().ResolveGitRepoWithoutStore(ctx)
			if err != nil {
				return err
			}
			runReview := func(progress io.Writer) (codereview.Report, error) {
				return codereview.Review(ctx, repo.RootPath, codereview.Options{
					Scope:          reviewScope,
					Deep:           deep,
					Focus:          focus,
					Base:           base,
					WholeRepo:      wholeRepo,
					Prompt:         prompt,
					Verbose:        verbose,
					Fast:           fast,
					MaxFindings:    maxFindings,
					ProgressWriter: progress,
					Color:          !jsonOut && !markdownOut,
				})
			}
			var report codereview.Report
			if jsonOut {
				// Machine-readable output: no spinner, no ANSI, nothing on
				// stdout but the report.
				report, err = runReview(nil)
			} else {
				report, err = runReviewWithLoader(cmd.InOrStdin(), cmd.ErrOrStderr(), runReview)
			}
			if err != nil {
				return err
			}
			switch {
			case jsonOut:
				if err := writeReviewJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			case markdownOut:
				fmt.Fprint(cmd.OutOrStdout(), codereview.RenderMarkdown(report))
			default:
				// The terminal render: lanes, fix plan, story, and the
				// Verdict/Next seam last. On a real terminal it is the accordion
				// — the same sections as a menu the reader walks. Piped, in CI,
				// or under GX_PLAIN_PROMPTS it is the linear text, and either
				// way the exit code below is the same. RenderMarkdown remains
				// the PR-comment body and the --md shape.
				if !browseReviewInteractively(cmd.InOrStdin(), cmd.OutOrStdout(), report) {
					fmt.Fprint(cmd.OutOrStdout(), codereview.RenderReviewText(report))
				}
			}
			// The saved-report notice goes to stderr so --json stdout stays
			// pure report.
			if reportPath, saveErr := saveReviewReport(repo.RootPath, report, startedAt); saveErr != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), labelWarningValue("Report", saveErr.Error()))
			} else if reportPath != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), labelValue("Report", muted("saved to "+reportPath)))
			}
			// Nothing was reviewed: commenting would overwrite a real review
			// comment with a no-op.
			if !noPublish && !noComment && report.Reviewed {
				postReviewSummaryComment(ctx, repo, report, cmd.ErrOrStderr())
			}
			return reviewGateError(report, failOnLevel, cmd.Flags().Changed("fail-on"))
		},
	}
	cmd.Flags().StringVar(&scope, "scope", codereview.DefaultScope, "review scope when explicitly set: architecture, security, performance, onboarding, docs, dependencies, testing, maintainability")
	cmd.Flags().StringVar(&focus, "focus", "", "limit review to files under this path prefix")
	cmd.Flags().StringVar(&base, "base", "", "review the commit range <ref>...HEAD instead of the working tree, e.g. --base origin/main")
	cmd.Flags().BoolVar(&wholeRepo, "repo", false, "review the whole repository rather than just the current change; uncommitted work stays in focus, and this wins over --base")
	cmd.Flags().BoolVar(&deep, "deep", false, "run full-spectrum review with more local and indexed context")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "include repo facts, docs, and changed files")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print the review report as JSON instead of the terminal render")
	cmd.Flags().BoolVar(&markdownOut, "md", false, "print the review report as markdown — the PR-comment body — instead of the terminal render")
	// The gate is on by default. "blocking" is lane-aware: it fails only on
	// findings that earned the blocking lane — a Blocking deterministic check,
	// or a Strong finding both reviewers raised and the judge confirmed. A
	// Strong finding one model raised is demoted to advisory and does not fail
	// it. --fail-on none restores the advisory-only exit 0.
	cmd.Flags().StringVar(&failOn, "fail-on", string(codereview.FailOnBlocking), fmt.Sprintf("exit %d when findings at or above this level survive: %s; the default %q fails only on findings in the blocking lane, and \"none\" turns the gate off (exit %d when there was nothing to review, exit %d when the review ran degraded)", reviewFindingsExitCode, strings.Join(codereview.FailOnLevels(), ", "), string(codereview.FailOnBlocking), reviewNothingToReviewExitCode, reviewDegradedExitCode))
	cmd.Flags().BoolVar(&noComment, "no-comment", false, "skip posting the PR review comment")
	// --no-publish and --client named the gx Cloud history record, which is
	// gone. They are still accepted, hidden, because installed MCP servers,
	// slash commands and scripts pass them, and an unknown flag would fail
	// the review outright.
	cmd.Flags().BoolVar(&noPublish, "no-publish", false, "same as --no-comment")
	cmd.Flags().StringVar(&clientOverride, "client", "", "accepted for compatibility; ignored")
	_ = cmd.Flags().MarkHidden("no-publish")
	_ = cmd.Flags().MarkHidden("client")
	cmd.Flags().IntVar(&maxFindings, "max-findings", 0, "cap how many recommendations the review reports (0 uses the default); applies to both what the model is asked for and what is reported")
	cmd.Flags().BoolVar(&fast, "fast", false, "optimize for wall clock: one reviewer instead of two, no verification pass, and findings written without code examples")
	return cmd
}

// saveReviewReport persists the full JSON report to
// <repoRoot>/review/review-findings-<timestamp>.json so a review survives the
// terminal it was printed in. Returns the repo-relative path, or "" when the
// review inspected nothing (an empty report is not worth a file).
func saveReviewReport(repoRoot string, report codereview.Report, startedAt time.Time) (string, error) {
	if !report.Reviewed {
		return "", nil
	}
	dir := filepath.Join(repoRoot, "review")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// Best-effort: without the exclude entry a saved report dirties
	// `git status` and shows up as a changed file in the NEXT review, but a
	// failure to write it should not cost the report itself.
	_ = ensureReviewDirIgnored(repoRoot)
	name := "review-findings-" + startedAt.UTC().Format("20060102-150405") + ".json"
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	if err := writeReviewJSON(file, report); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return filepath.Join("review", name), nil
}

// ensureReviewDirIgnored appends /review/ to .git/info/exclude so saved
// reports never show as untracked files. The exclude file is repo-local and
// never committed, so the ignore does not reach collaborators — a team that
// wants reports tracked can still add them explicitly with `git add -f`.
func ensureReviewDirIgnored(repoRoot string) error {
	gitDir, err := resolveGitCommonDir(repoRoot)
	if err != nil {
		return err
	}
	excludePath := filepath.Join(gitDir, "info", "exclude")
	existing, err := os.ReadFile(excludePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == "/review/" {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
		return err
	}
	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "/review/\n"
	return os.WriteFile(excludePath, []byte(content), 0o644)
}

// resolveGitCommonDir finds the directory whose info/exclude git actually
// reads: .git itself for a normal checkout, the pointed-to gitdir for a
// linked worktree — and, when that gitdir carries a commondir file, the
// shared common directory (per-worktree gitdirs' info/exclude is ignored by
// git).
func resolveGitCommonDir(repoRoot string) (string, error) {
	gitPath := filepath.Join(repoRoot, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", err
	}
	gitDir := gitPath
	if !info.IsDir() {
		data, readErr := os.ReadFile(gitPath)
		if readErr != nil {
			return "", readErr
		}
		pointer := strings.TrimSpace(string(data))
		const prefix = "gitdir:"
		if !strings.HasPrefix(pointer, prefix) {
			return "", fmt.Errorf(".git is neither a directory nor a gitdir pointer")
		}
		gitDir = strings.TrimSpace(strings.TrimPrefix(pointer, prefix))
		if !filepath.IsAbs(gitDir) {
			// Git resolves relative gitdir pointers against the directory
			// containing the .git file, not the repo root.
			gitDir = filepath.Join(filepath.Dir(gitPath), gitDir)
		}
	}
	if data, readErr := os.ReadFile(filepath.Join(gitDir, "commondir")); readErr == nil {
		common := strings.TrimSpace(string(data))
		if !filepath.IsAbs(common) {
			common = filepath.Join(gitDir, common)
		}
		gitDir = common
	}
	return filepath.Clean(gitDir), nil
}

// reviewGateError turns a review outcome into an exit code.
//
// Two strictnesses live here, and which one applies depends on whether the
// caller asked for a gate or got the default.
//
// The default (--fail-on blocking, flag not passed) is the linter contract:
// blocking-lane findings exit 3, and everything else — including a clean
// tree and a run that could not reach a model — exits 0. Interactive use, the
// MCP tool, and the slash commands all live here, and for them "nothing to
// fix" must be a green run, not a refusal.
//
// An explicit --fail-on (any level, passed on the command line) is a CI gate,
// and a gate is held to the stricter rule: "nothing to review" exits 4 and a
// degraded run exits 5, because a gate that passes on a diff it never opened,
// or on a review that never happened, is exactly the silent false pass
// --fail-on exists to catch. --fail-on none is advisory-only: every outcome
// stays exit 0.
func reviewGateError(report codereview.Report, level codereview.FailOnLevel, explicit bool) error {
	if !level.Enabled() {
		return nil
	}
	if !report.Reviewed {
		if !explicit {
			return nil
		}
		target := strings.TrimSpace(report.ReviewTarget)
		if target == "" {
			target = "the working tree"
		}
		return vcs.CodedErrorf(reviewNothingToReviewExitCode, fmt.Errorf("gx review: nothing was reviewed (looked at %s); refusing to pass a gate without inspecting any code", target))
	}
	// A degraded run is not a clean run with fewer findings. When no model ran,
	// `findings` is whatever the deterministic checks produced — usually nothing
	// once patch-focus filtering is applied — so the gate exited 0 and the pull
	// request merged reporting a review that never happened. The rendered report
	// says so in a banner, but an exit code is the only thing a CI step reads.
	// The default path still reports the degradation in the banner; it just
	// does not turn a degraded interactive run into a red exit.
	if reason := gateDegradedReason(report); reason != "" && explicit {
		return vcs.CodedErrorf(reviewDegradedExitCode, fmt.Errorf("gx review: %s; refusing to pass a gate on an incomplete review", reason))
	}
	failures := report.GateFailures(level)
	if len(failures) == 0 {
		return nil
	}
	return vcs.CodedErrorf(reviewFindingsExitCode, fmt.Errorf("gx review: %d finding(s) at or above %q", len(failures), string(level)))
}

// gateDegradedReason states why this review cannot answer the gate's question,
// or "" when it can. Both signals are already computed and rendered; no gate
// path read either until now.
func gateDegradedReason(report codereview.Report) string {
	if len(report.DegradedReasons) > 0 {
		return strings.Join(report.DegradedReasons, "; ")
	}
	if report.Coverage.Partial() {
		if statement := strings.TrimSpace(report.Coverage.Statement()); statement != "" {
			return statement
		}
		return "only part of the change was reviewed"
	}
	return ""
}

// requireReviewer refuses a review that has no model to review with.
//
// Without one, every reviewer leg comes back unavailable: the run still
// produces a report, still prints a verdict, and still writes a findings file,
// but nothing with a model ever looked at the change. It reports DEGRADED,
// which reads like a review that went slightly wrong rather than one that
// never happened. A user spent three days on exactly that, re-running a
// command that was telling them the truth in a form they could not act on.
//
// Refusing costs nothing that was working: a review with no reviewer had none
// either way. GX_REVIEW_AI=0 still runs, because the deterministic rules alone
// are a real review.
func requireReviewer() error {
	if reason := codereview.NoReviewerReason(); reason != "" {
		return fmt.Errorf("gx review cannot run: %s (or set GX_REVIEW_AI=0 to run only the deterministic checks)", reason)
	}
	return nil
}
