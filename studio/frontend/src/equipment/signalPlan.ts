import type { ChangeGroup } from './signalChanges.ts';
import type { Box, PlacedTag } from './signalLayout.ts';
import type { PulseKind } from './signalPulses.ts';
import { routeFan, type Point, type Route, type RouteEnd } from './signalRoute.ts';

/** Where each change is drawn (ADR-0103): from its symbol (or its IED's mark when no symbol holds
 * the signal) to each receiving IED's mark, with a small start stagger. Pure; the overlay renders it. */
export const STAGGER_MS = 80;
export const MAX_STAGGER_MS = 400;
/** At most this many lines move at once; the rest are drawn at once without a travelling dot. */
export const MAX_ANIMATED = 16;

export type PlannedLine = { key: number; linkId: string; kind: PulseKind; start: number; subscriber: string; route: Route };
/** `box`: the origin symbol to outline (null when the change starts at an IED mark). */
export type PlannedChange = { group: ChangeGroup; box: Box | null; chip: Point; lines: PlannedLine[] };

export const staggerOf = (index: number) => Math.min(index * STAGGER_MS, MAX_STAGGER_MS);

function tagEnd(name: string, tags: Map<string, PlacedTag>, centre: Map<string, Point>, boxes: Map<string, Box>): RouteEnd | null {
  const tag = tags.get(name);
  const point = centre.get(name);
  if (!tag || !point) return null;
  return { point, box: tag.anchor ? boxes.get(tag.anchor) ?? null : null };
}

/** Changes whose publisher has no place on the drawing are left to the outside-sender dots. */
export function planChanges(groups: ChangeGroup[], boxes: Box[], placed: PlacedTag[], centre: Map<string, Point>,
  scale = 1): PlannedChange[] {
  const byId = new Map(boxes.map(box => [box.id, box]));
  const tags = new Map(placed.map(tag => [tag.name, tag]));
  const result: PlannedChange[] = [];
  for (const group of groups) {
    const symbol = group.origin ? byId.get(group.origin.nodeId) ?? null : null;
    const from: RouteEnd | null = symbol
      ? { point: { x: symbol.x + symbol.width / 2, y: symbol.y + symbol.height / 2 }, box: symbol, edge: true }
      : tagEnd(group.publisher, tags, centre, byId);
    if (!from) continue;
    const targets = group.lines.map(line => ({ line, end: tagEnd(line.link.subscriber, tags, centre, byId) }))
      .filter((item): item is { line: typeof item.line; end: RouteEnd } => !!item.end);
    if (!targets.length) continue;
    const routes = routeFan(from, targets.map(item => item.end), boxes,
      { margin: 36 * scale, clearance: 8, radius: 8 * Math.min(scale, 1.5) });
    result.push({ group, box: symbol, chip: symbol ? { x: symbol.x, y: symbol.y - 4 * scale } : from.point,
      lines: targets.map((item, index) => ({ key: item.line.pulse.key, linkId: item.line.link.id, kind: item.line.pulse.kind,
        start: item.line.pulse.start + staggerOf(index), subscriber: item.line.link.subscriber, route: routes[index] })) });
  }
  return result;
}

/** Keys of the lines that animate: the newest changes first, at most `cap` lines. */
export function animatedLines(plan: PlannedChange[], cap = MAX_ANIMATED): Set<number> {
  const keys = new Set<number>();
  for (const change of [...plan].reverse()) for (const line of change.lines) if (keys.size < cap) keys.add(line.key);
  return keys;
}
