"use client";

import { useSyncExternalStore } from "react";

/**
 * Whether there is room on screen for the word wheel.
 *
 * The ring is anchored to the bottom-left corner and the copy column to the
 * top, so a short viewport closes the gap between them. Width matters too: the
 * column sits inside a centred `max-w-6xl`, so the narrower the window the
 * further left it starts, and the sooner it reaches the ring. Below these the
 * two would overlap, and the landing drops the wheel and lets the slides
 * scroll instead.
 *
 * Measured against the rendered ring rather than guessed — the widest title
 * reaches ~242px from the left edge, and the column bottom sits 478px down.
 */
const MIN_HEIGHT_BY_WIDTH: readonly { minWidth: number; minHeight: number }[] = [
  { minWidth: 1600, minHeight: 0 },
  { minWidth: 1440, minHeight: 700 },
  { minWidth: 1280, minHeight: 800 },
  // The wheel is desktop-only anyway; below this it is never shown.
  { minWidth: 1024, minHeight: 880 },
];

export function wheelFits(width: number, height: number): boolean {
  const rule = MIN_HEIGHT_BY_WIDTH.find(({ minWidth }) => width >= minWidth);
  return rule !== undefined && height >= rule.minHeight;
}

function subscribe(onChange: () => void): () => void {
  window.addEventListener("resize", onChange);
  return () => window.removeEventListener("resize", onChange);
}

const getSnapshot = () => wheelFits(window.innerWidth, window.innerHeight);
/** Rendered on the server, where there is no viewport: assume the wheel fits,
 * which is the common desktop case, and correct on hydration. */
const getServerSnapshot = () => true;

export function useWheelFits(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
