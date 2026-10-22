// Filters of the Devices area (change 0093). The IED list and Compare filter rows the page already
// has; the Data points tab sends its choices to the existing read-only signals endpoint.
import type { Ied } from '../../api/types';
import { KIND_COPY } from '../../copy/pointKinds.ts';
import { KINDS, kindOf61850, type Kind } from '../../points/pointKind.ts';
import { textMatch, type FilterSpec, type FilterToggle, type FilterValues } from '../../ui/filterState.ts';
import { ALL_KINDS, kindGroup, kindMatches } from '../../ui/kindFilter.ts';

export const DEVICE_FILTER: FilterSpec = {
  search: 'Search IEDs',
  groups: [{ key: 'role', label: 'What ScadaBench does with it', initial: 'all', options: [
    { value: 'all', label: 'All' }, { value: 'simulated', label: 'Simulate' }, { value: 'real', label: 'Monitor real' }, { value: 'off', label: 'Skip' }] }],
  toggles: [{ key: 'problems', label: 'Has problems', hint: 'IEDs stopped by an error, with file check notes or GOOSE notes.' }],
};

/** File check notes and GOOSE notes, as counted in the Overview. */
export const noteCount = (ied: Ied) => ied.warnings.length + (ied.goose?.diagnostics?.length ?? 0);
export const hasProblems = (ied: Ied) => ied.state === 'failed' || ied.validation?.status === 'invalid' || noteCount(ied) > 0;

export function deviceMatches(ied: Ied, values: FilterValues): boolean {
  if (values.role !== 'all' && ied.role !== values.role) return false;
  if (values.problems && !hasProblems(ied)) return false;
  return textMatch(values.text, ied.name, ied.scl_address?.ip, ied.live?.ip);
}

/** How many IEDs each role chip would show (the chip says "Monitor real 2"). */
export function roleCounts(ieds: Ied[]): Record<string, number> {
  const counts: Record<string, number> = { all: ieds.length, real: 0, simulated: 0, off: 0 };
  for (const ied of ieds) counts[ied.role] = (counts[ied.role] ?? 0) + 1;
  return counts;
}

/** Filter value key of the universal point-type chips; the engine gets it as `point_kind` (ADR-0095). */
export const POINT_KIND = 'point_kind';
/** "Hide quality / time": leaves out the q and t helper attributes; the engine gets `helpers=hide`. */
export const HIDE_HELPERS: FilterToggle = { key: 'helpers', label: KIND_COPY.hideHelpers, hint: KIND_COPY.hideHelpersHint };

export const SIGNAL_FILTER: FilterSpec = {
  search: 'Search name or path',
  selects: [{ key: 'ld', label: 'Only one logical device', all: 'All logical devices' },
    { key: 'report', label: 'Only points of one report', all: 'Any report' }],
  // One point-type list replaces the old "Kind of data" FC chips (ADR-0095), so the filters take one
  // row; the FC stays visible on every row.
  groups: [kindGroup(POINT_KIND, true)],
  toggles: [{ key: 'bad', label: 'Bad quality', hint: 'Points whose quality is not good right now.' },
    { key: 'dataset', label: 'In a dataset', hint: 'Points that a report or GOOSE dataset sends.' }, HIDE_HELPERS],
};

/** The query of the existing signals endpoint for these choices (all of it is read-side filtering). */
export function signalQuery(values: FilterValues, offset: number): string {
  const query = new URLSearchParams({ q: String(values.text ?? '').trim(), offset: String(offset) });
  query.set('kind_counts', '1'); // Go only: counts per type so empty type chips hide (ADR-0095)
  const pointKind = String(values[POINT_KIND] ?? ALL_KINDS);
  if ((KINDS as readonly string[]).includes(pointKind)) query.set('point_kind', pointKind);
  if (values.helpers) query.set('helpers', 'hide');
  if (values.ld) query.set('ld', String(values.ld));
  if (values.report) query.set('report', String(values.report));
  if (values.bad) query.set('bad_quality', 'true');
  if (values.dataset) query.set('in_dataset', 'true');
  return query.toString();
}

export const LIVE_FILTER: FilterSpec = { search: 'Search references', groups: SIGNAL_FILTER.groups, toggles: [HIDE_HELPERS] };

export const COMPARE_FILTER: FilterSpec = {
  search: 'Search references',
  groups: [{ key: 'show', label: 'Show', initial: 'problems', options: [
    { value: 'problems', label: 'Differences only' }, { value: 'missing_on_device', label: 'Missing in device' },
    { value: 'extra_on_device', label: 'Extra in device' }, { value: 'type_mismatch', label: 'Type mismatch' },
    { value: 'all', label: 'Show all' }] }, kindGroup(POINT_KIND)],
};

type CompareLike = { ref: string; fc: string; category: string; scd_type?: string | null; device_type?: string | null };

/** Kind of a compared point: the device's type when it has one (that is what clients read), else the SCD's. */
export function compareKind(row: CompareLike, classOf?: (ref: string) => string | undefined): Kind {
  return kindOf61850(row.fc, row.device_type ?? row.scd_type ?? '', classOf?.(row.ref));
}

/** Result, search and (when the row's kind is given) the point-type chip; counts use it without the kind. */
export function compareMatches(row: CompareLike, values: FilterValues, kind?: Kind): boolean {
  const show = values.show;
  if (show === 'problems' ? row.category === 'match' : show !== 'all' && row.category !== show) return false;
  if (kind && !kindMatches(values[POINT_KIND], kind)) return false;
  return textMatch(values.text, row.ref, row.fc);
}
