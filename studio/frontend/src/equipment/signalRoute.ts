import type { Box } from './signalLayout.ts';

/** "Smart lines" for GOOSE on the SLD (ADR-0103): pure orthogonal routes that never pass through
 * an equipment box. A route leaves its origin sideways into the nearest free vertical gap, climbs
 * to one shared lane above the drawing, runs along it and drops down the receiver's free gap.
 * Every receiver of one change shares the climb and the lane, so they read as one trunk. */
export type Point = { x: number; y: number };
/** `box`: the equipment the end belongs to; a route may touch only its own end boxes.
 * `edge`: start on the box's side (the symbol itself), not at `point`.
 * `up`: leave straight up from `point` when that is free (an IED component's top, ADR-0112). */
export type RouteEnd = { point: Point; box: Box | null; edge?: boolean; up?: boolean };
export type Route = { points: Point[]; path: string; length: number; clear: boolean; at: (t: number) => Point };
export type RouteOptions = { clearance?: number; margin?: number; radius?: number };

export const ROUTE_CLEARANCE = 8;
export const LANE_MARGIN = 36;
export const CORNER_RADIUS = 8;

/** The shared lane: `margin` above the topmost box (and above both route ends). */
export function laneY(boxes: Box[], margin = LANE_MARGIN, ends: Point[] = []): number {
  const tops = [...boxes.map(box => box.y), ...ends.map(point => point.y)];
  return (tops.length ? Math.min(...tops) : 0) - margin;
}

const inside = (value: number, low: number, high: number) => value > low && value < high;

/** True when the axis-aligned segment a-b enters `box` grown by `pad`. */
export function segmentHits(a: Point, b: Point, box: Box, pad = 0): boolean {
  const left = box.x - pad, right = box.x + box.width + pad, top = box.y - pad, bottom = box.y + box.height + pad;
  if (a.x === b.x) return inside(a.x, left, right) && Math.max(a.y, b.y) > top && Math.min(a.y, b.y) < bottom;
  if (a.y === b.y) return inside(a.y, top, bottom) && Math.max(a.x, b.x) > left && Math.min(a.x, b.x) < right;
  return true;
}

/** Boxes (other than `ignore`) that a polyline passes through. */
export function crossings(points: Point[], boxes: Box[], ignore: (string | undefined)[] = [], pad = 0): Box[] {
  const skip = new Set(ignore.filter(Boolean));
  return boxes.filter(box => !skip.has(box.id)
    && points.slice(1).some((point, index) => segmentHits(points[index], point, box, pad)));
}

const free = (a: Point, b: Point, boxes: Box[], skip: Set<string>, pad: number) =>
  boxes.every(box => skip.has(box.id) || !segmentHits(a, b, box, pad));

/** Vertical gaps worth trying: just outside every box edge, nearest to `x` first. */
function columns(x: number, boxes: Box[], clearance: number, toward: number): number[] {
  const all = boxes.flatMap(box => [box.x - clearance, box.x + box.width + clearance]);
  return [...new Set(all)].sort((a, b) => Math.abs(a - x) - Math.abs(b - x) || Math.sign(b - a) * toward);
}

/** From the origin up to the lane: [start, (gap, start.y), (gap, lane)], or straight up when free. */
export function exitPath(from: RouteEnd, boxes: Box[], lane: number, toward = 1, clearance = ROUTE_CLEARANCE): Point[] {
  const box = from.box;
  const skip = new Set(box ? [box.id] : []);
  const pad = clearance / 2;
  const starts = (gap: number): Point => !box || !from.edge ? from.point
    : { x: gap < box.x + box.width / 2 ? box.x : box.x + box.width, y: box.y + box.height / 2 };
  if ((!box || from.up) && free(from.point, { x: from.point.x, y: lane }, boxes, skip, pad)) return [from.point, { x: from.point.x, y: lane }];
  const origin = box ? { x: box.x + box.width / 2, y: from.point.y } : from.point;
  for (const gap of columns(origin.x, boxes, clearance, toward)) {
    if (box && inside(gap, box.x - clearance + 0.5, box.x + box.width + clearance - 0.5)) continue;
    const start = starts(gap);
    const corner = { x: gap, y: start.y };
    const top = { x: gap, y: lane };
    if (free(start, corner, boxes, skip, pad) && free(corner, top, boxes, skip, pad)) return [start, corner, top];
  }
  // Nothing free beside it: straight up from its top (reported as not clear by routeLine).
  const start = box && from.edge ? { x: box.x + box.width / 2, y: box.y } : from.point;
  return [start, { x: start.x, y: lane }];
}

/** From the lane down to the receiver: straight down when free, else down a gap beside it. */
export function dropPath(to: RouteEnd, boxes: Box[], lane: number, clearance = ROUTE_CLEARANCE): Point[] {
  const skip = new Set(to.box ? [to.box.id] : []);
  const pad = clearance / 2;
  const end = to.point;
  if (free({ x: end.x, y: lane }, end, boxes, skip, pad)) return [{ x: end.x, y: lane }, end];
  const toward = to.box ? 1 : -1;
  for (const gap of columns(end.x, boxes, clearance, toward)) {
    const top = { x: gap, y: lane };
    const corner = { x: gap, y: end.y };
    if (free(top, corner, boxes, skip, pad) && free(corner, end, boxes, skip, pad)) return [top, corner, end];
  }
  return [{ x: end.x, y: lane }, end];
}

