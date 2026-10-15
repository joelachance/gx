import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "gx: Install",
  description:
    "Install the gx CLI and its MCP server with one command, then set your own model API key.",
};

// This page used to wrap <DownloadCliInstall/>, which wrote a "seen" marker
// through Convex so the landing page could decide whether to nudge. The install
// itself is one curl, and the link is the same one the README and the post-
// install message print, so this is now a page that says the thing.
export default function DownloadPage() {
  return (
    <main className="mx-auto flex w-full max-w-xl flex-col gap-4 py-4">
      <div className="space-y-1">
        <h1 className="text-lg font-semibold tracking-tight text-zinc-950 dark:text-zinc-50">
          Install
        </h1>
        <p className="text-[13px] leading-5 text-zinc-600 dark:text-zinc-400">
          Run the install script. It places gx and its MCP server in
          ~/.local/bin and verifies the archive against its published checksum.
        </p>
      </div>

      <pre className="overflow-x-auto rounded bg-zinc-100 px-4 py-3 text-[13px] dark:bg-zinc-900">
        <code>curl -fsSL https://download.gx.run/install.sh | sh</code>
      </pre>

      <div className="space-y-1">
        <h2 className="text-sm font-semibold tracking-tight text-zinc-950 dark:text-zinc-50">
          Set a model key
        </h2>
        <p className="text-[13px] leading-5 text-zinc-600 dark:text-zinc-400">
          gx has no account. It reviews with your own key, sent straight to the
          provider and nowhere else.
        </p>
      </div>

      <pre className="overflow-x-auto rounded bg-zinc-100 px-4 py-3 text-[13px] dark:bg-zinc-900">
        <code>export ANTHROPIC_API_KEY=sk-ant-...</code>
      </pre>

      <p className="text-[13px] leading-5 text-zinc-600 dark:text-zinc-400">
        Then run <code className="text-zinc-950 dark:text-zinc-100">gx init</code> in
        a repository. Prefer GitHub Releases?{" "}
        <a
          href="https://github.com/satoricorp/gx/releases"
          className="underline underline-offset-2 hover:text-zinc-950 dark:hover:text-zinc-100"
        >
          The same archives are there
        </a>
        .
      </p>
    </main>
  );
}