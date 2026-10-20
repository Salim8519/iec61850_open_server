import type { GatewayPoint } from '../../../api/gateway';

/** How a row is matched with the backend's: own points by id, IED points by IED, reference and type. */
const liveKey = (row: GatewayPoint) => row.source === 'manual' ? `own|${row.id}` : `${row.ied}|${row.ref}|${row.kind}`;

/** A stable row key, also for rows that are not saved yet (no id). */
export const pointKey = (row: GatewayPoint, index: number) => row.id ?? `${row.ied}|${row.ref}|${row.kind}|${index}`;

/** Keep the user's unsaved edits but show the latest live values from the backend. */
export function mergeValues(draft: GatewayPoint[], fresh: GatewayPoint[]): GatewayPoint[] {
  const live = new Map(fresh.map(row => [liveKey(row), row]));
  return draft.map(row => {
    const current = row.draft ? undefined : live.get(liveKey(row));
    return current ? { ...row, value_text: current.value_text, online: current.online, ied_name: current.ied_name,
      monitor_only: current.monitor_only } : row;
  });
}
