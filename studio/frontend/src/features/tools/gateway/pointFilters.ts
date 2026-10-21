// Filters of the SCADA point lists (change 0093): display only, on rows the page already has.
import type { GatewayPoint, PointKind } from '../../../api/gateway';
import type { MasterPoint } from '../../../api/master';
import { KIND_LABELS, OWN_COPY } from '../../../copy/gateway.ts';
import { textMatch, type FilterGroup, type FilterSpec, type FilterValues } from '../../../ui/filterState.ts';

const TYPE_GROUP: FilterGroup = { key: 'kind', label: 'Point type', initial: 'all', options: [
  { value: 'all', label: 'All' }, { value: 'status', label: 'Status' },
  { value: 'measured', label: 'Measured' }, { value: 'commands', label: 'Commands' }] };

export const GATEWAY_POINT_FILTER: FilterSpec = {
  search: 'Search name or address',
  selects: [{ key: 'ied', label: 'Only one IED', all: 'All IEDs' }],
  groups: [TYPE_GROUP],
  toggles: [
    { key: 'bad', label: 'Bad quality', hint: 'Points SCADA receives as invalid, for example while the IEDs are not running.' },
    { key: 'unserved', label: 'Not served', hint: 'New rows that SCADA gets only after Save point list.' },
    { key: 'own', label: OWN_COPY.filter, hint: OWN_COPY.filterHint }],
};

export const gatewayKind = (kind: PointKind) => kind.startsWith('command') ? 'commands'
  : kind === 'single' || kind === 'double' ? 'status' : 'measured';

export function gatewayPointMatches(row: GatewayPoint, values: FilterValues): boolean {
  if (values.kind !== 'all' && gatewayKind(row.kind) !== values.kind) return false;
  if (values.ied && (row.ied_name ?? row.ied) !== values.ied) return false;
  if (values.bad && row.online !== false) return false;
  if (values.unserved && row.id && !row.draft) return false;
  if (values.own && row.source !== 'manual') return false;
  return textMatch(values.text, row.ref, row.ied_name ?? row.ied, row.address, KIND_LABELS[row.kind], row.description);
}

const MASTER_STATUS = new Set(['single', 'double', 'step', 'bitstring', 'binary', 'double_binary', 'coil',
  'discrete_input', 'binary_output']);

export const MASTER_POINT_FILTER: FilterSpec = {
  search: 'Search name, address or value',
  selects: [{ key: 'group', label: 'Only one point group', all: 'All point groups' }],
  groups: [TYPE_GROUP],
  toggles: [
    { key: 'bad', label: 'Bad quality', hint: 'Points whose quality is not good, or old values after the link was lost.' },
    { key: 'watched', label: 'Watched only', hint: 'Hide the points you unticked.' }],
};

/** Status or measured by point group; "Commands" are the points that accept a command (they may be both). */
export function masterKindMatches(row: MasterPoint, kind: string | boolean): boolean {
  if (kind === 'commands') return Boolean(row.command);
  if (kind === 'status') return MASTER_STATUS.has(row.group);
  return kind !== 'measured' || !MASTER_STATUS.has(row.group);
}

export function masterPointMatches(row: MasterPoint, values: FilterValues): boolean {
  if (!masterKindMatches(row, values.kind)) return false;
  if (values.group && row.group_name !== values.group) return false;
  if (values.bad && row.good) return false;
  if (values.watched && !row.ticked) return false;
  return textMatch(values.text, row.label, row.address, row.group_name, row.type, row.value_text);
}
