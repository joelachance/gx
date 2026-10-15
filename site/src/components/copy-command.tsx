"use client";

import { Check, Copy } from "lucide-react";
import { useState } from "react";

type CopyCommandProps = {
  command: string;
  onCopy?: () => void;
};

export function CopyCommand({ command, onCopy }: CopyCommandProps) {
  const [copied, setCopied] = useState(false);

  async function copyCommand() {
    // The clipboard write is best-effort: it rejects on an insecure origin or a
    // denied permission. Report that in the icon state, but still fire onCopy —
    // the funnel's Continue hangs off it, and a clipboard failure must not
    // strand someone on this step with no way forward.
    let wrote = false;
    try {
      await navigator.clipboard.writeText(command);
      wrote = true;
    } catch (error: unknown) {
      console.error("Failed to copy command to the clipboard", error);
    }

    if (wrote) {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    }
    onCopy?.();
  }

  const Icon = copied ? Check : Copy;

  return (
    <button
      type="button"
      onClick={() => void copyCommand()}
      aria-label={copied ? "Copied command" : `Copy command: ${command}`}
      className="flex min-h-9 w-full cursor-pointer items-center gap-3 bg-zinc-100 px-2.5 py-1.5 text-left text-zinc-950 transition-colors hover:bg-zinc-200/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400 dark:bg-zinc-900 dark:text-zinc-50 dark:hover:bg-zinc-800/70 dark:focus-visible:ring-zinc-600"
    >
      <code className="min-w-0 flex-1 overflow-x-auto whitespace-pre font-mono text-xs">
        {command}
      </code>
      <span
        aria-hidden="true"
        className="inline-flex h-6 w-6 shrink-0 items-center justify-center text-zinc-500 dark:text-zinc-400"
      >
        <Icon className="h-3.5 w-3.5" />
      </span>
    </button>
  );
}
