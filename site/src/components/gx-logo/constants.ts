/**
 * Single source of truth for gx logo sizing.
 * Tune container (CSS rem) and camera together.
 */

export const GX_MESH_PATH = "/models/gx-chrome.glb";
export const GX_HEADER_MESH_PATH = "/models/gx-icon.glb";

// The Text3D fallback does not match the Blender mark; deploys should always use the checked-in GLBs.
export const USE_GX_LOGO_MESH = true;

function splinePill(textSize: number) {
  return {
    textSize,
    textHeight: textSize * 0.06,
    bevelThickness: textSize * 0.3,
    bevelSize: textSize * 0.14,
    smoothPasses: 2 as const,
  };
}

type CameraConfig = {
  position: [number, number, number];
  fov: number;
};

type ContainerConfig = {
  widthRem: number;
  heightRem: number;
  maxWidthRem?: number;
  /** Left-align the mark inside the canvas (footer). */
  markAlign?: "center" | "start";
  /** Nudge canvas horizontally (rem) after alignment — negative pulls left. */
  markInsetXRem?: number;
  /** Shift mark right inside the scene to avoid clipping bevels on the left. */
  markSceneOffsetX?: number;
};

export type LogoVariantConfig = ContainerConfig & {
  camera: CameraConfig;
  meshPath: string;
  /** gx = full mark (mesh or text); x = single-letter export for tiny favicons. */
  glyph: "gx" | "x";
  /** Fit width and height inside the camera square (icon export). */
  squareFit?: boolean;
  /** Padding inside the square view (0–1). */
  squareFitPadding?: number;
  /** Scene units — max axis after centering; header uses smaller fit. */
  targetMaxDimension: number;
  textSize: number;
  textHeight: number;
  bevelThickness: number;
  bevelSize: number;
  smoothPasses: number;
};

type VariantOptions = {
  glyph?: "gx" | "x";
  squareFit?: boolean;
  squareFitPadding?: number;
};

function defineVariant(
  container: ContainerConfig,
  camera: CameraConfig,
  textSize: number,
  targetMaxDimension: number,
  meshPath: string,
  options: VariantOptions = {},
): LogoVariantConfig {
  return {
    ...container,
    camera,
    meshPath,
    glyph: options.glyph ?? "gx",
    squareFit: options.squareFit,
    squareFitPadding: options.squareFitPadding,
    targetMaxDimension,
    ...splinePill(textSize),
  };
}

export const LOGO_VARIANTS = {
  header: defineVariant(
    { widthRem: 8.25, heightRem: 2.5 },
    { position: [0, 0, 2.55], fov: 28 },
    1.22,
    1.88,
    GX_HEADER_MESH_PATH,
  ),
  footer: defineVariant(
    {
      widthRem: 16,
      heightRem: 4.5,
      markAlign: "start",
      markSceneOffsetX: 0.75,
      markInsetXRem: -2.75,
    },
    { position: [0, 0, 2.55], fov: 28 },
    1.35,
    2.05,
    GX_HEADER_MESH_PATH,
  ),
  /** Square export — full gx mark with padding so letters are not clipped. */
  icon: defineVariant(
    { widthRem: 1, heightRem: 1 },
    { position: [0, 0, 2.55], fov: 28 },
    1.48,
    1.88,
    GX_HEADER_MESH_PATH,
    { squareFit: true, squareFitPadding: 0.86 },
  ),
  /** Tiny favicons — chrome "x" clipped from gx-icon.glb (same material as header). */
  iconX: defineVariant(
    { widthRem: 1, heightRem: 1 },
    { position: [0, 0, 2.55], fov: 28 },
    1.48,
    1.88,
    GX_HEADER_MESH_PATH,
    { glyph: "x", squareFit: true, squareFitPadding: 0.74 },
  ),
  hero: defineVariant(
    { widthRem: 24, heightRem: 14, maxWidthRem: 26 },
    { position: [0, 0, 3.05], fov: 32 },
    1.72,
    1.92,
    GX_MESH_PATH,
  ),
} as const satisfies Record<string, LogoVariantConfig>;

export type LogoVariant = keyof typeof LOGO_VARIANTS;

export type LogoTone = "chrome" | "graphite";

/** Icon export / design preview — 1024 matches capture output; 4096 cubemap was ~400MB per canvas. */
export const ICON_ENVIRONMENT_RESOLUTION = 1024;

/** Nav bar — 1024 keeps chrome sharp; canvas is small so GPU cost stays low. */
export const HEADER_ENVIRONMENT_RESOLUTION = 1024;

/** Hero is large but one canvas; 1024 keeps chrome without a second 2048 map in the header. */
export const HERO_ENVIRONMENT_RESOLUTION = 1024;

export function getLogoConfig(variant: LogoVariant = "header") {
  return LOGO_VARIANTS[variant];
}

export const HEADER_LOGO = LOGO_VARIANTS.header;
export const HERO_LOGO = LOGO_VARIANTS.hero;
