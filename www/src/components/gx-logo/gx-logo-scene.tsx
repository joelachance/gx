"use client";

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useFrame, useThree } from "@react-three/fiber";
import {
  Center,
  Environment,
  Float,
  Lightformer,
  Text3D,
  useFont,
} from "@react-three/drei";
import {
  ACESFilmicToneMapping,
  SRGBColorSpace,
  type BufferGeometry,
  type Mesh,
  type WebGLRenderer,
} from "three";
import { Box3, Vector3, type Camera, type Group, type Object3D } from "three";

import {
  getLogoConfig,
  HEADER_ENVIRONMENT_RESOLUTION,
  HERO_ENVIRONMENT_RESOLUTION,
  ICON_ENVIRONMENT_RESOLUTION,
  USE_GX_LOGO_MESH,
  type LogoTone,
  type LogoVariant,
  type LogoVariantConfig,
} from "./constants";
import { GxGlbModel } from "./gx-glb-model";
import { laplacianSmoothGeometry } from "./round-geometry";

const BLOB_FONT = "/fonts/Blob-Regular.typeface.json";
const STUDIO_HDR = "/hdr/studio_small_03_1k.hdr";

useFont.preload(BLOB_FONT);

export type { LogoTone } from "./constants";

type GxLogoSceneProps = {
  variant?: LogoVariant;
  tone?: LogoTone;
  /** When false, no mouse parallax or float — used for icon PNG export. */
  interactive?: boolean;
  /** Hero bob on desktop — off on touch-only tracking. */
  heroFloat?: boolean;
  /** Touch: track finger only while pointer is down. */
  touchDragOnly?: boolean;
  /** Header / touch: repaint on interaction only (demand frameloop). */
  hoverDrivenMotion?: boolean;
  /** HDR cubemap face size — lower for icon export to save GPU memory. */
  environmentResolution?: number;
  /** Fires once the 3D mark is loaded and painted. */
  onReady?: () => void;
};

const MOTION_EPS = 0.002;

const _pivotBox = new Box3();
const _pivotCenter = new Vector3();
const _projectVec = new Vector3();

function clampPointer(value: number) {
  return Math.max(-1, Math.min(1, value));
}

/** Screen-space bounds of a 3D object — used for footer pointer normalization. */
function projectObjectBounds(
  object: Object3D,
  camera: Camera,
  canvas: HTMLCanvasElement,
) {
  _pivotBox.setFromObject(object);
  const rect = canvas.getBoundingClientRect();
  let minX = Infinity;
  let maxX = -Infinity;
  let minY = Infinity;
  let maxY = -Infinity;

  const { min, max } = _pivotBox;
  const corners: [number, number, number][] = [
    [min.x, min.y, min.z],
    [max.x, min.y, min.z],
    [min.x, max.y, min.z],
    [max.x, max.y, min.z],
    [min.x, min.y, max.z],
    [max.x, min.y, max.z],
    [min.x, max.y, max.z],
    [max.x, max.y, max.z],
  ];

  for (const [x, y, z] of corners) {
    _projectVec.set(x, y, z).project(camera);
    const sx = rect.left + (_projectVec.x * 0.5 + 0.5) * rect.width;
    const sy = rect.top + (-_projectVec.y * 0.5 + 0.5) * rect.height;
    minX = Math.min(minX, sx);
    maxX = Math.max(maxX, sx);
    minY = Math.min(minY, sy);
    maxY = Math.max(maxY, sy);
  }

  return {
    cx: (minX + maxX) / 2,
    cy: (minY + maxY) / 2,
    halfW: Math.max((maxX - minX) / 2, 1),
    halfH: Math.max((maxY - minY) / 2, 1),
  };
}

