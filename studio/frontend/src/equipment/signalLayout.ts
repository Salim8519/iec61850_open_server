import type { EquipmentNode } from '../api/types';
import { relayBindings, stateBindings } from './stateBindings.ts';

/** Where the small IED tags of the SLD signal layers sit (spec signal-layers, LAYER-10).
 * Tags are a view only: they are placed from the drawing every time and never saved. */
export type Box = { id: string; x: number; y: number; width: number; height: number };
export type TagSpec = { name: string; iedId: string | null; outside: boolean; remote: boolean };
export type PlacedTag = TagSpec & { x: number; y: number; anchor: string | null };

export const TAG_HEIGHT = 26;
export const TAG_GAP = 6;
const EDGE_WIDTH = 170;

/** Equipment node ids bound to each IED id, through state, command or relay bindings. */
export function equipmentByIed(nodes: EquipmentNode[]): Map<string, string[]> {
  const result = new Map<string, string[]>();
  for (const node of nodes) {
    const data = node.data;
    const ids = [...stateBindings(data.state_mapping), ...relayBindings(data)].map(item => item.ied_id);
    if (data.control_binding) ids.push(data.control_binding.ied_id);
    for (const id of new Set(ids)) result.set(id, [...(result.get(id) || []), node.id]);
  }
  return result;
}

function bounds(boxes: Box[]) {
  if (!boxes.length) return { minX: 0, minY: 0 };
  return { minX: Math.min(...boxes.map(box => box.x)), minY: Math.min(...boxes.map(box => box.y)) };
}

/** Beside the topmost bound equipment; everything else in a column at the left edge of the
 * drawing: IEDs outside the project (remote publishers) first, then unbound project IEDs. */
export function placeTags(boxes: Box[], bound: Map<string, string[]>, tags: TagSpec[]): PlacedTag[] {
  const byId = new Map(boxes.map(box => [box.id, box]));
  const { minX, minY } = bounds(boxes);
  const stacked = new Map<string, number>();
  let edge = 0;
  const order = (tag: TagSpec) => (tag.outside ? 0 : 1);
  return [...tags].sort((a, b) => order(a) - order(b) || a.name.localeCompare(b.name)).map(tag => {
    const anchors = (tag.iedId ? bound.get(tag.iedId) || [] : []).map(id => byId.get(id)).filter(Boolean) as Box[];
    if (!tag.outside && anchors.length) {
      const anchor = [...anchors].sort((a, b) => a.y - b.y || a.x - b.x)[0];
      const index = stacked.get(anchor.id) || 0;
      stacked.set(anchor.id, index + 1);
      return { ...tag, anchor: anchor.id, x: anchor.x + anchor.width + 10, y: anchor.y + index * (TAG_HEIGHT + TAG_GAP) };
    }
    return { ...tag, anchor: null, x: minX - EDGE_WIDTH, y: minY + edge++ * (TAG_HEIGHT + 14) };
  });
}

/** Tags grow back toward screen size when zoomed out, between 1× and 3× their drawing size. */
export function tagScale(zoom: number): number {
  return Math.min(3, Math.max(1, 0.9 / (zoom || 1)));
}

/** Short tag names ("C09" for "C09_BCU") unless two IEDs would then look the same. */
export function shortNames(names: string[]): Record<string, string> {
  const short = (name: string) => {
    const head = name.split('_')[0];
    if (head !== name && head.length >= 2) return head;
    return name.length <= 12 ? name : `${name.slice(0, 11)}…`;
  };
  const counts = new Map<string, number>();
  for (const name of new Set(names)) counts.set(short(name), (counts.get(short(name)) || 0) + 1);
  return Object.fromEntries([...new Set(names)].map(name => [name, counts.get(short(name))! > 1 ? name : short(name)]));
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
