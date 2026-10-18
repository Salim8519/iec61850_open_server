// Filters of the Devices area (change 0093). The IED list and Compare filter rows the page already
// has; the Data points tab sends its choices to the existing read-only signals endpoint.
import type { Ied } from '../../api/types';
import { textMatch, type FilterSpec, type FilterValues } from '../../ui/filterState.ts';

export const DEVICE_FILTER: FilterSpec = {
  search: 'Search IEDs',
  groups: [{ key: 'role', label: 'What ScadaBench does with it', initial: 'all', options: [
    { value: 'all', label: 'All' }, { value: 'real', label: 'Real' }, { value: 'simulated', label: 'Sim' }, { value: 'off', label: 'Off' }] }],
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

/** How many IEDs each role chip would show (the chip says "Real 2"). */
export function roleCounts(ieds: Ied[]): Record<string, number> {
  const counts: Record<string, number> = { all: ieds.length, real: 0, simulated: 0, off: 0 };
  for (const ied of ieds) counts[ied.role] = (counts[ied.role] ?? 0) + 1;
  return counts;
}

const KINDS: Record<string, string> = { status: 'ST', measured: 'MX', controls: 'CO', settings: 'SP', config: 'CF' };

export const SIGNAL_FILTER: FilterSpec = {
  search: 'Search name, logical node or path',
  selects: [{ key: 'ld', label: 'Only one logical device', all: 'All logical devices' },
    { key: 'report', label: 'Only points of one report', all: 'Any report' }],
  groups: [{ key: 'kind', label: 'Kind of data', initial: 'all', options: [
    { value: 'all', label: 'All' }, { value: 'status', label: 'Status' }, { value: 'measured', label: 'Measured' },
    { value: 'controls', label: 'Controls' }, { value: 'settings', label: 'Settings' }, { value: 'config', label: 'Config' }] }],
  toggles: [{ key: 'bad', label: 'Bad quality', hint: 'Points whose quality is not good right now.' },
    { key: 'dataset', label: 'In a dataset', hint: 'Points that a report or GOOSE dataset sends.' }],
};

/** The query of the existing signals endpoint for these choices (all of it is read-side filtering). */
export function signalQuery(values: FilterValues, offset: number): string {
  const query = new URLSearchParams({ q: String(values.text ?? '').trim(), offset: String(offset) });
  const fc = KINDS[String(values.kind)];
  if (fc) query.set('fc', fc);
  if (values.ld) query.set('ld', String(values.ld));
  if (values.report) query.set('report', String(values.report));
  if (values.bad) query.set('bad_quality', 'true');
  if (values.dataset) query.set('in_dataset', 'true');
  return query.toString();
}

export const LIVE_FILTER: FilterSpec = { search: 'Search references', groups: SIGNAL_FILTER.groups };

export const COMPARE_FILTER: FilterSpec = {
  search: 'Search references',
  groups: [{ key: 'show', label: 'Show', initial: 'problems', options: [
    { value: 'problems', label: 'Differences only' }, { value: 'missing_on_device', label: 'Missing in device' },
    { value: 'extra_on_device', label: 'Extra in device' }, { value: 'type_mismatch', label: 'Type mismatch' },
    { value: 'all', label: 'Show all' }] }],
};

export function compareMatches(row: { ref: string; fc: string; category: string }, values: FilterValues): boolean {
  const show = values.show;
  if (show === 'problems' ? row.category === 'match' : show !== 'all' && row.category !== show) return false;
  return textMatch(values.text, row.ref, row.fc);
}
