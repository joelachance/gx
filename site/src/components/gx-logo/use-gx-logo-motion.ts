"use client";

import { useCallback, useState } from "react";
import type { WebGLRenderer } from "three";

function isSoftwareWebGlRenderer(gl: WebGLRenderer): boolean {
  const context = gl.getContext();
  const debugInfo = context.getExtension("WEBGL_debug_renderer_info");
  if (!debugInfo) return false;

  const renderer = context.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
  return (
    typeof renderer === "string" &&
    /swiftshader|llvmpipe|software|basic render/i.test(renderer)
  );
}

function prefersReducedMotion(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/** Heuristic device check before enabling logo mouse motion. */
export function assessTxLogoMotionCapability(): boolean {
  if (typeof window === "undefined") return false;

  if (prefersReducedMotion()) return false;
  if (window.matchMedia("(hover: none)").matches) return false;

  const cores = navigator.hardwareConcurrency ?? 4;
  if (cores < 4) return false;

  const memory = (navigator as Navigator & { deviceMemory?: number }).deviceMemory;
  if (memory != null && memory < 4) return false;

  return true;
}

/** Touch devices: hero logo follows finger while dragging (lighter than full motion). */
export function assessTxLogoTouchTracking(): boolean {
  if (typeof window === "undefined") return false;
  if (prefersReducedMotion()) return false;
  return window.matchMedia("(hover: none)").matches;
}

export function useTxLogoMotion(requested: boolean) {
  const [motionEnabled, setMotionEnabled] = useState(
    () => requested && assessTxLogoMotionCapability(),
  );
  const [gpuBlocked, setGpuBlocked] = useState(false);

  const disableMotion = useCallback((gl: WebGLRenderer) => {
    if (!isSoftwareWebGlRenderer(gl)) return;
    setGpuBlocked(true);
    setMotionEnabled(false);
  }, []);

  const touchTrackingEnabled =
    requested && !gpuBlocked && assessTxLogoTouchTracking();

  return {
    motionEnabled: requested && motionEnabled && !gpuBlocked,
    touchTrackingEnabled,
    disableMotion,
  };
}
