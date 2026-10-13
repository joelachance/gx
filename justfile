set shell := ["zsh", "-cu"]
set dotenv-load := true

repo_root := `pwd`

# `just build` / `just install` stamp the version; VERSION defaults to dev.
gx_ldflags := "-X github.com/satoricorp/gx/internal/version.Version=${VERSION:-dev}"

build:
  go build -ldflags "{{gx_ldflags}}" -o gx ./cmd/gx

build-release:
  go build -ldflags "{{gx_ldflags}}" -o gx ./cmd/gx

package-cli:
  scripts/package-cli.sh

install-completions:
  mkdir -p ~/.local/share/bash-completion/completions ~/.zfunc
  go run ./cmd/gx-gen-completions ~/.local/share/bash-completion/completions/gx ~/.zfunc/_gx

install:
  mkdir -p ~/.local/bin
  just build
  command -v codesign >/dev/null 2>&1 && codesign --force --sign - ./gx || true
  cp ./gx ~/.local/bin/gx
  command -v codesign >/dev/null 2>&1 && codesign --force --sign - ~/.local/bin/gx || true
  just install-completions

test:
  go test ./...

# Build the MCP: xmcp JS output plus the standalone dist/gx-mcp binary.
mcp-build:
  cd mcp && bun install && bun run build

# Publish @satoricorp/gx to npm. prepublishOnly rebuilds and tests first.
# CI equivalent: push a tag like mcp-v0.1.0 (see .github/workflows/mcp.yml).
mcp-publish:
  cd mcp && bun install --frozen-lockfile && npm publish

run *args:
  {{repo_root}}/gx {{args}}

