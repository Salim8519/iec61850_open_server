import { useState } from 'react';
import type { ValueRow } from '../../../../api/snmp';

/** A value with its meaning ("1 · up"); click to type a new one, Enter saves, Escape cancels. */
export function ValueEditor({ row, commit }: { row: ValueRow; commit: (value: string) => void }) {
  const [editing, setEditing] = useState(false);
  if (editing) {
    const done = (value: string | null) => {
      setEditing(false);
      if (value !== null && value !== row.value) commit(value);
    };
    return <input autoFocus defaultValue={row.value} aria-label={`New value of ${row.name}`} className="snmp-edit"
      onBlur={event => done(event.target.value)}
      onKeyDown={event => {
        if (event.key === 'Enter') done((event.target as HTMLInputElement).value);
        if (event.key === 'Escape') done(null);
      }} />;
  }
  return <button type="button" className="snmp-value-button" title="Click to change" onClick={() => setEditing(true)}>
    <ValueText row={row} /></button>;
}

/** The value as text, with what it means after it when a MIB says so. */
export function ValueText({ row }: { row: ValueRow }) {
  const shown = row.value === '' ? '""' : row.value;
  return <span className="snmp-value-text"><strong className={row.hex ? 'mono' : ''}>{shown}</strong>
    {row.meaning && <span className="small muted"> · {row.meaning}</span>}</span>;
}
