import type { ChangeGroup, ChangeLine } from './signalChanges.ts';
import type { Box, Components } from './signalLayout.ts';
import type { PulseKind } from './signalPulses.ts';
import { routeFan, routeVia, topOf, type Point, type Route, type RouteEnd } from './signalRoute.ts';

/** Where each change is drawn (ADR-0112, routing of ADR-0103): from the component that triggered
 * it, through the sender's IED component, to every receiving IED component. Without a trigger it
 * starts at the sender's component; a sender without a component is skipped (origin straight to
 * the receivers). A line never ends on equipment: only IED components are targets. A sender with no
 * place at all (outside the project, or no component and no trigger) flies one dot into each
 * receiving component. Receivers without a component are named in one chip. Pure. */
export const STAGGER_MS = 80;
export const MAX_STAGGER_MS = 400;
/** At most this many lines move at once; the rest are drawn at once without a travelling dot. */
export const MAX_ANIMATED = 16;

export type PlannedLine = { key: number; linkId: string; kind: PulseKind; start: number; subscriber: string; route: Route };
/** A receiver reached by a flying dot (`to`: the top of its IED component); `label` is the
 * receiver's short line ("← FEED1 · Open"), `title` the full words on hover. */
export type FlyLine = { key: number; linkId: string; kind: PulseKind; start: number; subscriber: string; to: Point;
  label: string; title: string };
/** `box`: the origin to outline; `sender`: the sender's component; `chip`: where the change's words sit
 * (null: nothing on the drawing to hold them); `missing`: receivers with no IED component. */
export type PlannedChange = { group: ChangeGroup; box: Box | null; sender: Box | null; chip: Point | null;
  lines: PlannedLine[]; fly: FlyLine[]; missing: string[]; missingAt: Point | null };

export const staggerOf = (index: number) => Math.min(index * STAGGER_MS, MAX_STAGGER_MS);

const centre = (box: Box): Point => ({ x: box.x + box.width / 2, y: box.y + box.height / 2 });
const lineOf = (line: ChangeLine) => ({ key: line.pulse.key, linkId: line.link.id, kind: line.pulse.kind,
  start: line.pulse.start, subscriber: line.link.subscriber });

export function planChanges(groups: ChangeGroup[], boxes: Box[], components: Components, scale = 1): PlannedChange[] {
  const byId = new Map(boxes.map(box => [box.id, box]));
  const options = { margin: 36 * scale, clearance: 8, radius: 8 * Math.min(scale, 1.5) };
  return groups.map(group => {
    const sender = components.byName.get(group.publisher) ?? null;
    const origin = group.origin ? byId.get(group.origin.nodeId) ?? null : null;
    // Sent from the sender's own component: the line simply starts there.
    const box = origin && sender && origin.id === sender.id ? null : origin;
    const reached = group.lines.filter(line => components.byName.has(line.link.subscriber));
    const ends: RouteEnd[] = reached.map(line => {
      const target = components.byName.get(line.link.subscriber)!;
      return { point: topOf(target), box: target };
    });
    const from: RouteEnd | null = box ? { point: centre(box), box, edge: true } : null;
    let routes: Route[] = [];
    if (ends.length && from && sender) routes = routeVia(from, sender, ends, boxes, options);
    else if (ends.length && from) routes = routeFan(from, ends, boxes, options);
    else if (ends.length && sender) routes = routeFan({ point: topOf(sender), box: sender, up: true }, ends, boxes, options);
    const anchor = box ?? sender;
    const missing = group.receivers.filter(name => !components.byName.has(name));
    const holder = sender ?? box;
    return {
      group, box, sender, missing,
      chip: anchor ? { x: anchor.x, y: anchor.y - 4 * scale } : null,
      lines: routes.map((route, index) => ({ ...lineOf(reached[index]), start: reached[index].pulse.start + staggerOf(index), route })),
      fly: anchor ? [] : reached.map(line => ({ ...lineOf(line), to: topOf(components.byName.get(line.link.subscriber)!),
        label: line.pulse.short, title: line.pulse.label })),
      missingAt: missing.length && holder ? { x: holder.x, y: holder.y + holder.height + 4 * scale } : null,
    };
  });
}

/** Keys of the lines that animate: the newest changes first, at most `cap` lines. */
export function animatedLines(plan: PlannedChange[], cap = MAX_ANIMATED): Set<number> {
  const keys = new Set<number>();
  for (const change of [...plan].reverse()) for (const line of change.lines) if (keys.size < cap) keys.add(line.key);
  return keys;
}
