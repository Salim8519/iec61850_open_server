import type { Measurement, Signal, SignalBinding } from '../api/types.ts';

/** Measurement list rules (ADR-0090); the Go engine checks the same limits. */
export const MAX_MEASUREMENTS = 50;
export const MAX_DECIMALS = 6;
export const MAX_UNIT = 16;
export const SCALE_LIMIT = 1e9;
/** Signal kinds a measurement row accepts: integers and floats only. */
export const NUMERIC_KINDS = ['INT8', 'INT16', 'INT32', 'INT64', 'INT128', 'INT8U', 'INT16U', 'INT24U', 'INT32U',
  'FLOAT32', 'FLOAT64'];

export const MEASUREMENT_DEFAULTS = { scale: 1, offset: 0, unit: '', decimals: 2 } as const;

/** One-click scale choices; each says what it is for in plain words. */
export const SCALE_PRESETS: { scale: number; label: string; hint: string }[] = [
  { scale: 1, label: '× 1', hint: 'Show the value as the IED sends it' },
  { scale: 0.001, label: '× 0.001', hint: 'For example V to kV, or A to kA' },
  { scale: 1000, label: '× 1000', hint: 'For example kV to V, or MW to kW' },
];

export const MEASUREMENT_COPY = {
  scaleHint: 'Shown value = raw value × scale + offset. Example: raw 11000 × 0.001 = 11.00 kV.',
  bulkHint: 'Tick rows to give them the same scale, unit or decimals in one go.',
  empty: 'Add measurements in the sheet',
  noValue: 'No value yet',
  stopped: 'IED stopped',
  badQuality: 'Bad quality: the IED says this value may not be trusted',
} as const;

export type MeasurementState = 'value' | 'none' | 'stopped' | 'text';

/** Raw value as a number, or null when it is not a finite number. */
export function numericValue(raw: unknown): number | null {
  if (typeof raw === 'number') return Number.isFinite(raw) ? raw : null;
  if (typeof raw === 'bigint') return Number(raw);
  if (typeof raw === 'string' && raw.trim() !== '') {
    const parsed = Number(raw);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

/** Display = raw × scale + offset, rounded to `decimals`, then the unit. Text alone carries the state. */
export function formatMeasurement(raw: unknown, entry: Pick<Measurement, 'scale' | 'offset' | 'unit' | 'decimals'>,
  stopped = false): { text: string; state: MeasurementState } {
  if (stopped) return { text: 'Stopped', state: 'stopped' };
  if (raw === undefined || raw === null) return { text: '—', state: 'none' };
  const number = numericValue(raw);
  if (number === null) return { text: String(raw), state: 'text' };
  const shown = number * entry.scale + entry.offset;
  const digits = Math.min(MAX_DECIMALS, Math.max(0, Math.round(entry.decimals)));
  // Avoid "-0.00" for a tiny negative value.
  const fixed = (Object.is(Number(shown.toFixed(digits)), -0) ? 0 : shown).toFixed(digits);
  return { text: entry.unit ? `${fixed} ${entry.unit}` : fixed, state: 'value' };
}

/** IEC 61850 quality: 0 is good; anything else is flagged (shown in words, never colour alone). */
export function qualityBad(quality: unknown): boolean {
  return quality !== undefined && quality !== null && Number(quality) !== 0;
}

/** A new row for a chosen signal: its description or short reference as the name, default scaling. */
export function newMeasurement(binding: SignalBinding, signal: Signal, iedName?: string,
  id: string = crypto.randomUUID()): Measurement {
  const short = iedName && signal.ref.startsWith(iedName) && signal.ref.indexOf('/') > iedName.length
    ? signal.ref.slice(iedName.length) : signal.ref;
  return { id, label: (signal.description.trim() || short).slice(0, 100), source: binding, ...MEASUREMENT_DEFAULTS };
}

export type MeasurementPatch = Partial<Pick<Measurement, 'scale' | 'offset' | 'unit' | 'decimals'>>;

/** Gives every selected row the same settings; other rows and fields stay as they are. */
export function applyToRows(entries: Measurement[], ids: ReadonlySet<string>, patch: MeasurementPatch): Measurement[] {
  return entries.map(entry => ids.has(entry.id) ? { ...entry, ...patch } : entry);
}

/** Moves one row up (-1) or down (+1); out-of-range moves change nothing. */
export function moveRow<T>(entries: T[], index: number, direction: number): T[] {
  const target = index + direction;
  if (target < 0 || target >= entries.length) return entries;
  const next = [...entries];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

/** Parses a typed scale or offset; an error sentence when it cannot be used. */
export function parseNumber(text: string, name: 'Scale' | 'Offset'): { value?: number; error?: string } {
  const value = Number(text.trim().replace(',', '.'));
  if (text.trim() === '' || !Number.isFinite(value)) return { error: `${name} must be a number, for example 0.001.` };
  if (Math.abs(value) > SCALE_LIMIT) return { error: `${name} must be between -1000000000 and 1000000000.` };
  return { value };
}

/** Parses typed decimals (0 to 6 whole places). */
export function parseDecimals(text: string): { value?: number; error?: string } {
  const value = Number(text.trim());
  if (!Number.isInteger(value) || value < 0 || value > MAX_DECIMALS) {
    return { error: `Decimals must be a whole number from 0 to ${MAX_DECIMALS}.` };
  }
  return { value };
}

export type SettingKey = keyof MeasurementPatch;

/** Turns one typed setting into a patch, or the sentence that says what to type instead. */
export function settingFromText(key: SettingKey, text: string): { patch?: MeasurementPatch; error?: string } {
  if (key === 'unit') {
    const unit = text.trim();
    return unit.length > MAX_UNIT ? { error: `A unit can have at most ${MAX_UNIT} characters.` } : { patch: { unit } };
  }
  const parsed = key === 'decimals' ? parseDecimals(text) : parseNumber(text, key === 'scale' ? 'Scale' : 'Offset');
  return parsed.error !== undefined ? { error: parsed.error } : { patch: { [key]: parsed.value } };
}

/** The settings all selected rows share, or undefined for a field where they differ. */
export function sharedSettings(entries: Measurement[], ids: ReadonlySet<string>): MeasurementPatch {
  const chosen = entries.filter(entry => ids.has(entry.id));
  const result: MeasurementPatch = {};
  if (!chosen.length) return result;
  for (const key of ['scale', 'offset', 'unit', 'decimals'] as const) {
    if (chosen.every(entry => entry[key] === chosen[0][key])) (result as Record<string, unknown>)[key] = chosen[0][key];
  }
  return result;
}
