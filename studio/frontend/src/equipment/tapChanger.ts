import type { AvrSettings, Ied, TapChanger } from '../api/types.ts';
import { AVR_BLOCKED, AVR_STATE, TAP_COPY } from '../copy/tapChanger.ts';

/** Display helpers of the tap-changer transformer (ADR-0109). The Go engine owns the tap, the regulator and
 *  every decision; these only turn what it publishes into words. */

/** Whole-number kinds a tap position (or an integer Auto/Manual status) may have. */
export const TAP_KINDS = ['INT8', 'INT16', 'INT32', 'INT64', 'INT8U', 'INT16U', 'INT24U', 'INT32U'];
export const MODE_KINDS = ['BOOLEAN', ...TAP_KINDS];

export const DEFAULT_AVR: AvrSettings = { enabled: false, target_kv: 11, bandwidth_pct: 1.5, delay_s: 30,
  step_pct: 1.25, hv_rated_kv: 132, lv_rated_kv: 11, source_kv: 132 };

/** A new, unbound tap changer: taps 1 to 17, nominal 9 (the engine gives the same defaults). */
export function newTapChanger(): TapChanger {
  return { tap_source: null, command_mode: 'none', step_control: null, raise_control: null, lower_control: null,
    mode_source: null, mode_control: null, auto_value: 1, manual_value: 0, voltage_source: null,
    min_tap: 1, max_tap: 17, nominal_tap: 9, avr: { ...DEFAULT_AVR } };
}

/** Raise and lower can be sent: one step control, or both single commands (the engine checks the same). */
export function commandReady(tap: TapChanger): boolean {
  return tap.command_mode === 'step' ? !!tap.step_control :
    tap.command_mode === 'spc_pair' && !!tap.raise_control && !!tap.lower_control;
}

/** "Tap 9 / 1–17", or short words for a missing or not yet known tap (they fit the card on one line). */
export function tapText(tap: TapChanger, value: unknown): string {
  if (!tap.tap_source) return TAP_COPY.notMapped;
  return typeof value === 'number' ? `Tap ${value} / ${tap.min_tap}–${tap.max_tap}` : TAP_COPY.awaiting;
}

/** The regulator state the engine publishes as "<node>:avr". */
export type AvrState = { auto?: boolean; regulator?: boolean; state?: string; reason?: string;
  remaining_s?: number; deviation_pct?: number; lv_kv?: number };

/** Auto or Manual: from the bound status ("<node>:mode") when there is one, else the regulator's own mode. */
export function isAuto(tap: TapChanger, modeValue: unknown, avr: AvrState | undefined): boolean | undefined {
  if (tap.mode_source) {
    if (typeof modeValue === 'boolean') return modeValue;
    if (typeof modeValue === 'number') return modeValue === tap.auto_value;
    return undefined;
  }
  return avr?.auto ?? false;
}

/** Mode badge words; never colour alone (an icon and words go with it). */
export function modeWords(auto: boolean | undefined): string {
  return auto === undefined ? 'Mode unknown' : auto ? TAP_COPY.auto : TAP_COPY.manual;
}

/** One line about the regulator, e.g. "Out of band · waiting · 12 s"; empty when there is none. */
export function avrLine(avr: AvrState | undefined): string {
  if (!avr?.regulator || !avr.state || avr.state === 'manual') return ''; // the mode badge already says Manual
  const words = AVR_STATE[avr.state] ?? avr.state;
  if (avr.state === 'blocked') return `${words}: ${AVR_BLOCKED[avr.reason ?? ''] ?? avr.reason ?? ''}`;
  if (avr.state === 'timing' && typeof avr.remaining_s === 'number') return `${words} ${avr.remaining_s} s`;
  return words;
}

/** The LV voltage to show: the measured point when bound, else the regulator's calculated value. */
export function lvText(voltageValue: unknown, avr: AvrState | undefined): string {
  const kv = typeof voltageValue === 'number' ? voltageValue : avr?.lv_kv;
  return typeof kv === 'number' ? `LV ${kv.toFixed(2)} kV` : '';
}

/** Whether Raise / Lower can be pressed now, and why not. */
export function stepBlock(tap: TapChanger, tapValue: unknown, auto: boolean | undefined, direction: 'raise' | 'lower'):
  string | null {
  if (!commandReady(tap)) return TAP_COPY.noCommand;
  if (auto) return TAP_COPY.inAuto;
  if (typeof tapValue === 'number') {
    if (direction === 'raise' && tapValue >= tap.max_tap) return `Already at the highest tap (${tap.max_tap}).`;
    if (direction === 'lower' && tapValue <= tap.min_tap) return `Already at the lowest tap (${tap.min_tap}).`;
  }
  return null;
}

/** Controls an IED offers for each tap changer role. BSC/ISC and ENC come from typed controls (change 0167). */
export function tapControls(ied: Ied | undefined) {
  const typed = ied?.extended_controls ?? [];
  return {
    step: typed.filter(control => control.cdc === 'BSC' || control.cdc === 'ISC'),
    spc: (ied?.controls ?? []).filter(control => control.cdc === 'SPC'),
    mode: [...(ied?.controls ?? []).filter(control => control.cdc === 'SPC'),
      ...typed.filter(control => control.cdc === 'ENC')],
  };
}

/** Plain-words check of the tap range before it is saved (the engine checks it again). */
export function rangeProblem(tap: Pick<TapChanger, 'min_tap' | 'max_tap' | 'nominal_tap'>): string | null {
  if (tap.min_tap >= tap.max_tap) return 'The lowest tap must be smaller than the highest tap.';
  if (tap.nominal_tap < tap.min_tap || tap.nominal_tap > tap.max_tap) return 'The nominal tap must be inside the range.';
  return null;
}
