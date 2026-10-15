"use client";

import { useEffect, useRef, useState } from "react";

type Tone =
  | "default"
  | "muted"
  | "heading"
  | "high"
  | "med"
  | "low"
  | "magenta"
  | "green"
  | "cyan"
  // The review report's own palette, lifted from the CLI so the demo and a
  // real `gx review` read as the same program: internal/termstyle/theme.go
  // for rail/dim/value/strong/command/warning/danger, and the review accent
  // (rgb 61,220,151) for mint.
  | "rail"
  | "dim"
  | "value"
  | "strong"
  | "mint"
  | "command"
  | "warning"
  | "danger"
  // Go syntax inside a finding's evidence snippet.
  | "kw"
  | "str"
  | "num";

/** A line is either one uniformly-toned string or a run of tinted spans. */
type Span = { text: string; tone?: Tone; bold?: boolean };
type Line = { text: string; tone?: Tone } | Span[];

const TONE_CLASS: Record<Tone, string> = {
  default: "text-zinc-300",
  muted: "text-zinc-500",
  heading: "text-zinc-100",
  high: "text-red-400",
  med: "text-amber-400",
  low: "text-zinc-400",
  magenta: "text-[#ff80ff]",
  green: "text-green-400",
  cyan: "text-cyan-400",
  rail: "text-[#71717a]",
  dim: "text-[#a1a1aa]",
  value: "text-[#d4d4d8]",
  strong: "text-[#fafafa]",
  mint: "text-[#3ddc97]",
  command: "text-[#818cf8]",
  warning: "text-[#f59e0b]",
  danger: "text-[#f87171]",
  kw: "text-[#ff80ff]",
  str: "text-[#3ddc97]",
  num: "text-[#f59e0b]",
};

const s = (text: string, tone?: Tone, bold?: boolean): Span => ({
  text,
  tone,
  bold,
});

/** The clack gutter every body row of the report hangs from. */
const RAIL = s("│  ", "rail");
/** A breathing row — the rail continues, so it must not collapse to a gap. */
const GAP: Span[] = [s("│", "rail")];

