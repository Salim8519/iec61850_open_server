import type { EquipmentNode } from '../api/types';

/** Geometry of the SLD signal layers (spec signal-layers). Since ADR-0112 IEDs are drawn only as
 * IED components (diagram nodes the owner places); the old "G" corner marks and the left name-tag
 * column are gone. Nothing here is saved. */
export type Box = { id: string; x: number; y: number; width: number; height: number };

/** The IED components of the drawing, by IED name and by IED id (one per IED, Go enforces it). */
export type Components = { byName: Map<string, Box>; byId: Map<string, Box> };

export function componentBoxes(nodes: EquipmentNode[], boxes: Box[], iedNames: Record<string, string>): Components {
  const byNode = new Map(boxes.map(box => [box.id, box]));
  const byName = new Map<string, Box>(), byId = new Map<string, Box>();
  for (const node of nodes) {
    const iedId = node.data.kind === 'IED' ? node.data.ied_id : null;
    const box = byNode.get(node.id);
    if (!iedId || !box || byId.has(iedId)) continue;
    byId.set(iedId, box);
    if (iedNames[iedId]) byName.set(iedNames[iedId], box);
  }
  return { byName, byId };
}

/** Where a GOOSE dot from a sender with no place on the drawing (outside the project, or no IED
 * component and no trigger) starts: a fixed screen distance left of and above the receiver's IED
 * component (`scale` is tagScale), so it starts in view at every zoom (change 0102). */
export const ENTRY_DISTANCE = 220;
export function edgePoint(to: { x: number; y: number }, scale = 1) {
  return { x: to.x - ENTRY_DISTANCE * scale, y: to.y - 40 * scale };
}

/** Chips, dots and lines grow back toward screen size when zoomed out, between 1× and 3×. */
export function tagScale(zoom: number): number {
  return Math.min(3, Math.max(1, 0.9 / (zoom || 1)));
}

/** A gentle curve for the dot that flies in; returns the SVG path and the point at fraction t, so
 * a pulse dot can move along it with a transform only. */
export function linkCurve(from: { x: number; y: number }, to: { x: number; y: number }) {
  const bend = Math.min(80, Math.hypot(to.x - from.x, to.y - from.y) / 4);
  const control = { x: (from.x + to.x) / 2, y: Math.min(from.y, to.y) - bend };
  const at = (t: number) => ({
    x: (1 - t) ** 2 * from.x + 2 * (1 - t) * t * control.x + t ** 2 * to.x,
    y: (1 - t) ** 2 * from.y + 2 * (1 - t) * t * control.y + t ** 2 * to.y,
  });
  return { path: `M ${from.x} ${from.y} Q ${control.x} ${control.y} ${to.x} ${to.y}`, at };
}
