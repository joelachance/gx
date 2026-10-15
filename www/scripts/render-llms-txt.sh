#!/usr/bin/env bash
# Regenerates docs/llms.txt and docs/llms-full.txt from the MDX.
#
# They were hand-maintained, which meant they went stale the moment a doc page
# changed and nothing caught it: the llms.txt description of Getting Started
# still promised "publish your first reviewed change" months after publishing
# was removed. Both files exist to describe the docs to a model, so a stale one
# is worse than no one -- it is a confident, wrong answer.
#
# Run after editing anything in docs/, and in CI so a drifted file fails the
# build instead of shipping.
set -euo pipefail

docs_dir="${1:-docs}"
cd "$docs_dir"

site_url="${GX_SITE_URL:-https://gx.run}"

# Page order follows meta.json so the index and the nav agree. A while-read
# rather than mapfile: macOS ships bash 3.2, and this runs there.
# meta.json names pages as slugs. A slug can be a file (cli -> cli.mdx) or a
# directory whose index is overview.mdx (changelog -> changelog/overview.mdx).
# Resolving both here is what keeps a page from silently vanishing from the
# index -- the first version of this script looked only for <slug>.mdx, and
# the changelog simply disappeared from llms.txt with no error.
page_list="$(node -e '
  const fs = require("fs");
  const meta = require("./meta.json");
  for (const p of meta.pages) {
    const flat = p + ".mdx";
    if (fs.existsSync(flat)) { console.log(flat); continue; }
    const nested = p + "/overview.mdx";
    if (fs.existsSync(nested)) { console.log(nested); continue; }
    console.error("meta.json lists " + p + " but no .mdx exists for it");
    process.exit(1);
  }
')"
pages=()
while IFS= read -r line; do
  [ -n "$line" ] && pages+=("$line")
done <<< "$page_list"

# frontmatter -> description, title
read_field() {
  awk -v key="$2" '
    NR == 1 && $0 != "---" { exit }
    NR == 1 { next }
    $0 == "---" { exit }
    index($0, key ": ") == 1 {
      v = substr($0, length(key) + 3)
      gsub(/^"|"$/, "", v)
      print v
      exit
    }
  ' "$1"
}

# body -> markdown with frontmatter and JSX stripped
body() {
  awk '
    NR == 1 && $0 != "---" { print; next }
    NR == 1 { fm = 1; next }
    fm && $0 == "---" { fm = 0; next }
    fm { next }
    { print }
  ' "$1" \
  | sed -e 's/<[A-Z][A-Za-z]*[^>]*>//g' -e 's|</[A-Za-z]*>||g'
}

# --- llms.txt: the index ---------------------------------------------------
{
  echo "# gx"
  echo
  echo "> gx is a CLI for code review. It reads the change you are about to ship,"
  echo "> holds it against the session that produced it, and gives you a ship /"
  echo "> no-ship verdict. It runs on your own model API key."
  echo
  echo "## Docs"
  echo
  for page in "${pages[@]}"; do
    [ -f "$page" ] || continue
    title="$(read_field "$page" title)"
    description="$(read_field "$page" description)"
    slug="${page%.mdx}"
    slug="${slug%/index}"
    [ "$slug" = "index" ] && slug=""
    printf -- '- [%s](%s/docs/%s): %s\n' "$title" "$site_url" "$slug" "$description"
  done
} > llms.txt

# --- llms-full.txt: every page in full -------------------------------------
{
  echo "# gx — full documentation"
  echo
  echo "Generated from the MDX sources by scripts/render-llms-txt.sh. Do not edit."
  echo
  for page in "${pages[@]}"; do
    [ -f "$page" ] || continue
    echo "---"
    echo
    echo "# $(read_field "$page" title)"
    echo
    echo "_$(read_field "$page" description)_"
    echo
    body "$page"
    echo
  done
} > llms-full.txt

echo "Wrote llms.txt ($(wc -l < llms.txt | tr -d ' ') lines) and llms-full.txt ($(wc -l < llms-full.txt | tr -d ' ') lines)"