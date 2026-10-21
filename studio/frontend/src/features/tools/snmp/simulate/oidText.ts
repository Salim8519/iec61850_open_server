import { snmpApi, type SnmpPoint } from '../../../../api/snmp';

const NUMERIC = /^\.?\d+(\.\d+)+$/;

/** The OID the user typed (dotted, or a name a MIB knows), refused when the device already has it. */
export async function oidOf(text: string, points: SnmpPoint[]): Promise<string> {
  const typed = text.trim();
  const oid = NUMERIC.test(typed) ? typed.replace(/^\./, '') : (await snmpApi.resolve(typed)).oid;
  if (points.some(point => point.oid === oid)) throw new Error(`${oid} is already a value of this device`);
  return oid;
}
