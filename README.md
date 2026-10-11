<p align="center">
  <img src="docs/gx-chrome-logo-dark.png" alt="gx" width="420" />
</p>

SOTA code review that uses your session context.


With traditional code review, developers held the context for code changes, but now AI sessions hold that context.
This can improve code review drastically, keeping both session history and informing future sessions of best practices.

gx runs entirely on your machine. There is no account and no server: what it
records stays in `~/.gx` (or `$GX_HOME`), and the only network calls it makes
are the ones you give it a key for.

## Install

Build from source (needs Go, at the version in `go.mod`):

```bash
git clone https://github.com/satoricorp/gx.git
cd gx
just install          # or: go build -o ~/.local/bin/gx ./cmd/gx
```

Or download a release from
[GitHub Releases](https://github.com/satoricorp/gx/releases). Each archive
holds `bin/gx` and the `bin/gx-mcp` MCP server:

```bash
tar -xzf gx_<version>_<os>_<arch>.tar.gz
install -m 755 gx/bin/gx ~/.local/bin/gx
install -m 755 gx/bin/gx-mcp ~/.local/bin/gx-mcp
```

gx's git hooks look for the binary at `~/.local/bin/gx`, then on `PATH`.
Upgrade the same way.

## Get started

Inside your repo:

```bash
gx init
```

Give `gx review` a model, with your own API key:

```bash
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY=sk-...
```

Then use Git as usual. gx hooks record each commit, and the pre-push hook
captures the coding session behind it:

```bash
gx review

git add -p
git commit -m "some commit message"
git push
```

## Models

`gx review` and `gx enhance` need one key, and the review panel follows it:

| Key set | Reviewers | Verifier |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | Claude Haiku 4.5 and Claude Sonnet 4.6 | Claude Haiku 4.5 |
| `OPENAI_API_KEY` only | GPT-5.6 Luna and GPT-5.6 Terra | GPT-5.6 Luna |

With both keys set the Claude panel runs; `GX_REVIEW_MODELS=openai` (or
`=anthropic`) picks one explicitly. `ANTHROPIC_AUTH_TOKEN` works in place of an
Anthropic key.

Any single slot can be overridden: `GX_REVIEW_MODEL_A`, `GX_REVIEW_MODEL_B`,
`GX_REVIEW_JUDGE_MODEL` and `GX_REVIEW_RULE_SORT_MODEL` take a model ID such as
`claude-opus-5-5` or `gpt-5.6-sol`. Claude models run on the Anthropic API and
everything else on the OpenAI API, so a panel can mix the two.
`GX_REVIEW_MODEL_B=off` reviews with one model, and `GX_REVIEW_AI=0` runs only
the deterministic checks and your project's own linters and tests.

## Code index

With `TURBOPUFFER_API_KEY` and `OPENAI_API_KEY` (for embeddings) set,
`gx index` indexes the repository, and reviews retrieve from it. Run it again
after significant changes; `gx doctor` reports how far the index is behind.
Without the keys, a review reads the change and the files around it.

## GitHub

With `GH_TOKEN` or `GITHUB_TOKEN` set, or the GitHub CLI signed in
(`gh auth login`), `gx review` comments on the branch's pull request.
`--no-comment` skips that.

## Commands

| Command | Alias | Purpose |
| --- | --- | --- |
| `gx init` | | Set up this repo (identity, hooks, MCP, agent `/gx` command) |
| `gx review [intent]` | `gxr` | Review the current change |
| `gx enhance [intent]` | `gxe` | Top fix from that review, as a prompt for your coding model |
| `gx index` | | Index the repository for review retrieval |
| `gx doctor` | | Diagnose (and optionally repair) local gx state |
| `gx version` | | Print version |
| `gx help` | | Help |

Most commands auto-init the repo on first use. Run `gx init` once so identity
and `AGENTS.md` are set the way you want.

### `gx init`

```bash
gx init                 # this repo
gx init --global        # machine-wide hooks via core.hooksPath
gx init --yes           # defaults, quiet on success
gx init --name "…" --email "…"
```

### `gx review [intent]`

Reviews the working tree by default.

```bash
gx review
gx review --base origin/main   # review a branch
gx review --json               # for agents
```

| Flag | Effect |
| --- | --- |
| `--base <ref>` | Review `<ref>...HEAD` instead of the working tree |
| `--repo` | Review the whole repository (wins over `--base`) |
| `--fast` | One reviewer, no verification pass |
| `--deep` | More local and indexed context |
| `--focus <path>` | Limit to files under a path prefix |
| `--json` / `--md` | Machine / PR-comment output |
| `--no-comment` | Skip the PR comment |
| `--fail-on <level>` | Gate: `none`, `any`, `speculative`, `worth-exploring`, `strong`, `blocking` |
| `--scope <lane>` | `architecture` (default), `security`, `performance`, `onboarding`, `docs`, `dependencies`, `testing`, `maintainability` |
| `--verbose` | Include repo facts, docs, and changed files |
| `--max-findings <n>` | Cap reported recommendations |

Findings are **blocking** (exit 3) or **advisory**. A finding blocks only when
both reviewers raised it and verification confirmed it.

### `gx enhance`

Same review, narrowed to the single highest-value fix, written as a prompt for
a coding model. Always exits 0.

```bash
gx enhance | pbcopy
gx enhance "fix the auth timeout" | claude -p "apply this"
gx enhance --base origin/main --fast
```

### `gx doctor` / `gx version`

```bash
gx doctor
gx doctor --fix
gx doctor --json

gx version
gx version --json
```

## MCP server

`gx-mcp` exposes `gx_review` to coding agents over stdio. `gx init` registers
it with Claude Code, Cursor and Codex when they are installed. An MCP server
inherits its host application's environment, not your shell's, so put
`ANTHROPIC_API_KEY` or `OPENAI_API_KEY` in the server config's `env` block.
`GX_BINARY` points it at a specific `gx`. The tool is read-only: it never
initializes a repository and never comments on a pull request.

To build it from source, run `just mcp-build` (needs [Bun](https://bun.sh)).

## Privacy

gx keeps its data on your machine, in `~/.gx` or `$GX_HOME`. It sends no
analytics or telemetry, and `git push` uploads nothing beyond the push itself:
the pre-push hook only captures the session into `~/.gx`, redacting known
secret formats from session events as it does.

What leaves your machine, and only with credentials you set:

- **Reviews.** `gx review` and `gx enhance` send the change and its review
  context to the model provider for each model: the Anthropic API or the
  OpenAI API.
- **Code index.** With TurboPuffer and OpenAI keys set, `gx index` and review
  send code chunks to OpenAI for embeddings and to TurboPuffer for storage and
  search.
- **PR comments.** With a GitHub token, `gx review` posts its comment to
  GitHub. `--no-comment` skips it.

## Uninstall

```bash
rm -f ~/.local/bin/gx ~/.local/bin/gxr ~/.local/bin/gxe ~/.local/bin/gx-mcp
rm -f ~/.local/share/bash-completion/completions/gx ~/.zfunc/_gx
```

Local data stays in `~/.gx` (or `$GX_HOME`).

## Contributing

Bug reports, fixes, and docs are all welcome. For anything larger than a small
fix, open a discussion or an issue first so we can agree on the shape before
you write it. Questions go in
[Discussions](https://github.com/satoricorp/gx/discussions).

You need Go (the version in `go.mod`) and Git; [just](https://github.com/casey/just)
and zsh for the build recipes; and [Bun](https://bun.sh) only for the MCP server
in `mcp/`.

```bash
go build ./...
go vet ./...
go test ./...
```

The tests need no network access or credentials; the few live tests skip
themselves unless `GX_TEST_ALLOW_NETWORK=1` is set.

Two test rules come from production bugs that shipped past a green suite
([AGENTS.md](AGENTS.md) has the full story):

- **Seed through the code production uses.** Write test state with the same
  storage functions the product calls. `TestSeedersAreProductionWriters` fails
  on any exported storage method that only tests call.
- **Prefer a weathered database to a pristine one.** Build test state with
  `storagetest.New(t, shapes...)` and repositories with `gxtest.NewWorld(t)`,
  rather than a bare temporary `GX_HOME`. Mark a deliberate cold start with
  `storagetest.NoSessions()`.

Pull requests: branch from `main`, keep each PR to one logical change with
tests, run `gofmt`, `go vet` and `go test ./...` first (CI runs the same on
Linux and macOS), and describe what changed and why in your own words. Sign off
every commit to certify the
[Developer Certificate of Origin](https://developercertificate.org/):
`git commit -s`. Everyone taking part follows the
[code of conduct](CODE_OF_CONDUCT.md).

Report security issues privately as described in [SECURITY.md](SECURITY.md).

## License

Copyright (C) 2026 Joe LaChance.

gx is free software, licensed under the
[GNU Affero General Public License v3.0 only](LICENSE). See [NOTICE](NOTICE).
By contributing, you agree that your contribution is licensed under the same
terms.