const GX_REVIEW: Line[] = [
  [
    s("◆  ", "mint"),
    s("gx review", "mint", true),
    s(" — feature/checkout-retry → main · 9 files, +284/−61", "dim"),
  ],
  [RAIL, s('intent: "make checkout survive gateway blips"', "dim")],
  GAP,

  // The run ledger: what gx did, in the order it did it.
  [
    RAIL,
    s("1  ", "rail"),
    s("scope       ", "strong"),
    s("9 files vs origin/main", "value"),
  ],
  [
    RAIL,
    s("2  ", "rail"),
    s("your tools  ", "strong"),
    s("go test ✓ 126", "mint"),
    s("   ", "rail"),
    s("go vet ✓", "mint"),
    s("   ", "rail"),
    s("govulncheck ✓", "mint"),
  ],
  [
    RAIL,
    s("3  ", "rail"),
    s("det rules   ", "strong"),
    s("14 ran · 13 pass · 1 finding", "value"),
  ],
  [
    RAIL,
    s("4  ", "rail"),
    s("graded      ", "strong"),
    s("11 asked · 7 cached · 4 graded · 3.9s", "value"),
  ],
  [
    RAIL,
    s("5  ", "rail"),
    s("verify      ", "strong"),
    s("2 confirmed by both graders · 1 demoted to advisory", "value"),
  ],
  GAP,

  [
    s("◇  ", "danger"),
    s("Verdict: NO-SHIP", "danger", true),
    s(" — 1 blocking finding (no-secrets-in-logs)", "value"),
  ],
  GAP,

  [s("◆  ", "mint"), s("BLOCKING", "danger", true), s("  1 finding", "value")],
  GAP,
  [RAIL, s("  Secrets must not reach logs, traces, or error strings", "strong", true)],
  [
    RAIL,
    s("  gx:recommended", "mint"),
    s("/no-secrets-in-logs", "value"),
    s("  ·  ", "rail"),
    s("internal/checkout/session.go:88", "dim"),
    s("  [graded]", "rail"),
  ],
  GAP,

  // Evidence: the three lines that make the finding checkable at a glance.
  [
    RAIL,
    s("   86 │  ", "rail"),
    s("    ", "value"),
    s("for", "kw"),
    s(" attempt := ", "value"),
    s("0", "num"),
    s("; attempt < ", "value"),
    s("maxRetries", "cyan"),
    s("; attempt++ {", "value"),
  ],
  [
    RAIL,
    s("   87 │  ", "rail"),
    s("        resp, err := gateway.", "value"),
    s("Charge", "cyan"),
    s("(ctx, idemKey, req)", "value"),
  ],
  [
    RAIL,
    s(" ", "rail"),
    s("→", "danger"),
    s(" 88 │  ", "rail"),
    s("        log.", "value"),
    s("Printf", "cyan"),
    s("(", "value"),
    s('"charge attempt %d failed: %+v"', "str"),
    s(", attempt, req)", "value"),
  ],
  GAP,

  [
    RAIL,
    s("Why  ", "warning"),
    s("req embeds PaymentToken, and %+v prints it — the raw token", "value"),
  ],
  [
    RAIL,
    s("     reaches your log sink once per retry, up to 4× per checkout.", "value"),
  ],
  [
    RAIL,
    s("Fix  ", "mint"),
    s("log req.ID and req.Amount instead of req.", "value"),
  ],
  GAP,
  [RAIL, s("both graders agreed · judge confirmed 0.91", "dim")],
  GAP,
  [
    RAIL,
    s("  ", "rail"),
    s("[f]", "dim"),
    s(" copy fix   ", "rail"),
    s("[s]", "dim"),
    s(" copy suppress line   ", "rail"),
    s("[o]", "dim"),
    s(" open in $EDITOR   ", "rail"),
    s("[esc]", "dim"),
    s(" back", "rail"),
  ],
  GAP,

  // The accordion menu: every lane the report can open, and what is in it.
  [
    s("◆  ", "mint"),
    s("Open a section", "strong", true),
    s("   ↑↓ move · enter open · esc collapse · j json", "rail"),
  ],
  GAP,
  [
    RAIL,
    s("  ", "rail"),
    s("● ", "mint"),
    s("BLOCKING       ", "danger"),
    s("1 finding   ", "value"),
    s("no-secrets-in-logs", "dim"),
  ],
  [
    RAIL,
    s("❯ ", "mint"),
    s("● ", "mint"),
    s("ADVISORY       ", "warning"),
    s("2 findings  ", "value"),
    s("reuse-before-rewrite · comments-match-code", "dim"),
  ],
  [
    RAIL,
    s("  ", "rail"),
    s("○ ", "rail"),
    s("FIX PLAN       ", "command"),
    s("3 steps     ", "value"),
    s("the agent's list — fix, then rerun", "dim"),
  ],
  [
    RAIL,
    s("  ", "rail"),
    s("○ ", "rail"),
    s("WORTH KNOWING  ", "command"),
    s("3 changes   ", "value"),
    s("passed, and yours now · 2 high materiality", "dim"),
  ],
  [
    RAIL,
    s("  ", "rail"),
    s("○ ", "rail"),
    s("RUN DETAILS    ", "strong"),
    s("            ", "value"),
    s("tools · cache · rules loaded", "dim"),
  ],
  [
    RAIL,
    s("  ", "rail"),
    s("○ ", "rail"),
    s("Done           ", "strong"),
    s("            ", "value"),
    s("exit 3 (no-ship)", "dim"),
  ],
  GAP,

  [
    s("└  ", "rail"),
    s("gx:recommended v2", "dim"),
    s(" + 4 rules from REVIEW.md · report ", "rail"),
    s("https://gx.run/r/satoricorp/gx/8f2a19c", "command"),
  ],
];

