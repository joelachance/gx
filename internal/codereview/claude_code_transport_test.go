package codereview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClaudeOnPath puts an executable named claude first on PATH. It is never
// run: the tests only need LookPath to find one.
func fakeClaudeOnPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return path
}

func TestClaudeModelsFallBackToClaudeCodeWithoutAKey(t *testing.T) {
	binary := fakeClaudeOnPath(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv(claudeCodeEnv, "")

	transport, err := transportForModel("claude-haiku-4-5")
	if err != nil {
		t.Fatalf("transportForModel: %v", err)
	}
	cc, ok := transport.(*claudeCodeTransport)
	if !ok || cc.binary != binary {
		t.Fatalf("transport = %#v, want Claude Code at %s", transport, binary)
	}

	// A key wins by default, and GX_REVIEW_CLAUDE_CODE=1 hands it back.
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	if transport, _ = transportForModel("claude-haiku-4-5"); !isAnthropicTransport(transport) {
		t.Fatalf("with a key set, transport = %T, want the Anthropic API", transport)
	}
	t.Setenv(claudeCodeEnv, "1")
	if transport, _ = transportForModel("claude-haiku-4-5"); !isClaudeCodeTransport(transport) {
		t.Fatalf("with %s=1, transport = %T, want Claude Code", claudeCodeEnv, transport)
	}

	// Switched off, a missing key is an error again rather than a silent run.
	t.Setenv(claudeCodeEnv, "0")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	if _, err = transportForModel("claude-haiku-4-5"); err == nil {
		t.Fatal("no key and Claude Code off: want an error")
	}
}

func isAnthropicTransport(t modelTransport) bool  { _, ok := t.(*anthropicTransport); return ok }
func isClaudeCodeTransport(t modelTransport) bool { _, ok := t.(*claudeCodeTransport); return ok }

func TestClaudeCodeTransportRunsOneToollessCompletion(t *testing.T) {
	var gotArgs []string
	var gotStdin string
	transport := &claudeCodeTransport{binary: "claude", run: func(_ context.Context, _ string, args []string, stdin string) ([]byte, []byte, error) {
		gotArgs, gotStdin = args, stdin
		return []byte(`{"is_error":false,"result":" {\"recommendations\":[]} ","stop_reason":"end_turn"}`), nil, nil
	}}
	completion, err := transport.complete(context.Background(), "anthropic:claude-sonnet-4-6", "be a reviewer", "the brief", 0)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completion.Text != `{"recommendations":[]}` || completion.StopReason != "end_turn" {
		t.Fatalf("completion = %+v", completion)
	}
	if gotStdin != "the brief" {
		t.Fatalf("stdin = %q, want the brief", gotStdin)
	}
	joined := strings.Join(gotArgs, "\x00")
	for _, want := range []string{
		"-p", "--model\x00claude-sonnet-4-6", "--output-format\x00json", "--tools\x00\x00",
		"--strict-mcp-config", "--setting-sources\x00\x00", "--no-session-persistence",
		"--system-prompt\x00be a reviewer",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q lack %q", gotArgs, want)
		}
	}
}

func TestClaudeCodeTransportReportsFailuresAsFailures(t *testing.T) {
	cases := map[string]struct {
		stdout string
		err    error
		want   string
	}{
		"signed out":  {`{"is_error":true,"result":"Invalid API key · Please run /login"}`, errors.New("exit status 1"), "Please run /login"},
		"refusal":     {`{"is_error":false,"result":"","stop_reason":"refusal"}`, nil, "declined"},
		"empty reply": {`{"is_error":false,"result":"  ","stop_reason":"end_turn"}`, nil, "no content"},
		"not json":    {`command not found`, errors.New("exit status 127"), "command not found"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			transport := &claudeCodeTransport{binary: "claude", run: func(context.Context, string, []string, string) ([]byte, []byte, error) {
				return []byte(tc.stdout), nil, tc.err
			}}
			_, err := transport.complete(context.Background(), "claude-haiku-4-5", "", "brief", 0)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestClaudeCodeChildNeverSeesAnAnthropicKey(t *testing.T) {
	got := claudeCodeEnviron([]string{"PATH=/bin", "ANTHROPIC_API_KEY=sk", "ANTHROPIC_AUTH_TOKEN=tok", "HOME=/h"})
	if strings.Join(got, ",") != "PATH=/bin,HOME=/h" {
		t.Fatalf("child env = %v", got)
	}
}
