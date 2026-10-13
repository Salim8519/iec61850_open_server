import type { RelayStatus, Signal } from '../api/types';

export type RelayValueState = 'on' | 'off' | 'none' | 'value';

/**
 * Reference without the IED-name part of its logical device, e.g.
 * `IED1Master/LLN0.Mod.stVal` -> `Master/LLN0.Mod.stVal`. The logical-device
 * instance stays, so refs of one IED remain unambiguous; anything that does
 * not start with `<iedName>` followed by an LD instance is returned unchanged.
 */
export function shortRef(ref: string, iedName?: string): string {
  if (!iedName || !ref.startsWith(iedName)) return ref;
  const rest = ref.slice(iedName.length);
  return rest.indexOf('/') > 0 ? rest : ref;
}

/** Name shown for a relay row; untouched default names that equal the ref are shortened. */
export function relayRowLabel(entry: RelayStatus, iedName?: string): string {
  return entry.label === entry.source.ref ? shortRef(entry.label, iedName) : entry.label;
}

/** Default name for a newly added relay signal. */
export function defaultRelayLabel(signal: Signal, iedName?: string): string {
  return (signal.description.trim() || shortRef(signal.ref, iedName)).slice(0, 100);
}

/** Text plus state class; the text alone always identifies the value. */
export function relayValue(value: unknown): { text: string; state: RelayValueState } {
  if (value === undefined || value === null) return { text: '—', state: 'none' };
  if (typeof value === 'boolean') return value ? { text: '1', state: 'on' } : { text: '0', state: 'off' };
  return { text: String(value), state: 'value' };
}

/** A pulse drives the opposite of the current Boolean value; unknown values pulse to true. */
export function pulseTarget(value: unknown): boolean {
  return value !== true;
}
