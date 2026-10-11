import type { XmcpConfig } from "xmcp";

const config: XmcpConfig = {
  stdio: {
    silent: true,
  },
  http: false,
  paths: {
    tools: "src/tools",
    prompts: false,
    resources: false,
  },
  template: {
    name: "gx MCP",
    instructions: [
      "gx MCP exposes gx_review.",
      "Save work with plain Git: git add, then git commit. gx installs Git hooks that stamp each commit with its gx revision trailer and record it, so no gx-specific commit verb is needed.",
      "Publish with plain git push, then open the PR with gh pr create. The gx pre-push hook captures the agent session and links its edits to the changed hunks. Do not run gx capture push; it bypasses that hook.",
      "Run gx_review for better codegen context from local facts, previous sessions, PRs, and current code changes.",
      "Use git status for inspection.",
    ].join(" "),
  },
};

export default config;
