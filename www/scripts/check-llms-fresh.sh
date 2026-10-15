#!/usr/bin/env bash
# Fails if docs/llms.txt or docs/llms-full.txt is not what the MDX would
# produce right now.
#
# They were hand-maintained and drifted silently: llms.txt described Getting
# Started as "publish your first reviewed change" long after publishing was
# removed. Nothing regenerates them, so nothing notices. This is the noticing,
# and render-llms-txt.sh is the fixing.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cp docs/llms.txt "$tmp/llms.txt"
cp docs/llms-full.txt "$tmp/llms-full.txt"

bash scripts/render-llms-txt.sh >/dev/null

status=0
for f in llms.txt llms-full.txt; do
  if ! diff -q "$tmp/$f" "docs/$f" >/dev/null; then
    echo "docs/$f is out of date with the MDX." >&2
    echo "Run: bun run docs:llms" >&2
    diff -u "$tmp/$f" "docs/$f" | head -30 >&2
    status=1
  fi
done

exit "$status"