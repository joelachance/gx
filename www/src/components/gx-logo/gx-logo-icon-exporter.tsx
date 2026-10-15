"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type Dispatch,
  type SetStateAction,
} from "react";
import { useFrame, useThree } from "@react-three/fiber";
import type { WebGLRenderer } from "three";
import { Vector2 } from "three";

import { GxLogo } from "./gx-logo";
import { ICON_ENVIRONMENT_RESOLUTION, type LogoVariant } from "./constants";

const CAPTURE_SIZE = 1024;
/** Fewer frames after lowering icon env map from 4096 → 1024. */
const SETTLE_FRAMES = 48;

const FAVICON_X_MAX = 48;

export const ICON_EXPORT_SIZES = [
  { label: "favicon-16", size: 16 },
  { label: "favicon-32", size: 32 },
  { label: "favicon-48", size: 48 },
  { label: "apple-touch-icon", size: 180 },
  { label: "pwa-192", size: 192 },
  { label: "pwa-512", size: 512 },
  { label: "app-store", size: 1024 },
] as const;

export type IconExportBackground = "transparent" | "light" | "dark";

type IconCaptureVariant = Extract<LogoVariant, "icon" | "iconX">;

type IconCaptureJob = {
  variant: IconCaptureVariant;
  resolve: (canvas: HTMLCanvasElement) => void;
  reject: (reason?: unknown) => void;
};

function downloadDataUrl(dataUrl: string, filename: string) {
  const link = document.createElement("a");
  link.href = dataUrl;
  link.download = filename;
  link.click();
}

function compositeWithBackground(
  source: HTMLCanvasElement,
  size: number,
  background: IconExportBackground,
): string {
  const out = document.createElement("canvas");
  out.width = size;
  out.height = size;
  const ctx = out.getContext("2d");
  if (!ctx) return source.toDataURL("image/png");

  if (background === "light") {
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, size, size);
  } else if (background === "dark") {
    ctx.fillStyle = "#0a0a0a";
    ctx.fillRect(0, 0, size, size);
  }

  ctx.drawImage(source, 0, 0, size, size);
  return out.toDataURL("image/png");
}

function SceneCaptureBridge({
  onReady,
}: {
  onReady: (capture: () => Promise<HTMLCanvasElement>) => void;
}) {
  const { gl, scene, camera, invalidate } = useThree();
  const frames = useRef(0);
  const notified = useRef(false);

  const capture = useCallback(async () => {
    const renderer = gl as WebGLRenderer;
    const prevSize = new Vector2();
    const prevPixelRatio = renderer.getPixelRatio();
    renderer.getSize(prevSize);

    renderer.setPixelRatio(1);
    renderer.setSize(CAPTURE_SIZE, CAPTURE_SIZE, false);
    renderer.setClearColor(0x000000, 0);
    renderer.render(scene, camera);

    const canvas = document.createElement("canvas");
    canvas.width = CAPTURE_SIZE;
    canvas.height = CAPTURE_SIZE;
    canvas.getContext("2d")?.drawImage(renderer.domElement, 0, 0);

    renderer.setPixelRatio(prevPixelRatio);
    renderer.setSize(prevSize.x, prevSize.y, false);
    invalidate();

    return canvas;
  }, [gl, scene, camera, invalidate]);

  useFrame(() => {
    if (notified.current) return;
    frames.current += 1;
    if (frames.current >= SETTLE_FRAMES) {
      notified.current = true;
      onReady(capture);
      return;
    }
    // demand frameloop only paints on invalidate — keep stepping until env settles
    invalidate();
  });

  return null;
}

/** One short-lived WebGL context — avoids two 4k cubemaps at once. */
function EphemeralIconCapture({
  job,
  onCaptured,
}: {
  job: IconCaptureJob;
  onCaptured: () => void;
}) {
  const handleReady = useCallback(
    (capture: () => Promise<HTMLCanvasElement>) => {
      void capture()
        .then((canvas) => {
          job.resolve(canvas);
          onCaptured();
        })
        .catch(job.reject);
    },
    [job, onCaptured],
  );

  return (
    <div
      className="pointer-events-none fixed top-0 -left-[10000px] size-px overflow-hidden opacity-0"
      aria-hidden
    >
      <GxLogo
        key={job.variant}
        variant={job.variant}
        pixelSize={160}
        interactive={false}
        frameloop="always"
        preserveDrawingBuffer
        environmentResolution={ICON_ENVIRONMENT_RESOLUTION}
      >
        <SceneCaptureBridge onReady={handleReady} />
      </GxLogo>
    </div>
  );
}

