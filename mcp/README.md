# gx-mcp — the gx MCP server

`gx-mcp` exposes `gx_review` to coding agents over stdio. It shells out to the
local `gx` CLI and runs entirely on your machine, on your Claude Code login or
your own model key.

```bash
curl -fsSL https://download.gx.run/install.sh | sh
```

That installs both `gx` and `gx-mcp` into `~/.local/bin`. Then run `gx init` in a
repository; it registers this server with Claude Code, Cursor and Codex when
they are installed. To wire it up by hand, any MCP client takes:

```json
{
  "mcpServers": {
    "gx": {
      "command": "/Users/you/.local/bin/gx-mcp",
      "env": {}
    }
  }
}
```

With Claude Code signed in, that is all: reviews run on your Claude login. To
use an API key instead, put `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` in the
server config's `env` block, since an MCP server inherits its host
application's environment, not your shell's. `GX_BINARY` points it at a
specific `gx`.

`gx_review` is read-only: it never initializes a repository and never comments
on a pull request.

Everything else is in the main README: https://github.com/joelachance/gx#readme
