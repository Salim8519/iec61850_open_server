// Reading the engine's picker answer (ADR-0113). Pure: the Go engine counts, filters and finds the
// suggestions; this only orders and words them. An answer without the Go-only fields (the frozen
// Python engine) leaves them undefined, and the picker then behaves as before.
import type { Control, Signal } from '../../api/types.ts';
import { KINDS, type Kind } from '../../points/pointKind.ts';
import { kindOfSignal } from '../signalKind.ts';

export type PickedSignal = Signal & { point_kind?: string; suggested_by?: string };
export type PickerAnswer = {
  items: PickedSignal[]; total: number;
  kind_counts?: Partial<Record<Kind, number>>; ln_counts?: Record<string, number>; suggested?: PickedSignal[];
};
export type Chip = { kind: Kind; count: number };
export type ObjectOption = { value: string; label: string; count: number };

/** Go's kind when the engine sent it, otherwise the browser's own rule (older engines). */
export function kindOfPicked(signal: PickedSignal, controls?: readonly Pick<Control, 'ref' | 'cdc'>[]): Kind {
  const sent = signal.point_kind as Kind | undefined;
  return sent && KINDS.includes(sent) ? sent : kindOfSignal(signal, undefined, controls);
}

/** One chip per kind that has rows, in the kinds' display order; the active kind always keeps its chip. */
export function kindChips(counts: Partial<Record<Kind, number>>, active: string): Chip[] {
  return KINDS.filter(kind => (counts[kind] ?? 0) > 0 || kind === active).map(kind => ({ kind, count: counts[kind] ?? 0 }));
}

/** Rows of every type together (the "All types" chip). */
export const allCount = (counts: Partial<Record<Kind, number>>) =>
  Object.values(counts).reduce((sum, count) => sum + (count ?? 0), 0);

const natural = new Intl.Collator('en', { numeric: true, sensitivity: 'base' });

/** The object (logical node) choices in natural order (GGIO2 before GGIO10). When every node is in one
 *  logical device the label is the node alone ("ConInGGIO1"), otherwise "LD/LN". The chosen node stays
 *  in the list even when nothing of it is left. */
export function objectOptions(counts: Record<string, number>, active: string): ObjectOption[] {
  const keys = Object.keys(counts);
  if (active && !keys.includes(active)) keys.push(active);
  const devices = new Set(keys.map(key => key.split('/')[0]));
  const label = (key: string) => (devices.size === 1 && key.includes('/') ? key.slice(key.indexOf('/') + 1) : key);
  return keys.sort((a, b) => natural.compare(a, b)).map(key => ({ value: key, label: label(key), count: counts[key] ?? 0 }));
}

/** True when the answer came from the Go engine's picker support (type chips, objects, suggestions). */
export const isSmart = (answer?: PickerAnswer) => !!answer?.kind_counts;