function captureIconVariant(
  variant: IconCaptureVariant,
  setJob: Dispatch<SetStateAction<IconCaptureJob | null>>,
): Promise<HTMLCanvasElement> {
  return new Promise((resolve, reject) => {
    setJob({ variant, resolve, reject });
  });
}

const TRANSPARENT_CHECKER_LIGHT: CSSProperties = {
  backgroundColor: "#fafafa",
  backgroundImage: `
    linear-gradient(45deg, #e4e4e7 25%, transparent 25%),
    linear-gradient(-45deg, #e4e4e7 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #e4e4e7 75%),
    linear-gradient(-45deg, transparent 75%, #e4e4e7 75%)
  `,
  backgroundSize: "12px 12px",
  backgroundPosition: "0 0, 0 6px, 6px -6px, -6px 0",
};

function previewSurfaceStyle(
  background: IconExportBackground,
): CSSProperties | undefined {
  if (background === "light") return { backgroundColor: "#ffffff" };
  if (background === "dark") return { backgroundColor: "#0a0a0a" };
  return TRANSPARENT_CHECKER_LIGHT;
}

function IconPreviewImage({
  label,
  previewUrl,
  previewSize,
  loading,
  background,
}: {
  label: string;
  previewUrl: string | null;
  previewSize: number;
  loading: boolean;
  background: IconExportBackground;
}) {
  return (
    <div className="flex flex-col items-center gap-1.5">
      <p className="text-xs font-medium uppercase tracking-wide text-zinc-500">
        {label}
      </p>
      <div
        className="flex items-center justify-center"
        style={{ width: previewSize, height: previewSize, ...previewSurfaceStyle(background) }}
      >
        {previewUrl ? (
          <img
            src={previewUrl}
            alt=""
            width={previewSize}
            height={previewSize}
            className="block size-full object-contain"
          />
        ) : (
          <div
            className={`size-full ${loading ? "animate-pulse bg-zinc-200 dark:bg-zinc-800" : ""}`}
          />
        )}
      </div>
    </div>
  );
}

type GxLogoIconExporterProps = {
  compact?: boolean;
};

