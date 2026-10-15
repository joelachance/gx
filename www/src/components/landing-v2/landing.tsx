"use client";

import { Fragment, useRef, useState } from "react";
// Aliased: the deck below defines its own `Copy` for the inline markup
// marks (~bold~, ~italic~), and the import would shadow it.
import { Check as CheckIcon, Copy as CopyIcon } from "lucide-react";
import { GxLogo } from "@/components/gx-logo";
import { BenchmarkCharts } from "@/components/landing-v2/benchmark-charts";
import { berkeleyMono } from "@/components/landing-v2/fonts";
import { TerminalPanel } from "@/components/landing-v2/terminal-panel";
import {
  SLIDE_NAMES,
  type SlideName,
  TextWheel,
  type TextWheelHandle,
  slideIndexFor,
} from "@/components/landing-v2/text-wheel";
import { useWheelDriver } from "@/components/landing-v2/use-wheel-driver";
import { useWheelFits } from "@/components/landing-v2/use-wheel-fits";
import { WheelNav } from "@/components/landing-v2/wheel-nav";
import {
  DOCS_URL,
  FOUNDER_CALL_URL,
  SUPPORT_EMAIL_URL,
} from "@/lib/site-links";

/** One block of copy. Several lines means hard breaks between them — the copy
 * is written to specific line endings, not left to wrap where it lands.
 *
 * Two inline marks are available, and they are deliberately separate: some
 * phrases are stressed, some are coloured, and they are not the same phrases.
 *   `*italic*`   — emphasis, in the body colour
 *   `~accent~`   — the magenta accent, upright */
type Paragraph = readonly string[];

/** Splits on the marks; `split` keeps the captured delimiters, so the marked
 * runs come back inline with the plain ones. */
const MARKED = /(\*[^*]+\*|~[^~]+~)/;

function Copy({ line }: { line: string }) {
  return (
    <>
      {line.split(MARKED).map((part, i) => {
        const marked = part.length > 2 && part.at(0) === part.at(-1);
        if (marked && part.startsWith("*")) {
          return (
            <em key={i} className="italic">
              {part.slice(1, -1)}
            </em>
          );
        }
        if (marked && part.startsWith("~")) {
          // Written out rather than built from a constant: Tailwind only emits
          // an arbitrary value it can see spelled out in the source.
          return (
            <span key={i} className="text-[#b06ab5]">
              {part.slice(1, -1)}
            </span>
          );
        }
        return <Fragment key={i}>{part}</Fragment>;
      })}
    </>
  );
}

type Screen =
  | {
      kind: "text";
      paragraphs: readonly Paragraph[];
      /** Index of the paragraph the sign-up button follows, when the slide
       * carries one — the button reads as part of the copy, so it sits where
       * the copy calls for it rather than always at the end. */
      signUpAfter?: number;
      /** Closing link, in the bracketed style of the footer links. */
      cta?: { label: string; href: string };
      /** The benchmark bar charts, rendered after the copy. */
      benchmarks?: boolean;
    }
  | { kind: "terminal" };

/** The install CTA. `full` is the one anchoring the left column; `compact` is
 * the lighter one that sits inside a slide, where it follows body copy rather
 * than heading a column. */
const SIGN_UP_SIZES = {
  full: { box: "px-4 py-3 text-[11px] leading-4 max-w-[380px]", icon: "h-3.5 w-3.5" },
  // `leading-4` is load-bearing: an arbitrary text size carries no line-height
  // of its own, so without it the button inherits the slide's `leading-6` and
  // comes out exactly as tall as the full-size one.
  compact: { box: "px-4 py-2 text-[11px] leading-4", icon: "h-3 w-3" },
} as const;

const INSTALL_COMMAND_COPY = "curl -fsSL https://download.gx.run/install.sh | sh";
const INSTALL_COMMAND_DISPLAY = "curl -fsSL https://download.gx.run";

/**
 * The install command, as a call to action.
 *
 * It used to be a link to GitHub sign-in. There is no account, so there is
 * nothing to link to -- and a button that goes nowhere is worse than the thing
 * it replaced, because someone who clicks it expecting to be let in gets
 * nothing at all.
 *
 * It keeps the shape of a CTA: one solid block, white on the dark column, the
 * only high-contrast thing on it. What it does now is hand over the one
 * command that starts the product, with a copy button so the command does not
 * have to be transcribed by hand off a screen.
 */
