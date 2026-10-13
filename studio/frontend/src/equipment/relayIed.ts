import type { RelayStatus } from '../api/types';

/** Unique IED ids across relay entries, in first-seen order. */
export function relayIedIds(entries: RelayStatus[]): string[] {
  const ids: string[] = [];
  for (const entry of entries) {
    if (!ids.includes(entry.source.ied_id)) ids.push(entry.source.ied_id);
  }
  return ids;
}

/** The single IED a relay belongs to, or null when it has no entries. */
export function resolveRelayIedId(entries: RelayStatus[]): string | null {
  return entries.length ? entries[0].source.ied_id : null;
}

/** True when entries span more than one IED and need reassignment. */
export function isMixedRelay(entries: RelayStatus[]): boolean {
  return relayIedIds(entries).length > 1;
}
