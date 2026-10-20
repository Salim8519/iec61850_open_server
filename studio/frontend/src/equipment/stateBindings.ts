import type { EquipmentData, SignalBinding, StateMapping } from '../api/types';

export function stateBindings(mapping?: StateMapping | null): SignalBinding[] {
  if (!mapping) return [];
  return mapping.mode === 'contacts_to_dbpos' ? [mapping.open, mapping.closed] : [mapping.source];
}

export function relayBindings(data: EquipmentData): SignalBinding[] {
  return (data.relay_statuses || []).map(entry => entry.source);
}

/** Measurement list rows (ADR-0090), so signal layers and IED tags see the list. */
export function measurementBindings(data: EquipmentData): SignalBinding[] {
  return (data.measurements || []).map(entry => entry.source);
}

/** Live-value key for one relay status entry or measurement row on the shared equipment store. */
export function relayValueKey(nodeId: string, entryId: string): string {
  return `${nodeId}:${entryId}`;
}
