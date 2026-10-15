"use client";

import { Suspense } from "react";
import { Canvas } from "@react-three/fiber";
import type { WebGLRenderer } from "three";
import {
  getLogoConfig,
  type LogoTone,
  type LogoVariant,
} from "./constants";
import { configureTxLogoRenderer, GxLogoScene } from "./gx-logo-scene";
import { useTxLogoMotion } from "./use-gx-logo-motion";

export type { LogoTone } from "./constants";

export type GxLogoProps = {
  variant?: LogoVariant;
  tone?: LogoTone;
  className?: string;
  children?: React.ReactNode;
  /** Fixed square render size in CSS pixels (for icon export). */
  pixelSize?: number;
  /** Multiplies the variant's container size; the mark scales with it. */
  scale?: number;
  /** Mouse parallax and hero float. Off for icon export. */
  interactive?: boolean;
  /** Required for canvas.toDataURL() capture. */
  preserveDrawingBuffer?: boolean;
  /** HDR cubemap face size (icon export — keep ≤1024 to limit GPU memory). */
  environmentResolution?: number;
  /** Override frameloop (icon capture uses `always` until SceneCaptureBridge finishes). */
  frameloop?: "always" | "demand" | "never";
  onGlReady?: (gl: WebGLRenderer) => void;
  /** Fires once the 3D mark is loaded and painted. */
  onReady?: () => void;
};

export function GxLogo({
  variant = "header",
  tone = "chrome",
  className,
  pixelSize,
  scale = 1,
  interactive = true,
  preserveDrawingBuffer = false,
  environmentResolution,
  frameloop: frameloopProp,
  onGlReady,
  onReady,
  children,
}: GxLogoProps) {
  const config = getLogoConfig(variant);
  const isHero = variant === "hero";
  const { motionEnabled, touchTrackingEnabled, disableMotion } =
    useTxLogoMotion(interactive);
  const pointerTrackingEnabled = motionEnabled || (isHero && touchTrackingEnabled);
  const isSquare =
    variant === "icon" || variant === "iconX" || pixelSize != null;
  const ariaLabel = variant === "iconX" ? "x" : "gx";
  const isHeader = variant === "header" || variant === "footer";
  /** Header is a small canvas — allow Retina DPR without the hero's continuous loop cost. */
  const dpr =
    isHeader
      ? ([1, 2] as [number, number])
      : !motionEnabled || variant === "hero"
        ? 1
        : pixelSize && (variant === "icon" || variant === "iconX")
          ? 2
          : 1;
  const antialias = isHeader || isHero || motionEnabled;
  /** Header/footer: global pointer + demand frameloop. Hero desktop: always; touch: demand. */
  const hoverDrivenMotion =
    (isHeader && motionEnabled) || (isHero && touchTrackingEnabled);
  const canvasFrameloop =
    frameloopProp ??
    (motionEnabled && !hoverDrivenMotion ? "always" : "demand");

  return (
    <div
      role="img"
      aria-label={ariaLabel}
      className={[
        "block shrink-0",
        config.markAlign === "start" ? "overflow-visible" : "",
        pointerTrackingEnabled ? "cursor-pointer" : "",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
      style={{
        width: pixelSize ?? (isHero ? "100%" : `${config.widthRem * scale}rem`),
        height: pixelSize ?? `${config.heightRem * scale}rem`,
        maxWidth:
          !pixelSize && config.maxWidthRem
            ? `${config.maxWidthRem * scale}rem`
            : undefined,
        aspectRatio: isSquare ? "1" : undefined,
        marginLeft:
          config.markInsetXRem !== undefined
            ? `${config.markInsetXRem * scale}rem`
            : undefined,
      }}
    >
      <Canvas
        frameloop={canvasFrameloop}
        camera={{
          position: config.camera.position,
          fov: config.camera.fov,
        }}
        gl={{
          alpha: true,
          antialias,
          powerPreference:
            motionEnabled && isHero ? "high-performance" : "low-power",
          preserveDrawingBuffer,
        }}
        dpr={dpr}
        onCreated={({ gl, camera, invalidate }) => {
          configureTxLogoRenderer(gl, variant, tone);
          camera.lookAt(0, 0, 0);
          disableMotion(gl);
          invalidate();
          onGlReady?.(gl);
        }}
        style={{ width: "100%", height: "100%", display: "block" }}
      >
        <Suspense fallback={null}>
          <GxLogoScene
            variant={variant}
            tone={tone}
            interactive={pointerTrackingEnabled}
            heroFloat={motionEnabled && isHero}
            touchDragOnly={touchTrackingEnabled && isHero}
            hoverDrivenMotion={hoverDrivenMotion}
            environmentResolution={environmentResolution}
            onReady={onReady}
          />
          {children}
        </Suspense>
      </Canvas>
    </div>
  );
}
