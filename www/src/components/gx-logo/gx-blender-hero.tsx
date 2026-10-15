"use client";

/**
 * Standalone hero gx blender / 3D chrome transition.
 * Preserved for reuse — the marketing hero no longer mounts this as the main logo.
 */
import { GxLogo, type GxLogoProps } from "./gx-logo";

export type GxBlenderHeroProps = Omit<GxLogoProps, "variant">;

export function GxBlenderHero({
  className,
  onReady,
  ...props
}: GxBlenderHeroProps) {
  return (
    <GxLogo
      variant="hero"
      className={className}
      onReady={onReady}
      {...props}
    />
  );
}
