# @satoricorp/gx — the gx MCP server

`gx-mcp` exposes `gx_review` to coding agents over stdio. It shells out to the
local `gx` CLI and runs entirely on your machine with your own model key.

Install gx and `gx-mcp` from source or a GitHub release, then run `gx init` in a
repository; it registers this server with Claude Code, Cursor and Codex when
they are installed. Put `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` in the server
config's `env` block: an MCP server inherits its host application's
environment, not your shell's. `GX_BINARY` points it at a specific `gx`.

`gx_review` is read-only: it never initializes a repository and never comments
on a pull request.

Everything else is in the main README: https://github.com/satoricorp/gx#readme