/** Drop repeated points and middle points of straight runs; a turn back (down into an IED
 * component and up again) is kept, so the route really passes that component. */
export function simplify(points: Point[]): Point[] {
  const unique = points.filter((point, index) => index === 0 || point.x !== points[index - 1].x || point.y !== points[index - 1].y);
  return unique.filter((point, index) => {
    if (index === 0 || index === unique.length - 1) return true;
    const a = unique[index - 1], b = unique[index + 1];
    return !((a.x === point.x && point.x === b.x && (point.y - a.y) * (b.y - point.y) >= 0)
      || (a.y === point.y && point.y === b.y && (point.x - a.x) * (b.x - point.x) >= 0));
  });
}

const distance = (a: Point, b: Point) => Math.hypot(b.x - a.x, b.y - a.y);

/** SVG path with each corner rounded by up to `radius` (never more than half a leg). */
export function roundedPath(points: Point[], radius = CORNER_RADIUS): string {
  if (!points.length) return '';
  const parts = [`M ${points[0].x} ${points[0].y}`];
  for (let index = 1; index < points.length - 1; index++) {
    const [a, p, b] = [points[index - 1], points[index], points[index + 1]];
    const r = Math.min(radius, distance(a, p) / 2, distance(p, b) / 2);
    const inX = p.x - Math.sign(p.x - a.x) * r, inY = p.y - Math.sign(p.y - a.y) * r;
    const outX = p.x + Math.sign(b.x - p.x) * r, outY = p.y + Math.sign(b.y - p.y) * r;
    parts.push(`L ${inX} ${inY}`, `Q ${p.x} ${p.y} ${outX} ${outY}`);
  }
  const last = points[points.length - 1];
  if (points.length > 1) parts.push(`L ${last.x} ${last.y}`);
  return parts.join(' ');
}

/** The point at fraction t of the polyline's length (the dot rides it with a transform only). */
export function pointAt(points: Point[], t: number): Point {
  const legs = points.slice(1).map((point, index) => distance(points[index], point));
  let left = Math.max(0, Math.min(1, t)) * legs.reduce((sum, leg) => sum + leg, 0);
  for (let index = 0; index < legs.length; index++) {
    if (left <= legs[index] || index === legs.length - 1) {
      const f = legs[index] ? Math.min(1, left / legs[index]) : 1;
      const [a, b] = [points[index], points[index + 1]];
      return { x: a.x + (b.x - a.x) * f, y: a.y + (b.y - a.y) * f };
    }
    left -= legs[index];
  }
  return points[0] ?? { x: 0, y: 0 };
}

function build(points: Point[], boxes: Box[], ends: (string | undefined)[], radius: number, pad: number): Route {
  const clean = simplify(points);
  const length = clean.slice(1).reduce((sum, point, index) => sum + distance(clean[index], point), 0);
  return { points: clean, path: roundedPath(clean, radius), length, at: t => pointAt(clean, t),
    clear: crossings(clean, boxes, ends, pad).length === 0 };
}

/** One change to all its receivers: one shared exit, one lane, one drop per receiver. */
export function routeFan(from: RouteEnd, targets: RouteEnd[], boxes: Box[], options: RouteOptions = {}): Route[] {
  const clearance = options.clearance ?? ROUTE_CLEARANCE;
  const lane = laneY(boxes, options.margin ?? LANE_MARGIN, [from.point, ...targets.map(target => target.point)]);
  const middle = targets.length ? targets.reduce((sum, target) => sum + target.point.x, 0) / targets.length : from.point.x;
  const exit = exitPath(from, boxes, lane, Math.sign(middle - from.point.x) || 1, clearance);
  return targets.map(target => build([...exit, ...dropPath(target, boxes, lane, clearance)], boxes,
    [from.box?.id, target.box?.id], options.radius ?? CORNER_RADIUS, clearance / 2 - 0.5));
}

export function routeLine(from: RouteEnd, to: RouteEnd, boxes: Box[], options: RouteOptions = {}): Route {
  return routeFan(from, [to], boxes, options)[0];
}

/** The middle of a box's top edge: where a line ends on, or leaves, an IED component (ADR-0112). */
export const topOf = (box: Box): Point => ({ x: box.x + box.width / 2, y: box.y });

/** From the origin into the sender's IED component, then out of it to every receiver: one route per
 * receiver, sharing the first leg (ADR-0112: origin -> sender IED -> receiving IEDs). */
export function routeVia(from: RouteEnd, via: Box, targets: RouteEnd[], boxes: Box[], options: RouteOptions = {}): Route[] {
  const first = routeLine(from, { point: topOf(via), box: via }, boxes, options);
  const fan = routeFan({ point: topOf(via), box: via, up: true }, targets, boxes, options);
  const clearance = options.clearance ?? ROUTE_CLEARANCE;
  return fan.map((route, index) => build([...first.points, ...route.points], boxes,
    [from.box?.id, via.id, targets[index].box?.id], options.radius ?? CORNER_RADIUS, clearance / 2 - 0.5));
}