const CAT_AUTH: Line[] = [
  { text: "function requireUser(req, res, next) {", tone: "default" },
  { text: '  const header = req.headers.authorization || "";', tone: "default" },
  { text: '  const token = header.replace("Bearer ", "");', tone: "default" },
  { text: "  req.user = jwt.decode(token);   // never verifies the signature", tone: "high" },
  { text: "  if (!req.user) return res.status(401).json({ error: 'unauthorized' });", tone: "default" },
  { text: "  next();", tone: "default" },
  { text: "}", tone: "default" },
];

const HELP: Line[] = [
  { text: "available commands", tone: "heading" },
  { text: "  gx review          review the uncommitted change", tone: "default" },
  { text: "  ls                 list files", tone: "default" },
  { text: "  git status         show changed files", tone: "default" },
  { text: "  cat src/auth.js    show the auth module", tone: "default" },
  { text: "  clear              clear the screen", tone: "default" },
];

const LS: Line[] = [{ text: "README.md   package.json   src", tone: "cyan" }];

const GIT_STATUS: Line[] = [
  { text: "On branch main", tone: "default" },
  { text: "Changes not staged for commit:", tone: "default" },
  { text: "        modified:   src/auth.js", tone: "high" },
  { text: "        modified:   src/db.js", tone: "high" },
  { text: "        modified:   src/server.js", tone: "high" },
];

/** Map a typed line to canned output. Returns null for `clear`. */
function run(raw: string): Line[] | null {
  const cmd = raw.trim().replace(/\s+/g, " ");
  if (cmd === "") return [];
  if (cmd === "clear") return null;
  if (cmd === "help") return HELP;
  if (cmd === "gx review") return GX_REVIEW;
  if (cmd === "gx" || cmd.startsWith("gx "))
    return [{ text: "usage: gx review", tone: "muted" }];
  if (cmd === "ls" || cmd === "ls -la" || cmd === "ll") return LS;
  if (cmd.startsWith("git status")) return GIT_STATUS;
  if (cmd === "cat src/auth.js") return CAT_AUTH;
  if (cmd === "cat" || cmd.startsWith("cat "))
    return [{ text: `cat: ${cmd.slice(4)}: try 'cat src/auth.js'`, tone: "muted" }];
  const bin = cmd.split(" ")[0];
  return [{ text: `zsh: command not found: ${bin}`, tone: "muted" }];
}

type Entry = { id: number; cmd: string; lines: Line[]; pending?: boolean };

const SPINNER = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
/** gx "thinks" for this long before the findings appear. */
const REVIEW_DELAY_MS = 3200;

/**
 * A faux terminal for the /new playground: recognizes a handful of commands
 * and prints canned output. No backend, no wasm — just enough to let visitors
 * watch `gx review` flag the seeded findings.
 */
