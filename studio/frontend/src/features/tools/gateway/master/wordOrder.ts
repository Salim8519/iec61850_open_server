import type { CommandKind } from '../../../../api/master';

/** A Modbus write of two registers (FC16), where the word order matters (change 0178). */
export const hasWordOrder = (kind: CommandKind) => kind === 'registers_int32' || kind === 'registers_float';
