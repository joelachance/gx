"use client";

import {
  HIGHLIGHT_COLOR,
  NAV_BOTTOM_PX,
  NAV_LIST_GAP_PX,
  CENTER_OFFSET_PX,
  RADIUS_PX,
  SLIDE_NAMES,
} from "@/components/landing-v2/text-wheel";

const NAV_FONT_SIZE_PX = 12;
const NAV_LINE_HEIGHT_PX = 12;
const NAV_WHITE = "#f4f4f5";

/** Blend white (t=0) → orange (t=1). No gray in between. */
function mixOrangeWhite(t: number): string {
  const clamped = Math.min(1, Math.max(0, t));
  const r = Math.round(244 + (234 - 244) * clamped);
  const g = Math.round(244 + (88 - 244) * clamped);
  const b = Math.round(245 + (12 - 245) * clamped);
  return `rgb(${r}, ${g}, ${b})`;
}

type WheelNavProps = {
  activeSlide: number;
  incomingSlide: number | null;
  moving: boolean;
  settleProgress: number;
  onSlideClick: (slideIndex: number) => void;
};

export function WheelNav({
  activeSlide,
  incomingSlide,
  moving,
  settleProgress,
  onSlideClick,
}: WheelNavProps) {
  const listLeft = CENTER_OFFSET_PX + RADIUS_PX + NAV_LIST_GAP_PX;

  function labelColor(index: number): string {
    if (!moving) {
      return index === activeSlide ? HIGHLIGHT_COLOR : NAV_WHITE;
    }
    if (index === incomingSlide) {
      return mixOrangeWhite(settleProgress);
    }
    if (index === activeSlide) {
      return mixOrangeWhite(incomingSlide === null ? 0 : 1 - settleProgress);
    }
    return NAV_WHITE;
  }

  return (
    <nav
      className="pointer-events-auto absolute z-0 select-none"
      style={{
        left: listLeft,
        bottom: NAV_BOTTOM_PX,
        fontSize: NAV_FONT_SIZE_PX,
        lineHeight: `${NAV_LINE_HEIGHT_PX}px`,
      }}
      aria-label="Slides"
    >
      <ol
        className="m-0 p-0"
        style={{ margin: 0, padding: 0, listStyle: "none" }}
      >
        {SLIDE_NAMES.map((name, index) => (
          <li
            key={name}
            style={{
              margin: 0,
              padding: 0,
              height: NAV_LINE_HEIGHT_PX,
              lineHeight: `${NAV_LINE_HEIGHT_PX}px`,
            }}
          >
            <button
              type="button"
              onClick={() => onSlideClick(index)}
              aria-current={
                !moving && index === activeSlide ? "true" : undefined
              }
              className="m-0 block cursor-pointer border-0 bg-transparent p-0 text-left tracking-[0.08em] transition-colors duration-200"
              style={{
                margin: 0,
                padding: 0,
                height: NAV_LINE_HEIGHT_PX,
                lineHeight: `${NAV_LINE_HEIGHT_PX}px`,
                fontSize: NAV_FONT_SIZE_PX,
                color: labelColor(index),
              }}
            >
              {name}
            </button>
          </li>
        ))}
      </ol>
    </nav>
  );
}