function syncLogoCenterPivot(
  group: Group,
  content: Group,
  pivotReady: { current: boolean },
) {
  if (pivotReady.current) return true;

  _pivotBox.setFromObject(content);
  if (_pivotBox.isEmpty()) return false;

  _pivotBox.getCenter(_pivotCenter);
  _pivotBox.getSize(_projectVec);
  if (_projectVec.lengthSq() < 1e-6) return false;

  group.position.copy(_pivotCenter);
  content.position.copy(_pivotCenter).multiplyScalar(-1);
  pivotReady.current = true;
  return true;
}

function SplineChrome({ tone = "chrome" }: { tone?: LogoTone }) {
  if (tone === "graphite") {
    return (
      <meshPhysicalMaterial
        color="#5a6575"
        metalness={0.95}
        roughness={0.18}
        clearcoat={0.65}
        clearcoatRoughness={0.08}
        envMapIntensity={1.15}
        ior={1.45}
      />
    );
  }

  return (
    <meshPhysicalMaterial
      color="#e9edf7"
      metalness={1}
      roughness={0.04}
      clearcoat={1}
      clearcoatRoughness={0.015}
      reflectivity={1}
      envMapIntensity={1.85}
      ior={1.45}
    />
  );
}

function SplineLighting({
  variant,
  tone = "chrome",
  environmentResolution,
}: {
  variant: LogoVariant;
  tone?: LogoTone;
  environmentResolution?: number;
}) {
  if (tone === "graphite") {
    return (
      <>
        <ambientLight intensity={0.5} />
        <directionalLight position={[8, 10, 12]} intensity={1.75} color="#ffffff" />
        <directionalLight position={[-10, 4, 8]} intensity={0.85} color="#e4e4e7" />
        <pointLight position={[0, 0, 10]} intensity={10} color="#fafafa" />
        <Environment
          resolution={environmentResolution ?? 2048}
          environmentIntensity={0.72}
          files={STUDIO_HDR}
          blur={0.9}
        >
          <Lightformer
            form="rect"
            intensity={2.2}
            color="#f4f4f5"
            rotation-x={Math.PI / 2}
            position={[0, 6, -1]}
            scale={[14, 10, 1]}
          />
          <Lightformer
            form="ring"
            intensity={1.6}
            color="#e4e4e7"
            rotation-y={Math.PI / 2}
            position={[7, 0, 2]}
            scale={5}
          />
        </Environment>
      </>
    );
  }

  return (
    <>
      <ambientLight intensity={0.45} />
      <directionalLight position={[8, 10, 12]} intensity={2.2} color="#ffffff" />
      <directionalLight position={[-10, 4, 8]} intensity={1.1} color="#d8e6ff" />
      <pointLight position={[0, 0, 10]} intensity={16} color="#ffffff" />
      <Environment
        resolution={environmentResolution ?? 2048}
        environmentIntensity={1.1}
        files={STUDIO_HDR}
        blur={1}
      >
        <Lightformer
          form="rect"
          intensity={4}
          color="#ffffff"
          rotation-x={Math.PI / 2}
          position={[0, 6, -1]}
          scale={[14, 10, 1]}
        />
        <Lightformer
          form="ring"
          intensity={3}
          color="#f5f8ff"
          rotation-y={Math.PI / 2}
          position={[7, 0, 2]}
          scale={5}
        />
        <Lightformer
          form="circle"
          intensity={2}
          color="#ffffff"
          position={[0, -5, 4]}
          scale={7}
        />
      </Environment>
    </>
  );
}

const MOUSE_SMOOTHING = {
  header: { pointer: 10, rotation: 7, yaw: 0.42, pitch: 0.3 },
  footer: { pointer: 10, rotation: 7, yaw: 0.42, pitch: 0.3 },
  icon: { pointer: 10, rotation: 7, yaw: 0.42, pitch: 0.3 },
  iconX: { pointer: 10, rotation: 7, yaw: 0.42, pitch: 0.3 },
  hero: { pointer: 12, rotation: 8, yaw: 0.55, pitch: 0.38 },
} as const satisfies Record<LogoVariant, { pointer: number; rotation: number; yaw: number; pitch: number }>;

