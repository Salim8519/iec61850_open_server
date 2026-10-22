/** Where a symbol added from the Add menu goes (change 0176): the middle of what the canvas shows now,
 *  moved to the nearest free spot so it never lands on another symbol or outside the view. */
export type Box = { x: number; y: number; width: number; height: number };
export type Point = { x: number; y: number };

/** A card's size until React Flow has measured it (the SLD symbol card). */
export const NEW_SIZE = { width: 108, height: 130 };
const GAP = 20;
const GRID = 20;
const RINGS = 12;

const snap = (value: number) => Math.round(value / GRID) * GRID;

function overlaps(a: Box, b: Box): boolean {
  return a.x < b.x + b.width + GAP && b.x < a.x + a.width + GAP && a.y < b.y + b.height + GAP && b.y < a.y + a.height + GAP;
}

/** The top-left position for a new card centred near `center`: rings of grid steps around it, the first
 *  spot that touches no existing box; the centre itself when nothing is free nearby. */
export function freeSpot(boxes: readonly Box[], center: Point, size = NEW_SIZE): Point {
  const start = { x: snap(center.x - size.width / 2), y: snap(center.y - size.height / 2) };
  const stepX = size.width + GAP;
  const stepY = size.height + GAP;
  for (let ring = 0; ring <= RINGS; ring++) {
    for (let dy = -ring; dy <= ring; dy++) {
      for (let dx = -ring; dx <= ring; dx++) {
        if (Math.max(Math.abs(dx), Math.abs(dy)) !== ring) continue;
        const spot = { x: snap(start.x + dx * stepX), y: snap(start.y + dy * stepY) };
        if (!boxes.some(box => overlaps({ ...spot, ...size }, box))) return spot;
      }
    }
  }
  return start;
}
