import type { Control, ExtendedControl, Ied, PanelPoint } from '../api/types.ts';
import { POINT_COPY } from '../copy/pointPanel.ts';

/** Point panel requests (ADR-0102). Display only: the Go engine checks every request again, writes
 *  status, double and value points into the simulated IED (no command) and sends command points
 *  through the normal command path. All of it uses the existing test route of the node. */
export const pointRoute = (nodeId: string): string => `/equipment/${encodeURIComponent(nodeId)}/test`;

/** What a command row sends (ADR-0108): true/false (SPC, DPC), a number (INC, APC), an enum label (ENC)
 *  or "higher"/"lower" (BSC, ISC). */
export type CommandValue = boolean | number | string;
export type PointBody = { action: 'set' | 'send' | 'select' | 'cancel' | 'healthy'; row?: string; value?: CommandValue };

export const setBody = (point: PanelPoint, value: boolean | number): PointBody => ({ action: 'set', row: point.id, value });
export const sendBody = (point: PanelPoint, value: CommandValue): PointBody => ({ action: 'send', row: point.id, value });
export const selectBody = (point: PanelPoint, value: CommandValue): PointBody => ({ action: 'select', row: point.id, value });
export const cancelBody = (point: PanelPoint): PointBody => ({ action: 'cancel', row: point.id });
export const HEALTHY_BODY: PointBody = { action: 'healthy' };

/** What the Go engine answers to "healthy": one entry per point with a healthy value. */
export type HealthyAnswer = { requested: number; applied: number; failed: number;
  results: { row: string; ok: boolean; value?: unknown; error?: string }[] };

/** One sentence for the answer, and the failed rows with their reasons. */
export function healthySentence(answer: HealthyAnswer): string {
  if (!answer.failed) return POINT_COPY.healthyDone(answer.applied);
  return POINT_COPY.healthyPartly(answer.applied, answer.failed);
}

export type CommandChoice = { label: string; value: CommandValue };

/** Single command (SPC): ON or OFF. Double command (DPC): Close is true, Open is false. Typed classes
 *  (INC, ENC, APC, BSC, ISC) have their own inputs, so they have no fixed choices here. */
export function commandChoices(control: Pick<CommandControl, 'cdc'> | undefined): CommandChoice[] {
  if (!control || (control.cdc !== 'SPC' && control.cdc !== 'DPC')) return [];
  return control.cdc === 'DPC' ? [{ label: 'Open', value: false }, { label: 'Close', value: true }]
    : [{ label: 'Send ON', value: true }, { label: 'Send OFF', value: false }];
}

export type PointAvailability = { ok: boolean; reason: '' | 'missing' | 'real' | 'off' | 'stopped' };

/** Points can be changed only on a simulated IED that is running; a real device is read only. */
export function pointAvailability(ied: Ied | undefined, running: ReadonlySet<string>): PointAvailability {
  if (!ied) return { ok: false, reason: 'missing' };
  if (ied.role === 'real') return { ok: false, reason: 'real' };
  if (ied.role === 'off') return { ok: false, reason: 'off' };
  return running.has(ied.id) ? { ok: true, reason: '' } : { ok: false, reason: 'stopped' };
}

/** A control a command row can use: SPC/DPC from `controls`, or a typed one (ADR-0104). */
export type CommandControl = Control | ExtendedControl;

const TYPED_CLASSES = new Set(['INC', 'ENC', 'APC', 'BAC', 'BSC', 'ISC']);

/** Every control a command row may use: the IED's SPC/DPC list, then the typed list of GET /controls/typed
 *  (`extended` is empty until it arrives, or from an engine without it) without refs already in the first. */
export function commandControls(ied: Pick<Ied, 'controls'>, extended: readonly ExtendedControl[]): CommandControl[] {
  const known = new Set(ied.controls.map(control => control.ref));
  return [...ied.controls, ...extended.filter(control => !known.has(control.ref) && TYPED_CLASSES.has(control.cdc))];
}

/** The control of a command row: `controls` first, then the typed list, as the Go engine looks it up. */
export const controlOf = (point: PanelPoint, ied: Ied | undefined, extended: readonly ExtendedControl[]): CommandControl | undefined =>
  ied?.controls.find(control => control.ref === point.control?.ref)
  ?? (ied ? extended.find(control => control.ref === point.control?.ref) : undefined);
