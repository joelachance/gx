import { BufferGeometry, Float32BufferAttribute, Vector3 } from "three";

/** World-space split between g and x in gx-icon.glb (after Blender rotation). */
const GLYPH_X_SPLIT_RATIO = 0.5;

/**
 * Keeps triangles on the x side of the gx mark. Uses any-vertex test so bevels
 * on the split line are not shaved off (centroid-only clipping cut the x stem).
 */
export function clipGeometryToGlyphX(
  geometry: BufferGeometry,
  splitRatio = GLYPH_X_SPLIT_RATIO,
): BufferGeometry {
  geometry.computeBoundingBox();
  const box = geometry.boundingBox;
  if (!box) return geometry;

  const splitX = box.min.x + (box.max.x - box.min.x) * splitRatio;
  const pos = geometry.attributes.position;
  const index = geometry.index;
  const v = new Vector3();
  const verts: number[] = [];

  const triCount = index ? index.count / 3 : pos.count / 3;
  for (let i = 0; i < triCount; i++) {
    const indices: number[] = [];
    let keep = false;
    for (let j = 0; j < 3; j++) {
      const vi = index ? index.getX(i * 3 + j) : i * 3 + j;
      indices.push(vi);
      v.fromBufferAttribute(pos, vi);
      if (v.x >= splitX) keep = true;
    }
    if (!keep) continue;

    for (const vi of indices) {
      v.fromBufferAttribute(pos, vi);
      verts.push(v.x, v.y, v.z);
    }
  }

  const clipped = new BufferGeometry();
  clipped.setAttribute("position", new Float32BufferAttribute(verts, 3));
  clipped.computeVertexNormals();
  return clipped;
}
