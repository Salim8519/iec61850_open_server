import type { PanelPoint, PointType } from '../api/types.ts';
import { pointRef } from './pointPanel.ts';

/** The type chips of the popup: everything, or one type. */
export type PointFilter = 'all' | PointType;
export const FILTER_CHIPS: { value: PointFilter; label: string }[] = [
  { value: 'all', label: 'All' }, { value: 'status', label: 'Status' }, { value: 'double', label: 'Double' },
  { value: 'command', label: 'Commands' }, { value: 'value', label: 'Values' },
];

/** How many rows each chip would show (the search is not counted, so the chips stay put while typing). */
export function chipCounts(points: readonly PanelPoint[]): Record<PointFilter, number> {
  const counts: Record<PointFilter, number> = { all: points.length, status: 0, double: 0, command: 0, value: 0 };
  for (const point of points) counts[point.type] += 1;
  return counts;
}

/** Rows of the chosen type whose name, group or reference contains every word typed. */
export function filterPoints(points: readonly PanelPoint[], filter: PointFilter, query: string): PanelPoint[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  return points.filter(point => {
    if (filter !== 'all' && point.type !== filter) return false;
    if (!words.length) return true;
    const text = `${point.label}\n${point.group}\n${pointRef(point)}`.toLowerCase();
    return words.every(word => text.includes(word));
  });
}

export type PointGroup = { group: string; points: PanelPoint[] };

/** Rows under their group in the order the groups first appear; rows without a group form one group. */
export function groupPoints(points: readonly PanelPoint[]): PointGroup[] {
  const groups = new Map<string, PanelPoint[]>();
  for (const point of points) {
    const found = groups.get(point.group);
    if (found) found.push(point); else groups.set(point.group, [point]);
  }
  return [...groups].map(([group, list]) => ({ group, points: list }));
}

/** One flat list for the virtual scroller: a header before the rows of each group. */
export type PanelListItem = { key: string; kind: 'group'; group: string; count: number }
  | { key: string; kind: 'point'; point: PanelPoint };

export function listItems(points: readonly PanelPoint[]): PanelListItem[] {
  const items: PanelListItem[] = [];
  for (const { group, points: list } of groupPoints(points)) {
    items.push({ key: `g:${group}`, kind: 'group', group, count: list.length });
    for (const point of list) items.push({ key: point.id, kind: 'point', point });
  }
  return items;
}
