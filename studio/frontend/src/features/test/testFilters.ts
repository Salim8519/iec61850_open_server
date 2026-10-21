// Filters of the Test points list (ADR-0095): search, point type and, when the points come from
// several IEDs, one IED. Display only: the list is the project's own saved test points.
import type { Ied, Signal } from '../../api/types';
import { kindOfSignal, type ClassOf } from '../../components/signalKind.ts';
import type { Kind } from '../../points/pointKind.ts';
import { textMatch, type Choice, type FilterSpec, type FilterValues } from '../../ui/filterState.ts';
import { kindGroup, kindMatches } from '../../ui/kindFilter.ts';

export const TEST_FILTER: FilterSpec = {
  search: 'Search test points',
  groups: [kindGroup('point_kind')],
  selects: [{ key: 'ied', label: 'Only one IED', all: 'All IEDs' }],
};

type Point = { label: string; ied_id: string; ref: string; signal: Signal };

/** Kind of a test point: from the signal's protocol type, the data class (route) or the IED's control list. */
export function testPointKind(point: Point, ied: Pick<Ied, 'controls'> | undefined, classOf?: ClassOf): Kind {
  return kindOfSignal(point.signal, classOf, ied?.controls);
}

export function testPointMatches(point: Point, kind: Kind, iedName: string | undefined, values: FilterValues): boolean {
  if (values.ied && values.ied !== point.ied_id) return false;
  if (!kindMatches(values.point_kind, kind)) return false;
  return textMatch(values.text, point.label, point.ref, iedName);
}

/** The IED dropdown: only IEDs that have test points; it hides itself with fewer than two. */
export function testIedChoices(points: Pick<Point, 'ied_id'>[], ieds: Pick<Ied, 'id' | 'name'>[]): Choice[] {
  const used = new Set(points.map(point => point.ied_id));
  return ieds.filter(ied => used.has(ied.id)).map(ied => ({ value: ied.id, label: ied.name }));
}

/** Swap two saved test points (the moved one and its visible neighbour); ids unknown leave the order alone. */
export function swapPoints<T extends { id: string }>(entries: readonly T[], idA: string, idB: string): T[] {
  const next = [...entries];
  const a = next.findIndex(entry => entry.id === idA);
  const b = next.findIndex(entry => entry.id === idB);
  if (a < 0 || b < 0) return next;
  [next[a], next[b]] = [next[b], next[a]];
  return next;
}
