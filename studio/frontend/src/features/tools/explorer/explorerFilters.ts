// Filters of IED Explorer (change 0093): the model tree by logical-node class and the watch list.
// Display only; Explorer still loads one level at a time and never writes.
import type { Attribute, WatchItem } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools.ts';
import { KIND_COPY } from '../../../copy/pointKinds.ts';
import type { Kind } from '../../../points/pointKind.ts';
import { textMatch, type FilterSpec, type FilterValues } from '../../../ui/filterState.ts';
import { kindGroup, kindMatches } from '../../../ui/kindFilter.ts';
import { watchKind } from './attributeKind.ts';

export const TREE_FILTER: FilterSpec = {
  search: 'Search logical nodes',
  groups: [{ key: 'kind', label: 'Kind of logical node', initial: 'all', options: [
    { value: 'all', label: 'All' }, { value: 'controls', label: 'Controls' },
    { value: 'measured', label: 'Measured' }, { value: 'settings', label: 'Settings' }] }],
};

/** IEC 61850-7-4 logical-node groups: C/X/Y switch and control, M/T measure, P/R protection settings. */
export function nodeKind(name: string): 'controls' | 'measured' | 'settings' | 'other' {
  const lnClass = name.replace(/\d+$/, '').slice(-4);  // prefix + class + instance, e.g. Q0CSWI1 -> CSWI
  const letter = /^[A-Z]{4}$/.test(lnClass) ? lnClass[0] : '';
  if (letter && 'CXY'.includes(letter)) return 'controls';
  if (letter && 'MT'.includes(letter)) return 'measured';
  if (letter && 'PR'.includes(letter)) return 'settings';
  return 'other';
}

export function nodeMatches(name: string, values: FilterValues): boolean {
  if (values.kind !== 'all' && nodeKind(name) !== values.kind) return false;
  return textMatch(values.text, name);
}

export const WATCH_FILTER: FilterSpec = {
  search: 'Search watched values',
  groups: [kindGroup()],
  toggles: [
    { key: 'recent', label: 'Changed in last minute', hint: 'Values that changed while this page was open, in the last 60 s.' },
    { key: 'bad', label: 'Bad quality', hint: 'Values whose quality is not good, or that could not be read.' }],
};

export const watchKey = (item: Pick<WatchItem, 'ref' | 'fc'>) => `${item.ref}|${item.fc}`;

/** When each watched value last changed text, as seen by this page; the first reading is not a change. */
export function trackChanges(previous: Map<string, { text: string; at: number }>, items: WatchItem[], now: number) {
  const next = new Map<string, { text: string; at: number }>();
  for (const item of items) {
    const key = watchKey(item);
    const before = previous.get(key);
    next.set(key, !before ? { text: item.text, at: 0 } : before.text === item.text ? before : { text: item.text, at: now });
  }
  return next;
}

export function watchMatches(item: WatchItem, values: FilterValues, changedAt: number, now: number): boolean {
  if (values.recent && !(changedAt && now - changedAt <= 60_000)) return false;
  if (values.bad && item.status === 'ok') return false;
  if (!kindMatches(values.kind, watchKind(item))) return false;
  return textMatch(values.text, item.ref, item.fc, item.text);
}

/** One logical node's attributes (ADR-0095): search, point type, and a switch that hides the quality and
 *  time helpers so only the points are left. */
export const NODE_FILTER: FilterSpec = {
  search: EXPLORER_COPY.searchAttributes,
  groups: [kindGroup()],
  toggles: [{ key: 'helpers', label: KIND_COPY.hideHelpers, hint: KIND_COPY.hideHelpersHint }],
};

export function attributeShown(object: string, attribute: Attribute, kind: Kind, values: FilterValues): boolean {
  if (values.helpers && kind === 'AT') return false;
  if (!kindMatches(values.kind, kind)) return false;
  return textMatch(values.text, object, attribute.path, attribute.fc, attribute.type);
}
