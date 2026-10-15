import { Vector3, type BufferGeometry } from "three";

function addNeighbor(adj: Set<number>[], a: number, b: number) {
  adj[a].add(b);
  adj[b].add(a);
}

/** Laplacian smooth — softens flat caps and creases on extruded text. */
export function laplacianSmoothGeometry(
  geometry: BufferGeometry,
  iterations = 2,
  lambda = 0.35,
) {
  const position = geometry.getAttribute("position");
  const index = geometry.index;
  if (!index) return;

  const count = position.count;
  const adj: Set<number>[] = Array.from({ length: count }, () => new Set());

  for (let i = 0; i < index.count; i += 3) {
    const a = index.getX(i);
    const b = index.getX(i + 1);
    const c = index.getX(i + 2);
    addNeighbor(adj, a, b);
    addNeighbor(adj, b, c);
    addNeighbor(adj, c, a);
  }

  const current = new Vector3();
  const average = new Vector3();
  const next = new Float32Array(count * 3);

  for (let iter = 0; iter < iterations; iter++) {
    for (let v = 0; v < count; v++) {
      average.set(0, 0, 0);
      const neighbors = adj[v];
      if (neighbors.size === 0) continue;

      for (const n of neighbors) {
        average.x += position.getX(n);
        average.y += position.getY(n);
        average.z += position.getZ(n);
      }
      average.multiplyScalar(1 / neighbors.size);

      current.fromBufferAttribute(position, v);
      current.lerp(average, lambda);
      next[v * 3] = current.x;
      next[v * 3 + 1] = current.y;
      next[v * 3 + 2] = current.z;
    }

    position.array.set(next);
    position.needsUpdate = true;
  }

  geometry.computeVertexNormals();
}
