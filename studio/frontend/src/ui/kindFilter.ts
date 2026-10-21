// The type chips of every list (ADR-0095): "All types" plus one chip per point kind. Kinds with no
// rows hide themselves in the FilterBar, so each list shows only the kinds it has.
import { KIND_COPY, KIND_TEXT } from '../copy/pointKinds.ts';
import { KINDS, type Kind } from '../points/pointKind.ts';
import type { FilterGroup } from './filterState.ts';

export const ALL_KINDS = 'all';

/** The chip group for point kinds; `key` is the filter value key the list stores. By default it is one
 *  "All types ▾" dropdown with counts (13 chips would push the rows down); `select = false` shows chips. */
export function kindGroup(key = 'kind', select = true): FilterGroup {
  return { key, label: KIND_COPY.chips, initial: ALL_KINDS, kinds: true, select,
    options: [{ value: ALL_KINDS, label: KIND_COPY.all }, ...KINDS.map(kind => ({ value: kind, label: KIND_TEXT[kind].name }))] };
}

/** True when a row of this kind passes the chosen chip. */
export const kindMatches = (chosen: unknown, kind: Kind) => !chosen || chosen === ALL_KINDS || chosen === kind;
