import type { GatewayPoint } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway.ts';

/** The two choices of a command point: Close/Open for a double command, On/Off for a single one. */
export function commandLabels(row: GatewayPoint): [string, string] {
  return row.kind === 'command_double' ? ['Close', 'Open'] : ['On', 'Off'];
}

/** Why a command row cannot send now, or '' when its Send buttons are offered. */
export function blockedReason(row: GatewayPoint, unsaved: boolean): string {
  if (unsaved || !row.id) return GATEWAY_COPY.saveFirst;
  return row.monitor_only ? GATEWAY_COPY.commandsOff : '';
}