function damp(current: number, target: number, lambda: number, delta: number) {
  return current + (target - current) * (1 - Math.exp(-lambda * delta));
}

/** Stop the render loop when the tab is hidden. */
function PauseWhenHidden({ continuous }: { continuous: boolean }) {
  const setFrameloop = useThree((state) => state.setFrameloop);
  const invalidate = useThree((state) => state.invalidate);

  useEffect(() => {
    const sync = () => {
      if (document.hidden) {
        setFrameloop("never");
      } else {
        setFrameloop(continuous ? "always" : "demand");
        invalidate();
      }
    };
    sync();
    document.addEventListener("visibilitychange", sync);
    return () => document.removeEventListener("visibilitychange", sync);
  }, [continuous, setFrameloop, invalidate]);

  return null;
}

/** Demand-mode canvases still need explicit invalidates after async assets load. */
function StaticLogoSync() {
  const invalidate = useThree((state) => state.invalidate);

  useEffect(() => {
    invalidate();
    const timeouts = [100, 300, 600].map((ms) =>
      window.setTimeout(() => invalidate(), ms),
    );
    return () => timeouts.forEach(clearTimeout);
  }, [invalidate]);

  return null;
}

function MouseLook({
  children,
  variant,
  touchDragOnly = false,
  hoverDrivenMotion = false,
}: {
  children: React.ReactNode;
  variant: LogoVariant;
  touchDragOnly?: boolean;
  hoverDrivenMotion?: boolean;
}) {
  const groupRef = useRef<Group>(null);
  const contentRef = useRef<Group>(null);
  const pivotReady = useRef(false);
  const lastPointer = useRef<PointerEvent | null>(null);
  const smoothPointer = useRef({ x: 0, y: 0 });
  const rotation = useRef({ x: 0, y: 0 });
  const globalPointer = useRef({ x: 0, y: 0 });
  const dragging = useRef(false);
  const [hovered, setHovered] = useState(false);
  const invalidate = useThree((state) => state.invalidate);
  const invalidateRef = useRef(invalidate);
  invalidateRef.current = invalidate;
  const settings = MOUSE_SMOOTHING[variant];
  const trackGlobally =
    variant === "hero" || variant === "header" || variant === "footer";
  const useLogoCenterPivot = variant === "footer";
  const trackRelativeToLogo = variant === "footer";

  useEffect(() => {
    if (!trackGlobally) return;

    const updatePointer = (event: PointerEvent) => {
      if (trackRelativeToLogo) {
        lastPointer.current = event;
        return;
      }

      globalPointer.current.x = (event.clientX / window.innerWidth) * 2 - 1;
      globalPointer.current.y = -(event.clientY / window.innerHeight) * 2 + 1;
    };

    const onPointerMove = (event: PointerEvent) => {
      if (touchDragOnly && !dragging.current) return;
      updatePointer(event);
      if (hoverDrivenMotion) invalidateRef.current();
    };

    const onPointerDown = () => {
      dragging.current = true;
      if (hoverDrivenMotion) invalidateRef.current();
    };

    const onPointerUp = () => {
      dragging.current = false;
      if (hoverDrivenMotion) invalidateRef.current();
    };

    window.addEventListener("pointermove", onPointerMove);
    if (touchDragOnly) {
      window.addEventListener("pointerdown", onPointerDown);
      window.addEventListener("pointerup", onPointerUp);
      window.addEventListener("pointercancel", onPointerUp);
    }

    return () => {
      window.removeEventListener("pointermove", onPointerMove);
      if (touchDragOnly) {
        window.removeEventListener("pointerdown", onPointerDown);
        window.removeEventListener("pointerup", onPointerUp);
        window.removeEventListener("pointercancel", onPointerUp);
      }
    };
  }, [trackGlobally, trackRelativeToLogo, touchDragOnly, hoverDrivenMotion]);

  useFrame((state, delta) => {
    const group = groupRef.current;
    const content = contentRef.current;
    if (!group || !content) return;

    if (
      useLogoCenterPivot &&
      syncLogoCenterPivot(group, content, pivotReady)
    ) {
      invalidate();
    }

    if (trackRelativeToLogo && lastPointer.current && pivotReady.current) {
      const { cx, cy, halfW, halfH } = projectObjectBounds(
        content,
        state.camera,
        state.gl.domElement,
      );
      const event = lastPointer.current;
      globalPointer.current.x = clampPointer((event.clientX - cx) / halfW);
      globalPointer.current.y = clampPointer(-((event.clientY - cy) / halfH));
    }

    const dt = Math.min(delta, 0.05);
    const { pointer } = state;
    const dragActive = !touchDragOnly || dragging.current;

    const pointerTargetX = trackGlobally
      ? dragActive
        ? globalPointer.current.x
        : 0
      : hovered
        ? pointer.x
        : 0;
    const pointerTargetY = trackGlobally
      ? dragActive
        ? globalPointer.current.y
        : 0
      : hovered
        ? pointer.y
        : 0;

    smoothPointer.current.x = damp(
      smoothPointer.current.x,
      pointerTargetX,
      settings.pointer,
      dt,
    );
    smoothPointer.current.y = damp(
      smoothPointer.current.y,
      pointerTargetY,
      settings.pointer,
      dt,
    );

    const targetY = smoothPointer.current.x * settings.yaw;
    const targetX = -smoothPointer.current.y * settings.pitch;

    rotation.current.y = damp(
      rotation.current.y,
      targetY,
      settings.rotation,
      dt,
    );
    rotation.current.x = damp(
      rotation.current.x,
      targetX,
      settings.rotation,
      dt,
    );

    group.rotation.y = rotation.current.y;
    group.rotation.x = rotation.current.x;

    if (!hoverDrivenMotion) return;

    const animating =
      (!trackGlobally && hovered) ||
      Math.abs(rotation.current.x) > MOTION_EPS ||
      Math.abs(rotation.current.y) > MOTION_EPS ||
      Math.abs(smoothPointer.current.x) > MOTION_EPS ||
      Math.abs(smoothPointer.current.y) > MOTION_EPS;
    if (animating) invalidate();
  });

  return (
    <group
      ref={groupRef}
      {...(!trackGlobally && {
        onPointerOver: (e) => {
          e.stopPropagation();
          setHovered(true);
          if (hoverDrivenMotion) invalidate();
        },
        onPointerOut: (e) => {
          e.stopPropagation();
          setHovered(false);
          if (hoverDrivenMotion) invalidate();
        },
      })}
    >
      <group ref={contentRef}>{children}</group>
    </group>
  );
}

