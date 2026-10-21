// Filters of the Traffic and Commands tabs (change 0093). They only hide lines on this page;
// the backend keeps every line and Export/Clear behave as before.
import type { CommandEntry, TrafficEntry } from '../../../api/gateway';
import type { MasterCommandEntry } from '../../../api/master';
import { kindOfMasterCommand, type Kind } from '../../../points/pointKind.ts';
import { textMatch, type FilterSpec, type FilterValues } from '../../../ui/filterState.ts';
import { kindGroup, kindMatches } from '../../../ui/kindFilter.ts';

export const TRAFFIC_FILTER: FilterSpec = {
  search: 'Search messages',
  groups: [{ key: 'dir', label: 'Direction', initial: 'both', options: [
    { value: 'both', label: 'Both' }, { value: 'tx', label: 'Sent' }, { value: 'rx', label: 'Received' }] }],
  toggles: [
    { key: 'errors', label: 'Errors only', hint: 'Refusals, failures, unknown addresses and lost links.' },
    { key: 'quiet', label: 'Hide keep-alives', hint: 'Hide test frames (TESTFR) and link-status checks that only keep the link open.' }],
};

const ERROR = /refus|error|fail|unknown|invalid|negative|lost|time ?out|timed out|reject|not configured|cannot|closed/i;
const KEEP_ALIVE = /testfr|test frame|keep.?alive|link status|s-format|supervisory/i;

export const isError = (text: string) => ERROR.test(text);
export const isKeepAlive = (text: string) => KEEP_ALIVE.test(text);

export function trafficMatches(line: TrafficEntry, values: FilterValues): boolean {
  if (values.dir !== 'both' && line.dir !== values.dir) return false;
  if (values.errors && !isError(line.text)) return false;
  if (values.quiet && isKeepAlive(line.text)) return false;
  return textMatch(values.text, line.peer, line.text);
}

export const COMMAND_FILTER: FilterSpec = {
  search: 'Search commands',
  groups: [
    kindGroup(),
    { key: 'result', label: 'Result', initial: 'all', options: [
      { value: 'all', label: 'All' }, { value: 'done', label: 'Done' }, { value: 'refused', label: 'Refused' }] },
    { key: 'from', label: 'Sent by', initial: 'all', options: [
      { value: 'all', label: 'Anyone' }, { value: 'scada', label: 'From SCADA' }, { value: 'me', label: 'From me' }] }],
};

/** What the IED did with a gateway command: done, refused (or failed), or still waiting. */
export function gatewayResult(entry: CommandEntry): 'done' | 'refused' | 'waiting' {
  const text = entry.applied;
  if (!text) return 'waiting';
  return text.startsWith('operated') || text.includes('is now') ? 'done' : 'refused';
}

/** `kind` is the type of the operated point (see commandKinds.ts); "unknown" when the point is gone. */
export function gatewayCommandMatches(entry: CommandEntry, values: FilterValues, kind: Kind = 'UN'): boolean {
  if (!kindMatches(values.kind, kind)) return false;
  if (values.result !== 'all' && gatewayResult(entry) !== values.result) return false;
  const mine = entry.protocol === 'manual';
  if ((values.from === 'me' && !mine) || (values.from === 'scada' && mine)) return false;
  return textMatch(values.text, entry.label, entry.peer, entry.applied, entry.observed);
}

/** The master's own commands come from this page; the converter relays SCADA's ("SCADA via the converter"). */
export function masterCommandMatches(entry: MasterCommandEntry, values: FilterValues): boolean {
  if (!kindMatches(values.kind, kindOfMasterCommand(entry.kind))) return false;
  const result = entry.state === 'confirmed' ? 'done' : entry.state === 'refused' ? 'refused' : 'waiting';
  if (values.result !== 'all' && result !== values.result) return false;
  const scada = /scada/i.test(entry.who);
  if ((values.from === 'me' && scada) || (values.from === 'scada' && !scada)) return false;
  return textMatch(values.text, entry.kind, entry.address, entry.value, entry.who, entry.text);
}
