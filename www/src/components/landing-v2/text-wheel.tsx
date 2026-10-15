"use client";

import { forwardRef, useImperativeHandle, useMemo, useRef } from "react";

/**
 * The deck, in order. The wheel doubles as the nav: the pocket sitting
 * horizontal names the slide on screen.
 */
export const SLIDE_NAMES = [
  "The Problem",
  "The Solution",
  "Example",
  "Proof & Benchmarks",
  "Get Started",
] as const;

export type SlideName = (typeof SLIDE_NAMES)[number];

export const SLIDE_COUNT = SLIDE_NAMES.length;

/** Orange accent used for the active slide in the nav list. */
export const HIGHLIGHT_COLOR = "#ea580c";

/** Laps of the deck around the ring. The titles repeat so the arc stays as
 * dense as it looks, and the deck keeps cycling as you keep scrolling. */
const WHEEL_REPEATS = 12;
export const WORD_COUNT = SLIDE_COUNT * WHEEL_REPEATS;
export const WHEEL_STEP_DEG = 360 / WORD_COUNT;
/** Tight ring — sharp curvature so the visible arc stays short. */
export const RADIUS_PX = 160;
/** Wheel center sits this far off the left edge — only the right arc peeks in. */
export const CENTER_OFFSET_PX = -60;
/** Center height above the viewport bottom. */
export const CENTER_BOTTOM_PX = 96;

/** Gap between the wheel's right edge and the page list. */
export const NAV_LIST_GAP_PX = 24;

/** Page list anchor above the viewport bottom. */
export const NAV_BOTTOM_PX = 36;

/** Visual tick marks on the wheel (denser than Lenis pockets). */
const VISUAL_TICK_COUNT = 200;
const VISUAL_STEP_DEG = 360 / VISUAL_TICK_COUNT;

/** The slide a given pocket on the ring belongs to. */
export function slideIndexFor(pocket: number): number {
  return ((pocket % SLIDE_COUNT) + SLIDE_COUNT) % SLIDE_COUNT;
}

/** Pocket on the ring closest to `currentPocket` that carries `slideIndex`. */
export function nearestPocketForSlide(
  currentPocket: number,
  slideIndex: number,
): number {
  const currentNorm = ((currentPocket % WORD_COUNT) + WORD_COUNT) % WORD_COUNT;
  let best = slideIndex;
  let minDist = Infinity;
  for (let lap = 0; lap < WHEEL_REPEATS; lap++) {
    const pocket = slideIndex + lap * SLIDE_COUNT;
    if (pocket >= WORD_COUNT) break;
    let diff = pocket - currentNorm;
    if (diff > WORD_COUNT / 2) diff -= WORD_COUNT;
    if (diff < -WORD_COUNT / 2) diff += WORD_COUNT;
    const dist = Math.abs(diff);
    if (dist < minDist) {
      minDist = dist;
      best = pocket;
    }
  }
  return best;
}

const INNER_RADIUS_PX = 118;
const MIN_TICK_PX = 14;
const MAX_TICK_PX = 52;

/** Deterministic pseudo-random tick length in [0, 1] for pocket `i`. */
function tickLengthFactor(i: number): number {
  const x = Math.sin(i * 12.9898 + 78.233) * 43758.5453;
  return x - Math.floor(x);
}

/**
 * The wheel's motion surface. Rotation is written straight to the DOM rather
 * than rendered from state: the driver sets it from inside Lenis's rAF.
 */
export type TextWheelHandle = {
  /** Turn the ring to `degrees` (positive = clockwise). */
  setRotation: (degrees: number) => void;
};

/**
 * Tick-mark wheel pinned past the bottom-left corner: white radial dashes of
 * uneven length radiate from a center just off the left edge, so only the
 * right-hand arc peeks in. The ring is spun through the handle.
 */
export const TextWheel = forwardRef<
  TextWheelHandle,
  { className?: string }
>(function TextWheel({ className = "" }, ref) {
  const ringRef = useRef<SVGGElement>(null);

  const ticks = useMemo(
    () =>
      Array.from({ length: VISUAL_TICK_COUNT }, (_, i) => {
        const angleDeg = i * VISUAL_STEP_DEG;
        const angleRad = (angleDeg * Math.PI) / 180;
        const length =
          MIN_TICK_PX + tickLengthFactor(i) * (MAX_TICK_PX - MIN_TICK_PX);
        const innerR = INNER_RADIUS_PX;
        const outerR = innerR + length;
        return {
          x1: innerR * Math.cos(angleRad),
          y1: innerR * Math.sin(angleRad),
          x2: outerR * Math.cos(angleRad),
          y2: outerR * Math.sin(angleRad),
        };
      }),
    [],
  );

  useImperativeHandle(
    ref,
    () => ({
      setRotation(degrees) {
        const ring = ringRef.current;
        if (!ring) return;
        ring.style.transform = `rotate(${degrees}deg)`;
      },
    }),
    [],
  );

  const extent = RADIUS_PX + MAX_TICK_PX + 8;

  return (
    <div
      className={`pointer-events-none absolute z-0 size-0 select-none ${className}`}
      style={{ left: CENTER_OFFSET_PX, bottom: CENTER_BOTTOM_PX }}
      aria-hidden
    >
      <svg
        width={extent}
        height={extent}
        viewBox={`${-extent / 2} ${-extent / 2} ${extent} ${extent}`}
        className="absolute overflow-visible"
        style={{ left: -extent / 2, bottom: -extent / 2 }}
      >
        <g
          ref={ringRef}
          style={{
            transform: "rotate(0deg)",
            transformOrigin: "0px 0px",
            willChange: "transform",
          }}
        >
          {ticks.map((tick, i) => (
            <line
              key={i}
              x1={tick.x1}
              y1={tick.y1}
              x2={tick.x2}
              y2={tick.y2}
              stroke="#f4f4f5"
              strokeWidth={1}
              strokeLinecap="round"
            />
          ))}
        </g>
      </svg>
    </div>
  );
});
