// One point-kind classifier for the whole app (ADR-0095). Every list asks this module what kind
// of point a row is, so a double point looks the same in Devices, the SLD pickers, the gateway,
// the master, SNMP and GOOSE. Pure functions only. The kind comes from the protocol's own
// type information (FC, basic type, data class, type id, group); never from a reference name.

export type Kind = 'SP' | 'DP' | 'IS' | 'MV' | 'CT' | 'SC' | 'DC' | 'SE' | 'CM' | 'CF' | 'TX' | 'AT' | 'UN';

/** Display order of the kinds: chips, legends and counts follow it. */
export const KINDS: readonly Kind[] = ['SP', 'DP', 'IS', 'MV', 'CT', 'SC', 'DC', 'SE', 'CM', 'CF', 'TX', 'AT', 'UN'];
const COMMANDS = new Set<Kind>(['SC', 'DC', 'SE', 'CM']);

export const isCommandKind = (kind: Kind) => COMMANDS.has(kind);

const NUMERIC = /^(INT(8|16|32|64|128)U?|FLOAT(32|64))$/;
const TEXTUAL = /^(VisString|Unicode|Octet)/;

/** Data classes that decide the kind on their own (from the SCL DOType, when the engine sends it). */
const BY_CLASS: Record<string, Kind> = {
  SPS: 'SP', ACT: 'SP', ACD: 'SP', DPS: 'DP', INS: 'IS', ENS: 'IS', BCR: 'CT',
  MV: 'MV', CMV: 'MV', SAV: 'MV', WYE: 'MV', DEL: 'MV', SEQ: 'MV', HMV: 'MV', HWYE: 'MV', HDEL: 'MV',
  SPC: 'SC', DPC: 'DC', APC: 'SE', BAC: 'SE', INC: 'SE', ENC: 'SE', BSC: 'SE', ISC: 'SE',
};

/** IEC 61850 data attribute: functional constraint + basic type, refined by the data class when known.
 *  Without a data class a Boolean command stays a plain command (CM): SPC and DPC both use BOOLEAN. */
export function kindOf61850(fc: string, basicType: string, dataClass?: string | null): Kind {
  if (basicType === 'Quality' || basicType === 'Timestamp') return 'AT';
  if (fc === 'CO') {
    const known = dataClass ? BY_CLASS[dataClass] : undefined;
    if (known && COMMANDS.has(known)) return known;
    return NUMERIC.test(basicType) ? 'SE' : 'CM';
  }
  if (['SP', 'SG', 'SE', 'CF'].includes(fc)) return 'CF';
  if (fc === 'DC' || fc === 'EX' || TEXTUAL.test(basicType)) return 'TX';
  if (basicType === 'Dbpos') return 'DP';
  if (basicType === 'BOOLEAN' && fc === 'ST') return 'SP';
  if (fc === 'MX' && NUMERIC.test(basicType)) return 'MV';
  if (fc === 'ST' && (basicType === 'Enum' || NUMERIC.test(basicType))) {
    const known = dataClass ? BY_CLASS[dataClass] : undefined;
    return known === 'CT' ? 'CT' : 'IS';
  }
  return 'UN';
}

/** Data class of a data object (SPS, DPC …) when that is all a list knows. */
export const kindOfDataClass = (dataClass: string): Kind => BY_CLASS[dataClass] ?? 'UN';

/** IEC 60870-5-104 / -101 type identifier name such as M_DP_TB_1 or C_SE_NC_1. */
export function kindOfIec104(typeId: string): Kind {
  const prefix = typeId.slice(0, 4);
  const map: Record<string, Kind> = {
    M_SP: 'SP', M_DP: 'DP', M_ST: 'IS', M_BO: 'IS', M_ME: 'MV', M_IT: 'CT',
    C_SC: 'SC', C_DC: 'DC', C_RC: 'SE', C_SE: 'SE', C_BO: 'SE',
  };
  return map[prefix] ?? 'UN';
}

/** Gateway point kinds (what ScadaBench serves upstream over IEC 104, Modbus or DNP3). An "integer"
 *  point is served as a scaled measured value (M_ME_NB), so it is MV like the master's scaled group. */
export function kindOfGatewayPoint(kind: string): Kind {
  const map: Record<string, Kind> = {
    single: 'SP', double: 'DP', float: 'MV', integer: 'MV', command_single: 'SC', command_double: 'DC',
    // ADR-0107: a tap position is an integer status (M_ST); step and set point commands are set points.
    step_position: 'IS', command_step: 'SE', command_setpoint: 'SE',
  };
  return map[kind] ?? 'UN';
}

/** Master / converter point groups read from a device (IEC 104, DNP3 and Modbus groups). */
export function kindOfMasterGroup(group: string): Kind {
  const map: Record<string, Kind> = {
    single: 'SP', binary: 'SP', binary_output: 'SP', coil: 'SP', discrete_input: 'SP',
    double: 'DP', double_binary: 'DP', step: 'IS', bitstring: 'IS',
    normalized: 'MV', scaled: 'MV', float: 'MV', analog: 'MV', analog_output: 'MV',
    holding_register: 'MV', input_register: 'MV', counter: 'CT', frozen_counter: 'CT',
  };
  return map[group] ?? 'UN';
}

/** Commands a master sends to a device (single, double, set points, outputs, register writes). */
export function kindOfMasterCommand(kind: string): Kind {
  if (kind === 'single' || kind === 'binary_output' || kind === 'coil') return 'SC';
  if (kind === 'double') return 'DC';
  // A regulating step (C_RC) is SE, as the gateway's C_RC and command_step are (change 0178).
  if (kind === 'regulating') return 'SE';
  if (kind.startsWith('setpoint') || kind.startsWith('analog_output') || kind.startsWith('register')) return 'SE';
  return 'UN';
}

/** SNMP value types (SMIv2). An INTEGER may be a named state, so it is an integer status. */
export function kindOfSnmp(type: string): Kind {
  const map: Record<string, Kind> = {
    integer: 'IS', gauge32: 'MV', counter32: 'CT', counter64: 'CT', timeticks: 'CT',
    octets: 'TX', oid: 'TX', ipaddress: 'TX', opaque: 'TX',
  };
  return map[type.toLowerCase()] ?? 'UN';
}

/** The class of the data object that holds this attribute (refs from GET /ieds/<id>/data-classes):
 *  the longest object reference that equals the attribute reference or is followed by "." in it. */
export function dataClassOf(classes: Record<string, string>, ref: string): string | undefined {
  for (let at = ref.length; at > 0; at = ref.lastIndexOf('.', at - 1)) {
    const found = classes[ref.slice(0, at)];
    if (found) return found;
  }
  return undefined;
}

/** How many rows of each kind a list has, for the type chips. */
export function countKinds<T>(rows: readonly T[], kindOf: (row: T) => Kind): Partial<Record<Kind, number>> {
  const counts: Partial<Record<Kind, number>> = {};
  for (const row of rows) { const kind = kindOf(row); counts[kind] = (counts[kind] ?? 0) + 1; }
  return counts;
}
