// Which point type a picker slot opens with (ADR-0113). Pure: it reads only what the slot accepts
// (basic types and functional constraint), never a reference name.
import type { Kind } from '../../points/pointKind.ts';

const NUMERIC = /^(INT(8|16|24|32|64|128)U?|FLOAT(32|64))$/;

export type Slot = { kinds?: readonly string[]; fc?: string; command?: Kind };

/** The kind a slot is meant for, or undefined when it takes several kinds alike:
 *  - a command slot names its command kind (a number command is a set point);
 *  - a slot that takes Dbpos opens on double points (one Boolean point still works under All types);
 *  - Boolean only: single points;
 *  - numbers (and Enum): measured values in MX or when floats are allowed, integer status in ST or
 *    when only whole numbers are allowed (a tap position). */
export function preferredKind({ kinds, fc, command }: Slot): Kind | undefined {
  if (command) return command;
  if (fc === 'CO') return kinds?.length && kinds.every(kind => NUMERIC.test(kind)) ? 'SE' : undefined;
  if (!kinds?.length) return undefined;
  if (kinds.includes('Dbpos')) return 'DP';
  if (kinds.every(kind => kind === 'BOOLEAN')) return 'SP';
  if (!kinds.every(kind => NUMERIC.test(kind) || kind === 'Enum')) return undefined;
  if (fc === 'MX') return 'MV';
  if (fc === 'ST') return 'IS';
  return kinds.some(kind => kind.startsWith('FLOAT')) ? 'MV' : 'IS';
}

/** True when the slot can list quality or time attributes, so "Show quality and time" changes something. */
export const takesHelpers = (kinds?: readonly string[]) =>
  !kinds?.length || kinds.some(kind => kind === 'Quality' || kind === 'Timestamp' || kind === 'EntryTime');
