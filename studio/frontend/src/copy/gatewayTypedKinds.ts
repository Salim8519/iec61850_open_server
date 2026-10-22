// The gateway point kinds of ADR-0107 in one small table: name, own-point name, the protocol
// type IDs and a plain hint. copy/gateway.ts spreads them into its kind lists, so this file
// can move into the shared type catalog as it is.
import type { Protocol } from '../api/gateway';

export type TypedPointKind = 'step_position' | 'command_step' | 'command_setpoint';

export type TypedKindInfo = {
  label: string;
  own: string;
  where: Record<Protocol, string>;
  hint: string;
};

export const TYPED_KINDS: Record<TypedPointKind, TypedKindInfo> = {
  step_position: {
    label: 'Tap position', own: 'Tap position (whole number)',
    where: { iec104: 'M_ST (5 / 32)', modbus: 'Input register, int16', dnp3: 'Analog input' },
    hint: 'The tap changer position (BSC valWTr.posVal), a whole number from -64 to 63.',
  },
  command_step: {
    label: 'Step command', own: 'Step command (raise / lower)',
    where: { iec104: 'C_RC (47 / 60)', modbus: 'Not on Modbus', dnp3: 'CROB (close = raise, trip = lower)' },
    hint: 'Raises or lowers a tap changer (a BSC or ISC control). Select before operate is used when SCADA asks for it.',
  },
  command_setpoint: {
    label: 'Set point', own: 'Set point (a number)',
    where: { iec104: 'C_SE (48-50 / 61-63)', modbus: 'Holding register (FC6 / FC16)', dnp3: 'Analog output (g41)' },
    hint: 'Sends a number to an APC or INC control. ScadaBench divides it by the scale before it reaches the IED.',
  },
};

/** Words of the Send controls of step and set point rows. */
export const TYPED_COMMAND_COPY = {
  raise: 'Raise', lower: 'Lower', value: 'Set point to send', send: 'Send',
  badNumber: 'Type a number, for example 12.5.',
  setpointLabel: (value: number) => `set point ${value}`,
};

/** One column of the table, e.g. the labels, keyed by kind. */
export function typedColumn<T>(pick: (info: TypedKindInfo) => T): Record<TypedPointKind, T> {
  return {
    step_position: pick(TYPED_KINDS.step_position),
    command_step: pick(TYPED_KINDS.command_step),
    command_setpoint: pick(TYPED_KINDS.command_setpoint),
  };
}
