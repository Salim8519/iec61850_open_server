import type { EquipmentNode } from '../api/types';
import { measurementBindings, relayBindings, stateBindings } from './stateBindings.ts';

/** Where the small IED tags of the SLD signal layers sit (spec signal-layers, LAYER-10).
 * Tags are a view only: they are placed from the drawing every time and never saved. */
export type Box = { id: string; x: number; y: number; width: number; height: number };
export type TagSpec = { name: string; iedId: string | null; outside: boolean; remote: boolean };
/** `corner`: a small mark inside the anchor's top-right corner; x is then the mark's RIGHT edge. */
export type PlacedTag = TagSpec & { x: number; y: number; anchor: string | null; corner: boolean };

export const TAG_HEIGHT = 26;
export const TAG_GAP = 6;
/** Corner marks (change 0105): size, inset from the box edge and gap between two marks. */
export const MARK_SIZE = 18;
export const MARK_INSET = 4;
const EDGE_WIDTH = 170;

/** Equipment node ids bound to each IED id, through state, command or relay bindings. */
export function equipmentByIed(nodes: EquipmentNode[]): Map<string, string[]> {
  const result = new Map<string, string[]>();
  for (const node of nodes) {
    const data = node.data;
    const ids = [...stateBindings(data.state_mapping), ...relayBindings(data), ...measurementBindings(data)]
      .map(item => item.ied_id);
    if (data.control_binding) ids.push(data.control_binding.ied_id);
    for (const id of new Set(ids)) result.set(id, [...(result.get(id) || []), node.id]);
  }
  return result;
}

function bounds(boxes: Box[]) {
  if (!boxes.length) return { minX: 0, minY: 0 };
  return { minX: Math.min(...boxes.map(box => box.x)), minY: Math.min(...boxes.map(box => box.y)) };
}

/** A corner mark inside the topmost bound equipment (change 0105: a tag beside it covered the
 * neighbouring bay); a second IED on the same equipment steps left. Unbound project IEDs keep a
 * full-name tag in a small column left of the drawing.
 * IEDs outside the project get no tag at all: the SLD stays clean (change 0097); they are
 * listed in the toolbar and their GOOSE flies in from beside the receiver (edgePoint). */
export function placeTags(boxes: Box[], bound: Map<string, string[]>, tags: TagSpec[]): PlacedTag[] {
  const byId = new Map(boxes.map(box => [box.id, box]));
  const { minX, minY } = bounds(boxes);
  const stacked = new Map<string, number>();
  let edge = 0;
  return tags.filter(tag => !tag.outside).sort((a, b) => a.name.localeCompare(b.name)).map(tag => {
    const anchors = (tag.iedId ? bound.get(tag.iedId) || [] : []).map(id => byId.get(id)).filter(Boolean) as Box[];
    if (anchors.length) {
      const anchor = [...anchors].sort((a, b) => a.y - b.y || a.x - b.x)[0];
      const index = stacked.get(anchor.id) || 0;
      stacked.set(anchor.id, index + 1);
      return { ...tag, anchor: anchor.id, corner: true, y: anchor.y + MARK_INSET,
        x: anchor.x + anchor.width - MARK_INSET - index * (MARK_SIZE + MARK_INSET) };
    }
    return { ...tag, anchor: null, corner: false, x: minX - EDGE_WIDTH, y: minY + edge++ * (TAG_HEIGHT + 14) };
  });
}

/** Where a GOOSE dot from an IED outside the project starts: a fixed screen distance left of and
 * above the receiver tag (`scale` is tagScale), so it starts in view at every zoom. The drawing's
 * left edge put it off screen on large drawings (change 0102). */
export const ENTRY_DISTANCE = 220;
export function edgePoint(to: { x: number; y: number }, scale = 1) {
  return { x: to.x - ENTRY_DISTANCE * scale, y: to.y - 40 * scale };
}

/** Tags grow back toward screen size when zoomed out, between 1× and 3× their drawing size. */
export function tagScale(zoom: number): number {
  return Math.min(3, Math.max(1, 0.9 / (zoom || 1)));
}

/** Corner marks grow less (at most 1.5×) so they never fill the equipment they sit in. */
export function markScale(zoom: number): number {
  return Math.min(1.5, tagScale(zoom));
}

/** Where links and dots meet a tag: the mark's centre, or a name tag's left part. */
export function tagCentre(tag: PlacedTag, scale: number, markScaleValue = scale) {
  return tag.corner
    ? { x: tag.x - (MARK_SIZE / 2) * markScaleValue, y: tag.y + (MARK_SIZE / 2) * markScaleValue }
    : { x: tag.x + 36 * scale, y: tag.y + (TAG_HEIGHT / 2) * scale };
}

/** A gentle curve from the publisher tag to the subscriber tag; returns the SVG path and the
 * point at fraction t, so a pulse dot can move along it with a transform only. */
export function linkCurve(from: { x: number; y: number }, to: { x: number; y: number }) {
  const bend = Math.min(80, Math.hypot(to.x - from.x, to.y - from.y) / 4);
  const control = { x: (from.x + to.x) / 2, y: Math.min(from.y, to.y) - bend };
  const at = (t: number) => ({
    x: (1 - t) ** 2 * from.x + 2 * (1 - t) * t * control.x + t ** 2 * to.x,
    y: (1 - t) ** 2 * from.y + 2 * (1 - t) * t * control.y + t ** 2 * to.y,
  });
  return { path: `M ${from.x} ${from.y} Q ${control.x} ${control.y} ${to.x} ${to.y}`, at };
}
