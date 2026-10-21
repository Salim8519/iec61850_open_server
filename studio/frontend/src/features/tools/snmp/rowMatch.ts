import type { ValueRow } from '../../../api/snmp';

/** A value row matches the search in its label, name, OID, value or meaning (case does not matter). */
export function rowMatches(row: ValueRow, text: string): boolean {
  const needle = text.trim().toLowerCase();
  return !needle || [row.label, row.name, row.oid, row.value, row.meaning].some(part => (part ?? '').toLowerCase().includes(needle));
}