export function TerminalPanel({ className = "" }: { className?: string }) {
  const [entries, setEntries] = useState<Entry[]>([]);
  const [input, setInput] = useState("gx review");
  const [spinner, setSpinner] = useState(0);
  const nextId = useRef(0);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const lastEntryRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    // A review report is taller than the panel, so scrolling to the bottom
    // would land the reader on the footer. Anchor the newest command to the
    // top instead — clamped, so short output still settles like a shell.
    const last = lastEntryRef.current;
    const top = last
      ? last.getBoundingClientRect().top -
        el.getBoundingClientRect().top +
        el.scrollTop
      : el.scrollHeight;
    el.scrollTop = Math.max(0, Math.min(top, el.scrollHeight - el.clientHeight));
  }, [entries]);

  // Animate the spinner only while a review is "thinking".
  const thinking = entries.some((e) => e.pending);
  useEffect(() => {
    if (!thinking) return;
    const id = window.setInterval(
      () => setSpinner((s) => (s + 1) % SPINNER.length),
      90,
    );
    return () => window.clearInterval(id);
  }, [thinking]);

  function submit(raw: string) {
    const cmd = raw.trim().replace(/\s+/g, " ");
    if (cmd === "clear") {
      setEntries([]);
      return;
    }
    // gx review pauses to "think" before revealing findings.
    if (cmd === "gx review") {
      const id = nextId.current++;
      setEntries((prev) => [...prev, { id, cmd: raw, lines: [], pending: true }]);
      window.setTimeout(() => {
        setEntries((prev) =>
          prev.map((e) =>
            e.id === id ? { ...e, lines: GX_REVIEW, pending: false } : e,
          ),
        );
      }, REVIEW_DELAY_MS);
      return;
    }
    setEntries((prev) => [
      ...prev,
      { id: nextId.current++, cmd: raw, lines: run(raw) ?? [] },
    ]);
  }

  function onKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter") {
      submit(input);
      setInput("");
    }
  }

  return (
    // Keep wheel events inside the terminal from rotating the page's word wheel.
    <div
      onWheel={(e) => e.stopPropagation()}
      onClick={() => inputRef.current?.focus()}
      className={`flex h-full min-h-0 cursor-text flex-col overflow-hidden border border-zinc-800 bg-[#181716] ${className}`}
    >
      <div className="flex shrink-0 items-center gap-2 border-b border-zinc-800 px-3 py-2">
        <span className="flex gap-1.5" aria-hidden>
          <span className="h-2.5 w-2.5 rounded-full bg-zinc-700" />
          <span className="h-2.5 w-2.5 rounded-full bg-zinc-700" />
          <span className="h-2.5 w-2.5 rounded-full bg-zinc-700" />
        </span>
        <span className="text-[11px] tracking-[0.08em] text-zinc-500">
          api
        </span>
      </div>

      <div
        ref={scrollRef}
        className="min-h-0 flex-1 space-y-0.5 overflow-y-auto p-3 text-[11px] leading-[17px]"
      >
        <div className="text-zinc-500">{"'help' for commands"}</div>

        {/* Rows inside an entry sit flush so the report's `│` rail draws as one
            unbroken line; the spacing lives on the command line above them. */}
        {entries.map((entry, index) => (
          <div
            key={entry.id}
            ref={index === entries.length - 1 ? lastEntryRef : null}
          >
            <div className="pb-0.5 pt-1.5">
              <Prompt />
              <span className="text-zinc-200">{entry.cmd}</span>
            </div>
            {entry.pending ? (
              <Row
                line={{
                  text: `${SPINNER[spinner]} Reviewing changes...`,
                  tone: "mint",
                }}
              />
            ) : (
              entry.lines.map((line, i) => <Row key={i} line={line} />)
            )}
          </div>
        ))}

        {/* Hold the next prompt back until the review has finished loading. */}
        {!thinking && (
          <div className="flex items-baseline pt-1.5">
            <Prompt />
            <input
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={onKeyDown}
              size={Math.max(input.length, 1)}
              spellCheck={false}
              autoCapitalize="off"
              autoCorrect="off"
              aria-label="terminal input"
              className="bg-transparent text-zinc-100 caret-[#ff80ff] outline-none"
            />
            {entries.length === 0 && (
              <span className="ml-2 animate-pulse select-none whitespace-nowrap text-zinc-500">
                &lt;Hit Enter&gt;
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function Prompt() {
  return (
    <span aria-hidden className="mr-2 select-none whitespace-nowrap">
      <span className="text-green-400">➜</span>{" "}
      <span className="text-cyan-400">api</span>{" "}
      <span className="text-[#ff80ff]">git:(main)</span>
    </span>
  );
}

function Row({ line }: { line: Line }) {
  if (Array.isArray(line)) {
    return (
      <div className="whitespace-pre-wrap">
        {line.map((span, i) => (
          <span
            key={i}
            className={`${TONE_CLASS[span.tone ?? "default"]}${
              span.bold ? " font-semibold" : ""
            }`}
          >
            {span.text}
          </span>
        ))}
      </div>
    );
  }
  if (line.text === "") return <div className="h-3" />;
  return (
    <div className={`whitespace-pre-wrap ${TONE_CLASS[line.tone ?? "default"]}`}>
      {line.text}
    </div>
  );
}