function InstallCommand({
  size,
  className,
}: {
  size: keyof typeof SIGN_UP_SIZES;
  className?: string;
}) {
  const { box, icon } = SIGN_UP_SIZES[size];
  const [copied, setCopied] = useState(false);

  // Best-effort, and it reports its own failure in the button state. A
  // clipboard write rejects on an insecure origin, and the command is on screen
  // either way -- so this is a convenience, never the only way to get it.
  async function copyInstall() {
    try {
      await navigator.clipboard.writeText(INSTALL_COMMAND_COPY);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch (error: unknown) {
      console.error("Failed to copy the install command", error);
    }
  }

  const Icon = copied ? CheckIcon : CopyIcon;

  return (
    <div
      className={`group inline-flex items-center gap-3 bg-white font-medium text-black transition-colors hover:bg-zinc-100 ${box} ${className ?? ""}`}
    >
      <code className="min-w-0 font-mono font-normal">{INSTALL_COMMAND_DISPLAY}</code>
      <button
        type="button"
        onClick={() => void copyInstall()}
        aria-label={copied ? "Copied install command" : "Copy install command"}
        className="ml-auto inline-flex shrink-0 cursor-pointer items-center gap-1.5 text-black/60 transition-colors hover:text-black"
      >
        <Icon className={icon} aria-hidden="true" />
        {copied ? "Copied" : "Copy"}
      </button>
    </div>
  );
}

/**
 * The deck, keyed by the wheel word that selects it — so a renamed or missing
 * slide is a type error rather than a screen the wheel can't reach.
 */
const SCREENS: Readonly<Record<SlideName, Screen>> = {
  "The Problem": {
    kind: "text",
    paragraphs: [
      ["We’re tired of reviewing AI slop."],
      [
        "Hallucinations are rampant with newer frontier models.",
        "The worst part is that models are confidently plausible.",
      ],
      ["Code is now abundant, and it’s difficult to review changes confidently."],
      ["*Human attention is now the scarce resource.*"],
    ],
  },
  "The Solution": {
    kind: "text",
    paragraphs: [
      [
        "In human review, developers look for bugs and code quality based on coding standards.",
        "So, our agentic review needs rules.",
      ],
      [
        "The other half of the equation is understanding the code.",
        "*This is now the bottleneck.*",
      ],
      [
        "GX automates best in class rulesets, and ~helps you understand the most important changes~ so you’re not left in the dark.",
      ],
    ],
  },
  Example: { kind: "terminal" },
  "Proof & Benchmarks": {
    kind: "text",
    paragraphs: [
      [
        "GX identifies bugs using a unique workflow resulting in ~finding twice the real issues of GPT 5.6~, increasing the efficacy of your review.",
      ],
      [
        "GX also uses a unique, weighted ruleset of 33 foundational rules, based in computer science principles and industry-wide best practices.",
      ],
      [
        "GX returns 55% fewer false positives than leading rabbit-based competitors.",
      ],
    ],
    benchmarks: true,
  },
  "Get Started": {
    kind: "text",
    paragraphs: [
      ["Try GX right now."],
      ["Setup takes 2 minutes, and it runs entirely on your machine."],
      [
        "If you have questions or want the founder to set GX up for you, say hello.",
      ],
    ],
    signUpAfter: 1,
    cta: { label: "[Say hello]", href: FOUNDER_CALL_URL },
  },
};

const FOOTER_LINKS = [
  { label: "[Documentation]", href: DOCS_URL },
  { label: "[Say hi]", href: SUPPORT_EMAIL_URL },
] as const;

export function LandingV2() {
  // Too small for the ring and the copy to coexist: drop the wheel and let the
  // deck scroll like an ordinary page instead of hijacking the gesture.
  const hasWheel = useWheelFits();
  const wheelRef = useRef<TextWheelHandle>(null);
  const { activeStep, incomingSlide, moving, settleProgress, spinToIndex } =
    useWheelDriver(wheelRef, hasWheel);

  // The ring repeats the deck, so every pocket folds back onto a slide.
  const activeScreen = slideIndexFor(activeStep);

  const spinToSlide = (slideIndex: number) => {
    spinToIndex(slideIndex);
  };

  /** Fade out current and fade in the committed adjacent page only — never intermediates. */
  const slideOpacity = (index: number): number => {
    if (!moving || incomingSlide === null) {
      return index === activeScreen ? 1 : 0;
    }
    const from = activeScreen;
    const to = incomingSlide;
    if (index !== from && index !== to) return 0;
    if (from === to) return index === from ? 1 : 0;
    const p = Math.min(1, Math.max(0, settleProgress));
    if (index === from) return 1 - p;
    if (index === to) return p;
    return 0;
  };

  return (
    <main
      className={`${berkeleyMono.className} relative bg-[#181716] text-zinc-100 ${
        hasWheel ? "h-dvh overflow-hidden" : "min-h-dvh pb-32"
      }`}
    >
      {hasWheel ? (
        <>
          <TextWheel ref={wheelRef} />
          <WheelNav
            activeSlide={activeScreen}
            incomingSlide={incomingSlide}
            moving={moving}
            settleProgress={settleProgress}
            onSlideClick={spinToSlide}
          />
        </>
      ) : null}

      <div className="relative z-10 mx-auto grid w-full max-w-6xl gap-14 px-8 pt-24 sm:pt-36 lg:grid-cols-[1fr_1.8fr] lg:gap-12">
        <div className="flex min-w-0 flex-col items-start">
          {/* The footer mark sits inset inside its canvas — pull it back so
              its left edge lines up with the text below. */}
          <div className="-ml-10">
            <GxLogo variant="footer" scale={2} />
          </div>

          <p className="mt-5 flex items-center gap-2.5 text-xs text-zinc-100">
            <span className="font-extralight">&copy; 2026</span>
            <span className="flex items-center gap-2">
              <span
                aria-hidden
                className="h-2.5 w-2.5 shrink-0 rounded-full bg-current"
              />
              Most Fun Corporation
            </span>
          </p>

          <div className="mt-2 flex flex-col items-start gap-1.5 text-xs">
            {FOOTER_LINKS.map((link) => (
              <a
                key={link.label}
                href={link.href}
                className="text-zinc-100 transition-colors hover:text-white hover:underline"
              >
                {link.label}
              </a>
            ))}
          </div>

          <InstallCommand size="full" className="mt-7" />
          <a
            href="https://github.com/satoricorp/gx"
            className="mt-3 inline-flex text-zinc-300 transition-colors hover:text-white"
            aria-label="Source on GitHub"
          >
            <svg
              className="h-5 w-5"
              viewBox="0 0 24 24"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
                clipRule="evenodd"
              />
            </svg>
          </a>
        </div>

        <div
          className={`relative text-xs leading-6 text-[#f4f4f5] lg:mt-14 ${
            hasWheel ? "" : "space-y-20"
          }`}
        >
          {SLIDE_NAMES.map((name, index) => {
            const screen = SCREENS[name];
            return (
            <div
              key={name}
              className={`${
                screen.kind === "terminal" ? "h-[620px]" : "max-w-md space-y-6"
              } ${hasWheel ? "absolute inset-x-0 top-0" : ""}`}
              style={
                hasWheel
                  ? {
                      opacity: slideOpacity(index),
                      // Example terminal (and CTAs) must receive clicks on the
                      // active slide; inactive slides stay inert.
                      pointerEvents:
                        slideOpacity(index) > 0.5 ? "auto" : "none",
                    }
                  : undefined
              }
            >
              {/* Without the wheel there is nothing else naming the slides. */}
              {hasWheel ? null : (
                <p className="mb-6 text-xs tracking-[0.08em] text-[#ea580c]">
                  {name}
                </p>
              )}
              {screen.kind === "terminal" ? (
                <TerminalPanel className="h-full" />
              ) : (
                <>
                  {screen.paragraphs.map((lines, paragraphIndex) => (
                    <Fragment key={lines[0]}>
                      <p>
                        {lines.map((line, lineIndex) => (
                          <Fragment key={line}>
                            {lineIndex > 0 ? <br /> : null}
                            <Copy line={line} />
                          </Fragment>
                        ))}
                      </p>
                      {screen.signUpAfter === paragraphIndex ? (
                        <InstallCommand size="compact" />
                      ) : null}
                    </Fragment>
                  ))}
                  {screen.benchmarks ? <BenchmarkCharts /> : null}
                  {screen.cta ? (
                    <a
                      href={screen.cta.href}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-block text-[#b06ab5] transition-colors hover:text-[#cfa3d3] hover:underline"
                    >
                      {screen.cta.label}
                    </a>
                  ) : null}
                </>
              )}
            </div>
            );
          })}
        </div>
      </div>
    </main>
  );
}