function roundTextGeometry(geometry: BufferGeometry, passes: number) {
  laplacianSmoothGeometry(geometry, passes, 0.38);
}

function GxTextMark({
  config,
  tone = "chrome",
  onReady,
}: {
  config: LogoVariantConfig;
  tone?: LogoTone;
  onReady?: () => void;
}) {
  const meshRef = useRef<Mesh>(null);
  const fitRef = useRef<Group>(null);
  const label = config.glyph;

  useLayoutEffect(() => {
    const geometry = meshRef.current?.geometry as BufferGeometry | undefined;
    if (!geometry) return;
    roundTextGeometry(geometry, config.smoothPasses);
  }, [config]);

  useLayoutEffect(() => {
    const mesh = meshRef.current;
    const fit = fitRef.current;
    if (!config.squareFit || !mesh || !fit) return;

    const box = new Box3().setFromObject(mesh);
    const size = box.getSize(new Vector3());
    const halfFov = (config.camera.fov * Math.PI) / 360;
    const padding = config.squareFitPadding ?? 0.86;
    const viewPlane =
      2 * config.camera.position[2] * Math.tan(halfFov) * padding;
    const scale = Math.min(
      viewPlane / Math.max(size.x, 0.001),
      viewPlane / Math.max(size.y, 0.001),
    );
    fit.scale.setScalar(scale);
  }, [config]);

  useLayoutEffect(() => {
    if (!onReady) {
      return;
    }

    let cancelled = false;
    const frame = requestAnimationFrame(() => {
      if (!cancelled) {
        onReady();
      }
    });

    return () => {
      cancelled = true;
      cancelAnimationFrame(frame);
    };
  }, [onReady]);

  const text = (
    <Text3D
      ref={meshRef}
      font={BLOB_FONT}
      size={config.textSize}
      height={config.textHeight}
      curveSegments={72}
      bevelEnabled
      bevelThickness={config.bevelThickness}
      bevelSize={config.bevelSize}
      bevelOffset={0}
      bevelSegments={40}
      letterSpacing={label === "x" ? 0 : -0.01}
      smooth={0.06}
    >
      {label}
      <SplineChrome tone={tone} />
    </Text3D>
  );

  if (config.squareFit) {
    return (
      <Center>
        <group ref={fitRef}>{text}</group>
      </Center>
    );
  }

  if (config.markAlign === "start") {
    return text;
  }

  return <Center>{text}</Center>;
}

