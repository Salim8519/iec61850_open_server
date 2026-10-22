import { api } from '../api/client';

/** The four tap changer commands; the Go engine routes them (BSC/ISC step, SPC pair, Auto/Manual). */
export type TapAction = 'raise' | 'lower' | 'auto' | 'manual';
export type TapAnswer = { action: TapAction; wire_value: boolean | number | null; tap?: number | null };

/** The one place a tap changer command is sent (ADR-0109): a direct operate through
 *  POST /equipment/<node>/operate. Select-before-operate is not done here: at merge the shared
 *  SBO hook (useSelectOperate, change 0171) wraps this function, and nothing else changes. */
export function sendTapCommand(nodeId: string, action: TapAction): Promise<TapAnswer> {
  return api<TapAnswer>(`/equipment/${encodeURIComponent(nodeId)}/operate`, 'POST', { action });
}

/** The words shown after a command, e.g. "Raise done · tap 10". */
export function tapAnswerText(answer: TapAnswer): string {
  const name = answer.action[0].toUpperCase() + answer.action.slice(1);
  return typeof answer.tap === 'number' ? `${name} done · tap ${answer.tap}` : `${name} done`;
}
