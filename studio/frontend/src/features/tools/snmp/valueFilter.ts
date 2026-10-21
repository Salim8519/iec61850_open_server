// Filters of the two SNMP value lists (ADR-0095): search, point-type chips and one toggle each.
// Display only: the lists still receive every value; these functions only choose which rows show.
import type { ValueRow } from '../../../api/snmp';
import { POINTS_COPY } from '../../../copy/snmp.ts';
import { READ_COPY } from '../../../copy/snmpRead.ts';
import { countKinds, kindOfSnmp, type Kind } from '../../../points/pointKind.ts';
import type { FilterSpec, FilterValues } from '../../../ui/filterState.ts';
import { kindGroup, kindMatches } from '../../../ui/kindFilter.ts';
import { rowMatches } from './rowMatch.ts';

/** Simulate devices › Values: search, point type, and only the values a manager may change. */
export const SIM_FILTER: FilterSpec = { search: POINTS_COPY.search, groups: [kindGroup()],
  toggles: [{ key: 'writable', label: POINTS_COPY.writable }] };

/** Read devices › Values: search, point type, important values and watched values. */
export const READ_FILTER: FilterSpec = { search: READ_COPY.search, groups: [kindGroup()], toggles: [
  { key: 'key', label: READ_COPY.keyOnly, hint: READ_COPY.keyHint }, { key: 'polled', label: READ_COPY.watchedOnly }] };

/** The kind of an SNMP value comes from its SMI type; a row without a type is Unknown, never guessed. */
export const kindOfValue = (row: Pick<ValueRow, 'type'>): Kind => kindOfSnmp(row.type ?? '');

const base = (row: ValueRow, values: FilterValues) =>
  rowMatches(row, String(values.text ?? '')) && kindMatches(values.kind, kindOfValue(row));

export const simShown = (row: ValueRow, values: FilterValues) => base(row, values) && (!values.writable || row.writable);

export const readShown = (row: ValueRow, values: FilterValues) =>
  base(row, values) && (!values.key || Boolean(row.key)) && (!values.polled || Boolean(row.polled));

/** How many values of each kind a list has: the numbers on the type chips. */
export const kindCounts = (rows: readonly ValueRow[]) => ({ kind: countKinds(rows, kindOfValue) });
