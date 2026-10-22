import type { PanelPoint } from '../api/types.ts';
import type { Kind } from '../points/pointKind.ts';
import { formatMeasurement, MAX_DECIMALS, numericValue, SCALE_LIMIT, settingFromText, type MeasurementPatch, type SettingKey }
  from './measurementDisplay.ts';
import { DOUBLE_POSITIONS } from './pointPanel.ts';

/** How a row's live value reads. The text always says it; the state picks a marker, never colour alone. */
export type PointState = 'on' | 'off' | 'position' | 'value' | 'none' | 'stopped' | 'text';
export type PointShown = { text: string; state: PointState };

const POSITION_WORDS = new Map(DOUBLE_POSITIONS.map(item => [item.raw, item.word]));

/** The word for a Dbpos position, or the raw number when it is none of the four. */
export const positionWord = (raw: number): string => POSITION_WORDS.get(raw) ?? String(raw);

/** The live value of a status, double or value row as words (ON, OFF, Open, Closed, 11.00 kV …). */
export function showPoint(point: PanelPoint, raw: unknown, stopped = false): PointShown {
  if (stopped) return { text: 'Stopped', state: 'stopped' };
  if (raw === undefined || raw === null) return { text: '—', state: 'none' };
  if (point.type === 'status') {
    return typeof raw === 'boolean' ? { text: raw ? 'ON' : 'OFF', state: raw ? 'on' : 'off' } : { text: String(raw), state: 'text' };
  }
  if (point.type === 'double') {
    const number = numericValue(raw);
    return number === null ? { text: String(raw), state: 'text' } : { text: positionWord(number), state: 'position' };
  }
  return formatMeasurement(raw, point);
}

/** True when the live value equals the healthy value (numbers within float32 rounding). */
export function isHealthy(point: PanelPoint, raw: unknown): boolean {
  if (point.healthy === null) return true;
  if (typeof point.healthy === 'boolean') return raw === point.healthy;
  const number = numericValue(raw);
  return number !== null && Math.abs(number - point.healthy) <= 1e-6 * Math.max(1, Math.abs(point.healthy));
}

/** The healthy value as words, in the unit the row shows ("ON", "Closed", "230.00 V"); empty when none. */
export function healthyText(point: PanelPoint): string {
  if (point.healthy === null) return '';
  return showPoint(point, point.healthy).text;
}

export type PanelSummary = {
  counts: Record<'status' | 'double' | 'command' | 'value', number>;
  pinned: { id: string; label: string; text: string; state: PointState }[];
  /** Rows with a healthy value, and how many of them do not hold it now (no value counts as not healthy). */
  withHealthy: number; notHealthy: number;
};

/** What the small canvas card shows. `running` is false when the panel's IED is not running: nothing
 *  is then called unhealthy, because there is no value to judge. */
export function summarize(points: readonly PanelPoint[], valueOf: (point: PanelPoint) => unknown, running: boolean): PanelSummary {
  const summary: PanelSummary = { counts: { status: 0, double: 0, command: 0, value: 0 }, pinned: [], withHealthy: 0, notHealthy: 0 };
  for (const point of points) {
    summary.counts[point.type] += 1;
    if (point.type === 'command') continue;
    const raw = valueOf(point);
    if (point.pinned && point.type === 'value') {
      const shown = showPoint(point, raw, !running);
      summary.pinned.push({ id: point.id, label: point.label, text: shown.text, state: shown.state });
    }
    if (point.healthy !== null) {
      summary.withHealthy += 1;
      if (running && !isHealthy(point, raw)) summary.notHealthy += 1;
    }
  }
  return summary;
}

/** Typed display value → the raw value the IED holds: (typed − offset) ÷ scale. */
export function displayToRaw(point: Pick<PanelPoint, 'scale' | 'offset'>, text: string): { value?: number; error?: string } {
  const typed = Number(text.trim().replace(',', '.'));
  if (text.trim() === '' || !Number.isFinite(typed)) return { error: 'Type a number, for example 230.' };
  const raw = (typed - point.offset) / point.scale;
  if (!Number.isFinite(raw) || Math.abs(raw) > SCALE_LIMIT * SCALE_LIMIT) return { error: 'That number is too large.' };
  // 11.04 ÷ 0.001 is 11040.000000000002 in floating point: cut that noise before the IED sees it.
  return { value: Number(raw.toPrecision(12)) };
}

/** The raw value as the text a person would type into the row (no unit). */
export function rawToInput(point: PanelPoint, raw: unknown): string {
  const number = numericValue(raw);
  if (number === null) return '';
  const digits = Math.min(MAX_DECIMALS, Math.max(0, Math.round(point.decimals)));
  return (number * point.scale + point.offset).toFixed(digits);
}

/** A value row read from a status (ST) attribute: an integer status such as an operation counter's stVal
 *  or a tap position (valWTr.posVal). Measured (MX) values end in mag.f, cVal, mxVal and the like. */
const STATUS_VALUE = /\.(stVal|posVal)$/;

/** The universal point kind of a row (ADR-0095): SP, DP, MV, IS for a value read from a status
 *  (stVal, posVal; change 0178), or SC / DC for a command by its control; a typed command (INC, ENC,
 *  APC, BSC, ISC, ADR-0108) is a set point (SE), its class shown beside it. */
export function kindOfPoint(point: PanelPoint, control?: { cdc: string }): Kind {
  if (point.type === 'status') return 'SP';
  if (point.type === 'double') return 'DP';
  if (point.type === 'value') return STATUS_VALUE.test(point.source?.ref ?? '') ? 'IS' : 'MV';
  if (control && ['INC', 'ENC', 'APC', 'BAC', 'BSC', 'ISC'].includes(control.cdc)) return 'SE';
  return control?.cdc === 'DPC' ? 'DC' : control?.cdc === 'SPC' ? 'SC' : 'CM';
}

/** A typed scale, offset, unit or decimals: the measurement rules, and a scale of 0 is refused because
 *  the typed value could not be turned back into the IED's raw value. */
export function pointSettingFromText(key: SettingKey, text: string): { patch?: MeasurementPatch; error?: string } {
  const result = settingFromText(key, text);
  if (key === 'scale' && result.patch?.scale === 0) {
    return { error: 'Scale cannot be 0, because the shown value could then never change.' };
  }
  return result;
}
