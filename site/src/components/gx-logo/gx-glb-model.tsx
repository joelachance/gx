"use client";

import { useLayoutEffect, useMemo } from "react";
import { Center, useGLTF } from "@react-three/drei";
import {
  Box3,
  MeshPhysicalMaterial,
  Vector3,
  type BufferGeometry,
  type Mesh,
} from "three";

import {
  GX_HEADER_MESH_PATH,
  GX_MESH_PATH,
  type LogoTone,
  type LogoVariantConfig,
} from "./constants";
import { clipGeometryToGlyphX } from "./gx-mesh-glyph";

/** Correct Blender export orientation until rotation is applied in Blender. */
const BLENDER_MESH_ROTATION: [number, number, number] = [Math.PI, 0, 0];

function createChromeMaterial(tone: LogoTone = "chrome") {
  if (tone === "graphite") {
    return new MeshPhysicalMaterial({
      color: "#5a6575",
      metalness: 0.95,
      roughness: 0.18,
      clearcoat: 0.65,
      clearcoatRoughness: 0.08,
      envMapIntensity: 1.15,
    });
  }

  return new MeshPhysicalMaterial({
    color: "#e9edf7",
    metalness: 1,
    roughness: 0.04,
    clearcoat: 1,
    clearcoatRoughness: 0.015,
    envMapIntensity: 1.85,
  });
}

type GxGlbModelProps = {
  config: LogoVariantConfig;
  tone?: LogoTone;
  onReady?: () => void;
};

export function GxGlbModel({ config, tone = "chrome", onReady }: GxGlbModelProps) {
  const { scene } = useGLTF(config.meshPath);
  const { model, scale } = useMemo(() => {
    const clone = scene.clone(true);
    clone.rotation.set(...BLENDER_MESH_ROTATION);
    clone.updateMatrixWorld(true);

    if (config.glyph === "x") {
      clone.traverse((child) => {
        const mesh = child as Mesh;
        if (!mesh.isMesh) return;
        mesh.geometry = clipGeometryToGlyphX(
          mesh.geometry as BufferGeometry,
        );
      });
    }

    const box = new Box3().setFromObject(clone);
    const center = box.getCenter(new Vector3());
    clone.position.sub(center);
    clone.updateMatrixWorld(true);

    const sizedBox = new Box3().setFromObject(clone);
    const size = sizedBox.getSize(new Vector3());
    const variantScale = config.textSize / 1.48;

    let fitScale: number;
    if (config.squareFit) {
      const halfFov = (config.camera.fov * Math.PI) / 360;
      const padding = config.squareFitPadding ?? 0.86;
      const viewPlane =
        2 * config.camera.position[2] * Math.tan(halfFov) * padding;
      const scaleX = viewPlane / Math.max(size.x, 0.001);
      const scaleY = viewPlane / Math.max(size.y, 0.001);
      // Fill the square at full size — variantScale < 1 was shrinking gx vs x.
      fitScale = Math.min(scaleX, scaleY);
    } else {
      const maxDim = Math.max(size.x, size.y, size.z, 0.001);
      fitScale = (config.targetMaxDimension / maxDim) * variantScale;
    }

    return { model: clone, scale: fitScale };
  }, [
    scene,
    config.textSize,
    config.targetMaxDimension,
    config.squareFit,
    config.camera,
    config.glyph,
    config.squareFitPadding,
  ]);

  useLayoutEffect(() => {
    const chrome = createChromeMaterial(tone);
    model.traverse((child) => {
      const mesh = child as Mesh;
      if (!mesh.isMesh) return;
      mesh.material = chrome;
      mesh.castShadow = false;
      mesh.receiveShadow = false;
    });

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
  }, [model, tone, onReady]);

  const content = <primitive object={model} scale={scale} />;

  if (config.markAlign === "start") {
    return content;
  }

  return <Center>{content}</Center>;
}

useGLTF.preload(GX_MESH_PATH);
useGLTF.preload(GX_HEADER_MESH_PATH);
