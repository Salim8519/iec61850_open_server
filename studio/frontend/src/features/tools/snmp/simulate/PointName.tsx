import type { ValueRow } from '../../../../api/snmp';
import { POINTS_COPY } from '../../../../copy/snmp';
import { EditableText } from '../../../../ui/EditableText';

/** A label is at most 120 characters (the Add value form stops typing there too). */
const LABEL_MAX = 120;

/** The name of a value: the user's own label when there is one, else the MIB name. Both can be changed in
 *  place; clearing the text brings the MIB name back. The label is kept with the project; managers never see it. */
export function PointLabel({ row, rename }: { row: ValueRow; rename: (label: string) => void }) {
  const save = (text: string) => {
    const next = text.trim().slice(0, LABEL_MAX);
    rename(next === row.name ? '' : next);
  };
  return <EditableText value={row.label || row.name} placeholder={POINTS_COPY.labelPlaceholder}
    label={POINTS_COPY.labelOf(row.name)} onSave={save} />;
}

/** The OID of a value, shown in mono. A new OID keeps the value's settings; a name a MIB knows works too. */
export function PointOid({ row, move }: { row: ValueRow; move: (text: string) => void }) {
  return <EditableText mono value={row.oid} label={POINTS_COPY.oidOf(row.name)}
    onSave={text => { if (text.trim()) move(text); }} />;
}