function GxLogoMark({
  config,
  tone = "chrome",
  onReady,
}: {
  config: LogoVariantConfig;
  tone?: LogoTone;
  onReady?: () => void;
}) {
  const mark = USE_GX_LOGO_MESH ? (
    <GxGlbModel config={config} tone={tone} onReady={onReady} />
  ) : (
    <GxTextMark config={config} tone={tone} onReady={onReady} />
  );

  if (config.markAlign === "start") {
    return (
      <group position={[config.markSceneOffsetX ?? 0, 0, 0]}>
        <Center left>{mark}</Center>
      </group>
    );
  }

  if (USE_GX_LOGO_MESH) {
    return mark;
  }

  return <Center>{mark}</Center>;
}

function defaultEnvironmentResolution(variant: LogoVariant) {
  if (variant === "icon" || variant === "iconX") {
    return ICON_ENVIRONMENT_RESOLUTION;
  }
  if (variant === "header" || variant === "footer") {
    return HEADER_ENVIRONMENT_RESOLUTION;
  }
  if (variant === "hero") return HERO_ENVIRONMENT_RESOLUTION;
  return 2048;
}

export function GxLogoScene({
  variant = "header",
  tone = "chrome",
  interactive = true,
  heroFloat = false,
  touchDragOnly = false,
  hoverDrivenMotion = false,
  environmentResolution,
  onReady,
}: GxLogoSceneProps) {
  const config = getLogoConfig(variant);
  const isHero = variant === "hero";
  const envResolution =
    environmentResolution ?? defaultEnvironmentResolution(variant);
  const mark = <GxLogoMark config={config} tone={tone} onReady={onReady} />;

  const content =
    interactive && heroFloat && isHero ? (
      <Float speed={1.4} rotationIntensity={0.015} floatIntensity={0.04}>
        {mark}
      </Float>
    ) : (
      mark
    );

  return (
    <>
      <SplineLighting
        variant={variant}
        tone={tone}
        environmentResolution={envResolution}
      />
      {interactive ? (
        <>
          <PauseWhenHidden continuous={isHero && !touchDragOnly} />
          <MouseLook
            variant={variant}
            touchDragOnly={touchDragOnly}
            hoverDrivenMotion={hoverDrivenMotion}
          >
            {content}
          </MouseLook>
        </>
      ) : (
        content
      )}
      {!interactive ? <StaticLogoSync /> : null}
    </>
  );
}

export function configureTxLogoRenderer(
  gl: WebGLRenderer,
  variant: LogoVariant = "header",
  tone: LogoTone = "chrome",
) {
  gl.outputColorSpace = SRGBColorSpace;
  gl.toneMapping = ACESFilmicToneMapping;
  gl.toneMappingExposure = tone === "graphite" ? 1.18 : 1.3;
}
