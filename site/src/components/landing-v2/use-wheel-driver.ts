"use client";

import { type RefObject, useEffect, useRef, useState } from "react";
import type { TextWheelHandle } from "@/components/landing-v2/text-wheel";
import { createLenisWheel } from "@/components/landing-v2/wheel-lenis";

export type WheelDriver = {
  activeStep: number;
  /** Slide the wheel is easing toward once deceleration starts. */
  incomingSlide: number | null;
  moving: boolean;
  /** 0 at spin start; ramps during deceleration as the pocket is approached. */
  settleProgress: number;
  spinToIndex: (index: number) => void;
};

export function useWheelDriver(
  wheel: RefObject<TextWheelHandle | null>,
  enabled = true,
): WheelDriver {
  const [activeStep, setActiveStep] = useState(0);
  const [incomingSlide, setIncomingSlide] = useState<number | null>(null);
  const [moving, setMoving] = useState(false);
  const [settleProgress, setSettleProgress] = useState(0);
  const spinRef = useRef<(index: number) => void>(() => {});

  useEffect(() => {
    if (!enabled) return;
    const lenisWheel = createLenisWheel({
      onRotation: (degrees) => wheel.current?.setRotation(degrees),
      onMoving: () => {
        setMoving(true);
        setSettleProgress(0);
      },
      onLandingCommitted: (slideIndex) => {
        setIncomingSlide(slideIndex);
      },
      onSettleProgress: (progress) => {
        setSettleProgress(progress);
      },
      onSettled: (step) => {
        setActiveStep(step);
        setMoving(false);
        setIncomingSlide(null);
        setSettleProgress(1);
      },
    });
    spinRef.current = lenisWheel.spinToIndex;
    return () => {
      spinRef.current = () => {};
      lenisWheel.destroy();
    };
  }, [enabled, wheel]);

  return {
    activeStep,
    incomingSlide,
    moving,
    settleProgress,
    spinToIndex: (index: number) => spinRef.current(index),
  };
};
