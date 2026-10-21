// Filters of the SCADA point lists (change 0093, type chips ADR-0095): display only, on rows the page already has.
import type { GatewayPoint } from '../../../api/gateway';
import type { MasterPoint } from '../../../api/master';
import { KIND_LABELS, OWN_COPY } from '../../../copy/gateway.ts';
import { kindOfGatewayPoint, kindOfMasterGroup } from '../../../points/pointKind.ts';
import { textMatch, type FilterSpec, type FilterValues } from '../../../ui/filterState.ts';
import { kindGroup, kindMatches } from '../../../ui/kindFilter.ts';

export const GATEWAY_POINT_FILTER: FilterSpec = {
  search: 'Search name or address',
  selects: [{ key: 'ied', label: 'Only one IED', all: 'All IEDs' }],
  groups: [kindGroup()],
  toggles: [
    { key: 'bad', label: 'Bad quality', hint: 'Points SCADA receives as invalid, for example while the IEDs are not running.' },
    { key: 'unserved', label: 'Not served', hint: 'New rows that SCADA gets only after Save point list.' },
    { key: 'own', label: OWN_COPY.filter, hint: OWN_COPY.filterHint }],
};

export function gatewayPointMatches(row: GatewayPoint, values: FilterValues): boolean {
  const kind = kindOfGatewayPoint(row.kind);
  if (!kindMatches(values.kind, kind)) return false;
  if (values.ied && (row.ied_name ?? row.ied) !== values.ied) return false;
  if (values.bad && row.online !== false) return false;
  if (values.unserved && row.id && !row.draft) return false;
  if (values.own && row.source !== 'manual') return false;
  return textMatch(values.text, row.ref, row.ied_name ?? row.ied, row.address, KIND_LABELS[row.kind], kind, row.description);
}

export const MASTER_POINT_FILTER: FilterSpec = {
  search: 'Search name, address or value',
  selects: [{ key: 'group', label: 'Only one point group', all: 'All point groups' }],
  groups: [kindGroup()],
  toggles: [
    { key: 'bad', label: 'Bad quality', hint: 'Points whose quality is not good, or old values after the link was lost.' },
    { key: 'commandable', label: 'Has a command', hint: 'Only points the device accepts a command for (they get a Command… button).' },
    { key: 'watched', label: 'Watched only', hint: 'Hide the points you unticked.' }],
};

export function masterPointMatches(row: MasterPoint, values: FilterValues): boolean {
  const kind = kindOfMasterGroup(row.group);
  if (!kindMatches(values.kind, kind)) return false;
  if (values.group && row.group_name !== values.group) return false;
  if (values.bad && row.good) return false;
  if (values.commandable && !row.command) return false;
  if (values.watched && !row.ticked) return false;
  return textMatch(values.text, row.label, row.address, row.group_name, row.type, kind, row.value_text);
}

/** The device points served upstream (Converter): same type chips and search, read only. */
export const SERVED_POINT_FILTER: FilterSpec = { search: 'Search name or address', groups: [kindGroup()] };

export function servedPointMatches(row: GatewayPoint, values: FilterValues): boolean {
  const kind = kindOfGatewayPoint(row.kind);
  return kindMatches(values.kind, kind) && textMatch(values.text, row.description, row.address, KIND_LABELS[row.kind], kind);
}
