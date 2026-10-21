import type { ValueRow } from '../../../../api/snmp';
import { BEHAVIOURS, POINTS_COPY } from '../../../../copy/snmp';
import { AutoValue, EditableText } from '../../../../ui/EditableText';

/** The value of a simulated point. A fixed value shows the ✎ pin: click, type, Enter saves, Esc cancels.
 *  A value that changes by itself (uptime, a growing counter) shows the "auto" tag instead: pick Fixed
 *  under Changes to type one. What a MIB says the value means ("1 · up") follows it. */
export function ValueEditor({ row, commit }: { row: ValueRow; commit: (value: string) => void }) {
  const behaviour = row.behaviour || 'fixed';
  if (behaviour !== 'fixed') {
    return <AutoValue value={row.meaning ? `${row.value} · ${row.meaning}` : row.value} how={BEHAVIOURS[behaviour].toLowerCase()} />;
  }
  return <span className="snmp-value-edit">
    <EditableText value={row.value} onSave={commit} label={POINTS_COPY.valueOf(row.name)} placeholder={'""'} mono={row.hex} />
    {row.meaning && <span className="small muted">· {row.meaning}</span>}
  </span>;
}

/** The value as text, with what it means after it when a MIB says so (read-only lists and traps). */
export function ValueText({ row }: { row: ValueRow }) {
  const shown = row.value === '' ? '""' : row.value;
  return <span className="snmp-value-text"><strong className={row.hex ? 'mono' : ''}>{shown}</strong>
    {row.meaning && <span className="small muted"> · {row.meaning}</span>}</span>;
}
