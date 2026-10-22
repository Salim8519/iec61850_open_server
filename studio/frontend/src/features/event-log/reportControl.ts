import type { StudioEvent } from '../../api/types';
import { iedName } from './iedNames.ts';

/** Report control block events of a simulated IED (native `report_control`, libiec61850 RCB event kinds).
 *  "Report created" comes every time a report is built, so Activity folds those into one row per block and
 *  minute (change 0177) instead of filling its 500 rows with them. */
export const REPORT_CREATED = 9;

const ACTIONS: Record<number, string> = {
  0: 'settings read', 2: 'reservation released', 3: 'reserved', 4: 'enabled', 5: 'disabled',
  6: 'general interrogation asked', 7: 'buffer emptied', 8: 'buffer overflowed (oldest reports lost)',
};

/** The Activity fold key: one row per IED and report block for "report created", nothing for the rest. */
export function reportFoldKey(event: StudioEvent): string | null {
  if (event.kind !== 'report_control' || Number(event.native_kind) !== REPORT_CREATED) return null;
  return `${String(event.ied_id ?? '')}|${String(event.name ?? '')}`;
}

/** Details of one report control row, in plain words. */
export function reportControlDetails(event: StudioEvent, count = 1): string {
  const owner = event.ied_id ? `${iedName(event.ied_id)} ` : '';
  const block = `${owner}${String(event.name || 'report block')}`;
  const kind = Number(event.native_kind);
  const by = event.peer ? ` by ${String(event.peer)}` : '';
  if (kind === REPORT_CREATED) return count > 1 ? `${block} / ${count} reports created` : `${block} / report created`;
  if (kind === 1) {
    const what = event.parameter ? `${String(event.parameter)} changed` : 'setting changed';
    const refused = Number(event.error || 0) ? ` (refused, error ${String(event.error)})` : '';
    return `${block} / ${what}${by}${refused}`;
  }
  return `${block} / ${ACTIONS[kind] ?? `event ${String(event.native_kind ?? 'unknown')}`}${by}`;
}
