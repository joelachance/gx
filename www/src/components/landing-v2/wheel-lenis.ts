import Lenis from "lenis";
import {
  SLIDE_COUNT,
  WHEEL_STEP_DEG,
  WORD_COUNT,
  slideIndexFor,
} from "@/components/landing-v2/text-wheel";

/**
 * Each scroll gesture:
 * - Spins the tick ring freely (multi-turn coast) for fortune-wheel feel
 * - Advances page content by exactly one adjacent slide (±1)
 * - Never exposes intermediate pages (committed slide only)
 */

const PX_PER_STEP = 18;
const LAP_PX = PX_PER_STEP * WORD_COUNT;
const WRAP_LAPS = 60;
const TRACK_PX = LAP_PX * WRAP_LAPS;
/** Full ring revolutions added on top of the +1 pocket page step. */
const COAST_TURNS = 2;

export type LenisWheelCallbacks = {
  onRotation: (rotation: number) => void;
  onSettled: (step: number) => void;
  onMoving: () => void;
  onLandingCommitted: (slideIndex: number) => void;
  onSettleProgress: (progress: number) => void;
};

export type LenisWheel = {
  destroy: () => void;
  spinToIndex: (slideIndex: number) => void;
};

function createVirtualScroller(): {
  wrapper: HTMLElement;
  content: HTMLElement;
} {
  const wrapper = document.createElement("div");
  wrapper.setAttribute("aria-hidden", "true");
  wrapper.style.cssText =
    "position:fixed;top:0;left:0;width:1px;height:1px;overflow-y:auto;" +
    "overflow-x:hidden;scrollbar-width:none;opacity:0;pointer-events:none;" +
    "z-index:-1;";
  const content = document.createElement("div");
  content.style.cssText = `width:1px;height:${TRACK_PX + 1}px;`;
  wrapper.appendChild(content);
  document.body.appendChild(wrapper);
  return { wrapper, content };
}

function rotationFor(scroll: number): number {
  return -(scroll / PX_PER_STEP) * WHEEL_STEP_DEG;
}

const COAST_DURATION_S = 1.55;
const CLICK_DURATION_S = 1.25;

export function createLenisWheel({
  onRotation,
  onSettled,
  onMoving,
  onLandingCommitted,
  onSettleProgress,
}: LenisWheelCallbacks): LenisWheel {
  const { wrapper, content } = createVirtualScroller();

  let moving = false;
  let settledPocket = 0;
  let committedLandingPocket: number | null = null;
  let committedSlide: number | null = null;
  let rolling = false;
  let coastFromScroll = 0;
  let coastDistance = 0;
  let coastStartedAt = 0;
  let coastDurationMs = COAST_DURATION_S * 1000;

  let lenis: Lenis;

  const settle = () => {
    if (!rolling) return;
    if (committedLandingPocket !== null) {
      settledPocket = committedLandingPocket;
    }
    const slide =
      committedSlide !== null ? committedSlide : slideIndexFor(settledPocket);
    rolling = false;
    committedLandingPocket = null;
    committedSlide = null;
    coastDistance = 0;
    onSettleProgress(1);
    onRotation(rotationFor(lenis.scroll));
    if (!moving) return;
    moving = false;
    onSettled(slide);
  };

  const beginCoast = (
    pocketDelta: number,
    toSlide: number,
    durationS: number,
  ) => {
    if (rolling || committedLandingPocket !== null) return;
    if (pocketDelta === 0) return;

    rolling = true;
    committedLandingPocket = settledPocket + pocketDelta;
    committedSlide = toSlide;
    coastFromScroll = lenis.animatedScroll;
    coastDistance = Math.abs(pocketDelta) * PX_PER_STEP;
    coastDurationMs = durationS * 1000;
    coastStartedAt = performance.now();

    if (!moving) {
      moving = true;
      onMoving();
    }
    onSettleProgress(0);
    onLandingCommitted(toSlide);

    lenis.scrollTo(coastFromScroll + pocketDelta * PX_PER_STEP, {
      duration: durationS,
      easing: (t) => 1 - Math.pow(1 - t, 3),
      force: true,
    });
  };

  const beginAdjacent = (dir: 1 | -1, durationS: number) => {
    const fromSlide = slideIndexFor(settledPocket);
    const toSlide = slideIndexFor(fromSlide + dir);
    const pocketDelta = dir * (1 + WORD_COUNT * COAST_TURNS);
    beginCoast(pocketDelta, toSlide, durationS);
  };

  lenis = new Lenis({
    wrapper,
    content,
    eventsTarget: window,
    infinite: true,
    lerp: 0.023,
    smoothWheel: true,
    syncTouch: false,
    prevent: () => false,
    virtualScroll: ({ deltaY }) => {
      if (rolling || committedLandingPocket !== null) return false;
      if (deltaY === 0) return false;
      beginAdjacent(deltaY > 0 ? 1 : -1, COAST_DURATION_S);
      return false;
    },
  });

  (window as Window & { __wheelLenis?: Lenis }).__wheelLenis = lenis;

  const unsubscribeScroll = lenis.on("scroll", () => {
    onRotation(rotationFor(lenis.scroll));
  });

  let frame = requestAnimationFrame(function raf(time: number) {
    lenis.raf(time);

    if (rolling && coastDistance > 0) {
      const traveled = Math.abs(lenis.animatedScroll - coastFromScroll);
      const byDistance = Math.min(1, traveled / coastDistance);
      const byTime = Math.min(
        1,
        Math.max(0, (performance.now() - coastStartedAt) / coastDurationMs),
      );
      // Prefer distance; fall back to time so we always complete the fade
      // even if Lenis reports odd remaining values mid-duration.
      const progress = Math.max(byDistance, byTime > 0.15 ? byTime : 0);
      onSettleProgress(progress);

      if (progress >= 0.995 || byTime >= 1) {
        settle();
      }
    }

    frame = requestAnimationFrame(raf);
  });

  const spinToIndex = (slideIndex: number) => {
    if (rolling || committedLandingPocket !== null) return;
    const currentSlide = slideIndexFor(settledPocket);
    let diff = slideIndex - currentSlide;
    if (diff > SLIDE_COUNT / 2) diff -= SLIDE_COUNT;
    if (diff < -SLIDE_COUNT / 2) diff += SLIDE_COUNT;
    if (diff === 0) return;
    const dir: 1 | -1 = diff > 0 ? 1 : -1;
    const pocketDelta = diff + dir * WORD_COUNT * COAST_TURNS;
    beginCoast(pocketDelta, slideIndex, CLICK_DURATION_S);
  };

  return {
    spinToIndex,
    destroy: () => {
      cancelAnimationFrame(frame);
      unsubscribeScroll();
      lenis.destroy();
      delete (window as Window & { __wheelLenis?: Lenis }).__wheelLenis;
      wrapper.remove();
    },
  };
};
