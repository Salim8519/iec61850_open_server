import type { GatewayPoint } from '../../../api/gateway';

/** A stable row key, also for rows that are not saved yet (no id). */
export const pointKey = (row: GatewayPoint, index: number) => row.id ?? `${row.ied}|${row.ref}|${row.kind}|${index}`;

/** Keep the user's unsaved edits but show the latest live values from the backend. */
export function mergeValues(draft: GatewayPoint[], fresh: GatewayPoint[]): GatewayPoint[] {
  const live = new Map(fresh.map(row => [`${row.ied}|${row.ref}|${row.kind}`, row]));
  return draft.map(row => {
    const current = live.get(`${row.ied}|${row.ref}|${row.kind}`);
    return current ? { ...row, value_text: current.value_text, online: current.online, ied_name: current.ied_name,
      monitor_only: current.monitor_only } : row;
  });
}
