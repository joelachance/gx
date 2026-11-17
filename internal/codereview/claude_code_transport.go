package codereview

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// claudeCodeEnv switches the Claude Code wire: "1" uses it even when an
// Anthropic key is set, "0" never uses it. Unset, it is the fallback for a
// Claude model when no Anthropic key is set and `claude` is on PATH.
const claudeCodeEnv = "GX_REVIEW_CLAUDE_CODE"

// claudeCodeTransport implements modelTransport by running Claude Code
// headless (`claude -p`), so a review runs on whatever Claude Code is signed in
// with — a Claude subscription included — rather than on an API key.
//
// It goes through the CLI rather than sending the subscription's OAuth token
// to the Messages API itself: that token is issued to Claude Code, and the CLI
// is the supported way for another program to use it.
type claudeCodeTransport struct {
	binary string
	// run executes the CLI. Injected so tests never start a real one.
	run func(ctx context.Context, binary string, args []string, stdin string) (stdout, stderr []byte, err error)
}

// claudeCodeMode reads claudeCodeEnv: forced on, forced off, or neither.
func claudeCodeMode() (forced, disabled bool) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(claudeCodeEnv))) {
	case "1", "true", "on", "yes":
		return true, false
	case "0", "false", "off", "no":
		return false, true
	}
	return false, false
}

// claudeCodeBinary finds the Claude Code CLI, unless the wire is switched off.
func claudeCodeBinary() (string, bool) {
	if _, disabled := claudeCodeMode(); disabled {
		return "", false
	}
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", false
	}
	return path, true
}

// ClaudeCodeAvailable reports whether a Claude model can run through Claude
// Code on this machine.
func ClaudeCodeAvailable() bool {
	_, ok := claudeCodeBinary()
	return ok
}

// claudeCodePreferred reports whether Claude models should use Claude Code
// even though an Anthropic key may be set.
func claudeCodePreferred() bool {
	forced, _ := claudeCodeMode()
	return forced && ClaudeCodeAvailable()
}

func newClaudeCodeTransport(binary string) *claudeCodeTransport {
	return &claudeCodeTransport{binary: binary, run: runClaudeCode}
}

func (t *claudeCodeTransport) detail() string { return "Claude Code (claude -p, your Claude login)" }

// claudeCodeResult is the single JSON object `claude -p --output-format json`
// prints.
type claudeCodeResult struct {
	IsError    bool   `json:"is_error"`
	Subtype    string `json:"subtype"`
	Result     string `json:"result"`
	StopReason string `json:"stop_reason"`
}

func (t *claudeCodeTransport) complete(ctx context.Context, model, system, input string, _ int) (modelCompletion, error) {
	if t == nil || t.binary == "" {
		return modelCompletion{}, fmt.Errorf("Claude Code reviewer has no transport")
	}
	id := apiModelName(model)
	// The review is one completion, so everything that makes the CLI an agent
	// is switched off: no tools, no MCP servers, no settings, hooks or
	// CLAUDE.md from the user or the repository, and no transcript — a saved
	// one would be captured by gx as a coding session on the next push.
	args := []string{
		"-p",
		"--model", id,
		"--output-format", "json",
		"--tools", "",
		"--strict-mcp-config",
		"--setting-sources", "",
		"--disable-slash-commands",
		"--no-session-persistence",
	}
	if strings.TrimSpace(system) != "" {
		args = append(args, "--system-prompt", system)
	}
	ctx, cancel := context.WithTimeout(ctx, modelCallTimeout)
	defer cancel()
	stdout, stderr, runErr := t.run(ctx, t.binary, args, input)

	var result claudeCodeResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout), &result); err != nil {
		if runErr != nil {
			return modelCompletion{}, fmt.Errorf("run Claude Code for %s: %w: %s", id, runErr, firstLine(append(stderr, stdout...)))
		}
		return modelCompletion{}, fmt.Errorf("Claude Code returned no result for %s: %s", id, firstLine(stdout))
	}
	if result.IsError || runErr != nil {
		detail := strings.TrimSpace(result.Result)
		if detail == "" {
			detail = firstNonEmpty(result.Subtype, firstLine(stderr), "unknown error")
		}
		return modelCompletion{}, fmt.Errorf("Claude Code failed for %s: %s", id, detail)
	}
	if result.StopReason == "refusal" {
		return modelCompletion{}, fmt.Errorf("Anthropic model %s declined to review this change (stop_reason refusal)", id)
	}
	reply := strings.TrimSpace(result.Result)
	if reply == "" {
		return modelCompletion{}, fmt.Errorf("Claude Code returned no content for %s (stop_reason %s)", id, result.StopReason)
	}
	return modelCompletion{Text: reply, StopReason: result.StopReason}, nil
}

// runClaudeCode runs the CLI with the prompt on stdin.
func runClaudeCode(ctx context.Context, binary string, args []string, stdin string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	// Not the repository: nothing there should be read as project context.
	cmd.Dir = os.TempDir()
	cmd.Env = claudeCodeEnviron(os.Environ())
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// claudeCodeEnviron drops the Anthropic credentials from the child's
// environment. The CLI prefers a key over its own login, so leaving one in
// would bill the API account this wire exists to avoid.
func claudeCodeEnviron(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "ANTHROPIC_API_KEY=") || strings.HasPrefix(entry, "ANTHROPIC_AUTH_TOKEN=") {
			continue
		}
		out = append(out, entry)
	}
	return out
}