export function GxLogoIconExporter({ compact = false }: GxLogoIconExporterProps) {
  const [captureJob, setCaptureJob] = useState<IconCaptureJob | null>(null);
  const [previewTx, setPreviewTx] = useState<string | null>(null);
  const [previewX, setPreviewX] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [background, setBackground] =
    useState<IconExportBackground>("transparent");
  const finishCaptureJob = useCallback(() => {
    setCaptureJob(null);
  }, []);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const gxCanvas = await captureIconVariant("icon", setCaptureJob);
        if (cancelled) return;
        setPreviewTx(gxCanvas.toDataURL("image/png"));

        const xCanvas = await captureIconVariant("iconX", setCaptureJob);
        if (cancelled) return;
        setPreviewX(xCanvas.toDataURL("image/png"));
        setReady(true);
      } catch {
        /* strict-mode unmount or tab switch — effect re-runs */
      }
    })();

    return () => {
      cancelled = true;
      setCaptureJob((job) => {
        if (job) job.reject(new Error("icon capture cancelled"));
        return null;
      });
    };
  }, []);

  const runCapture = useCallback(
    (variant: IconCaptureVariant) => captureIconVariant(variant, setCaptureJob),
    [],
  );

  const exportSize = useCallback(
    async (size: number, label: string, source: HTMLCanvasElement) => {
      const prefix = size <= FAVICON_X_MAX ? "gx-x" : "gx";
      const suffix = background === "transparent" ? "" : `-${background}`;
      const dataUrl = compositeWithBackground(source, size, background);
      downloadDataUrl(dataUrl, `${prefix}-icon-${label}${suffix}.png`);
    },
    [background],
  );

  const exportAll = useCallback(async () => {
    setBusy(true);
    try {
      const gxSource = await runCapture("icon");
      for (const { label, size } of ICON_EXPORT_SIZES) {
        if (size <= FAVICON_X_MAX) continue;
        await exportSize(size, label, gxSource);
        await new Promise((r) => setTimeout(r, 80));
      }

      const xSource = await runCapture("iconX");
      for (const { label, size } of ICON_EXPORT_SIZES) {
        if (size > FAVICON_X_MAX) continue;
        await exportSize(size, label, xSource);
        await new Promise((r) => setTimeout(r, 80));
      }
    } finally {
      setBusy(false);
    }
  }, [exportSize, runCapture]);

  const exportOne = useCallback(
    async (size: number, label: string) => {
      setBusy(true);
      try {
        const variant: IconCaptureVariant =
          size <= FAVICON_X_MAX ? "iconX" : "icon";
        const source = await runCapture(variant);
        await exportSize(size, label, source);
      } finally {
        setBusy(false);
      }
    },
    [exportSize, runCapture],
  );

  const previewSize = compact ? 160 : 200;
  const rootGap = compact ? "gap-5" : "gap-8";
  const panelClass = compact
    ? "grid grid-cols-2 gap-3 border border-zinc-200 p-3 dark:border-zinc-800"
    : "grid grid-cols-2 gap-4 rounded-2xl border border-zinc-200 p-4 dark:border-zinc-800";

  const loadingTx = !previewTx && Boolean(captureJob?.variant === "icon");
  const loadingX = previewTx != null && !previewX && captureJob?.variant === "iconX";

  return (
    <div className={`mx-auto flex w-full max-w-lg flex-col ${rootGap}`}>
      {captureJob ? (
        <EphemeralIconCapture job={captureJob} onCaptured={finishCaptureJob} />
      ) : null}

      <div className={panelClass}>
        <IconPreviewImage
          label="gx (48px+)"
          previewUrl={previewTx}
          previewSize={previewSize}
          loading={loadingTx}
          background={background}
        />
        <IconPreviewImage
          label="x (≤48px)"
          previewUrl={previewX}
          previewSize={previewSize}
          loading={loadingX || (previewTx != null && !previewX && !ready)}
          background={background}
        />
      </div>
      <p className="text-center text-sm text-zinc-600 dark:text-zinc-400">
        {ready
          ? "Full gx for larger icons; chrome x only for 16–48px favicons."
          : "Rendering chrome previews…"}
      </p>

      <fieldset className="flex flex-col gap-1.5">
        <legend className="text-xs font-medium uppercase tracking-wide text-zinc-500">
          Background
        </legend>
        <div className="flex flex-wrap gap-1.5">
          {(
            [
              ["transparent", "Transparent"],
              ["light", "White"],
              ["dark", "Dark"],
            ] as const
          ).map(([value, label]) => (
            <label
              key={value}
              className="flex cursor-pointer items-center gap-1.5 border border-zinc-200 px-2.5 py-1 text-xs dark:border-zinc-800"
            >
              <input
                type="radio"
                name="icon-bg"
                value={value}
                checked={background === value}
                onChange={() => setBackground(value)}
              />
              {label}
            </label>
          ))}
        </div>
      </fieldset>

      <div className="flex flex-col gap-2">
        <button
          type="button"
          disabled={!ready || busy}
          onClick={() => void exportAll()}
          className="bg-zinc-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
        >
          Download all PNG sizes
        </button>
        <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-3">
          {ICON_EXPORT_SIZES.map(({ label, size }) => (
            <button
              key={label}
              type="button"
              disabled={!ready || busy}
              onClick={() => void exportOne(size, label)}
              className="border border-zinc-200 px-2.5 py-1.5 text-left text-sm hover:bg-zinc-50 disabled:opacity-50 dark:border-zinc-800 dark:hover:bg-zinc-900"
            >
              <span className="font-medium text-zinc-900 dark:text-zinc-50">
                {size}×{size}
              </span>
              <span className="block text-xs text-zinc-500">
                {label}
                {size <= FAVICON_X_MAX ? " · x" : " · gx"}
              </span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
