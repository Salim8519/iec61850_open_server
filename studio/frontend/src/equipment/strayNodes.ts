import type { EquipmentNode } from '../api/types.ts';

/** A symbol this far (flow units) from every other symbol is a stray (change 0180): before 0176 Add placed
 *  new symbols by node count, so one could land 20 000 units below the drawing. Fit view leaves strays
 *  out, so the drawing never disappears off screen; the strays are named so the user can bring them back. */
export const STRAY_GAP = 3000;

/** Ids of symbols with no other symbol within STRAY_GAP, when the rest is the large majority. */
export function strayIds(nodes: readonly EquipmentNode[], gap = STRAY_GAP): string[] {
  if (nodes.length < 3) return [];
  const xs = nodes.map(node => node.position.x);
  const ys = nodes.map(node => node.position.y);
  const strays: string[] = [];
  nodes.forEach((node, index) => {
    for (let other = 0; other < nodes.length; other++) {
      if (other !== index && Math.abs(xs[other] - xs[index]) < gap && Math.abs(ys[other] - ys[index]) < gap) return;
    }
    strays.push(node.id);
  });
  // Only a few lone symbols are strays; a drawing of scattered single items is left as it is.
  return strays.length * 4 < nodes.length ? strays : [];
}

/** Fit view options that fit only the drawing (all symbols but the strays). */
export function fitOptionsFor(nodes: readonly EquipmentNode[], base: { maxZoom: number; padding: number }) {
  const strays = new Set(strayIds(nodes));
  if (!strays.size) return base;
  return { ...base, nodes: nodes.filter(node => !strays.has(node.id)).map(node => ({ id: node.id })) };
}

/** Where to put a stray so it sits just below the drawing's left edge. */
export function besideDrawing(nodes: readonly EquipmentNode[], strays: readonly string[]): { x: number; y: number } {
  const skip = new Set(strays);
  const rest = nodes.filter(node => !skip.has(node.id));
  if (!rest.length) return { x: 0, y: 0 };
  return { x: Math.min(...rest.map(node => node.position.x)), y: Math.max(...rest.map(node => node.position.y)) + 200 };
}
