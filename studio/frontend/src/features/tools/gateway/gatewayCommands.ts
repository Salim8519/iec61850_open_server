import type { CommandValue, GatewayPoint } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway.ts';
import { TYPED_COMMAND_COPY } from '../../../copy/gatewayTypedKinds.ts';

/** The two choices of a command point: Close/Open for a double command, Raise/Lower for a step command
 *  (ADR-0107), On/Off for a single one. */
export function commandLabels(row: Pick<GatewayPoint, 'kind'>): [string, string] {
  if (row.kind === 'command_step') return [TYPED_COMMAND_COPY.raise, TYPED_COMMAND_COPY.lower];
  return row.kind === 'command_double' ? ['Close', 'Open'] : ['On', 'Off'];
}

/** What the two buttons send: true/false, or "higher"/"lower" for a step command. */
export function commandValues(row: Pick<GatewayPoint, 'kind'>): [CommandValue, CommandValue] {
  return row.kind === 'command_step' ? ['higher', 'lower'] : [true, false];
}

/** A typed set point, or the plain reason it cannot be sent. */
export function parseSetpoint(text: string): { value: number } | { error: string } {
  const trimmed = text.trim().replace(',', '.');
  const value = Number(trimmed);
  return trimmed === '' || !Number.isFinite(value) ? { error: TYPED_COMMAND_COPY.badNumber } : { value };
}

/** Why a command row cannot send now, or '' when its Send buttons are offered. */
export function blockedReason(row: GatewayPoint, unsaved: boolean): string {
  if (unsaved || !row.id || row.draft) return GATEWAY_COPY.saveFirst;
  return row.monitor_only ? GATEWAY_COPY.commandsOff : '';
}
