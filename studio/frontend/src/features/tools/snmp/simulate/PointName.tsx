import { Pencil } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { ValueRow } from '../../../../api/snmp';
import { POINTS_COPY } from '../../../../copy/snmp';

/** The name cell of a value: the user's own label (always editable), the MIB name, and the OID,
 *  which the pencil turns into a box (Enter saves, Escape cancels). */
export function PointName({ row, rename, move }: {
  row: ValueRow; rename: (label: string) => void; move: (text: string) => Promise<boolean>;
}) {
  const saved = row.label ?? '';
  const [label, setLabel] = useState(saved);
  const [editing, setEditing] = useState(false);
  useEffect(() => setLabel(saved), [saved]);
  const commitLabel = () => { if (label.trim() !== saved) rename(label.trim()); };
  const done = (text: string | null) => {
    if (text === null || !text.trim() || text.trim() === row.oid) { setEditing(false); return; }
    move(text).then(moved => { if (moved) setEditing(false); });
  };
  return <div className="snmp-name">
    <input className="snmp-label" value={label} maxLength={120} placeholder={POINTS_COPY.labelPlaceholder}
      aria-label={`${POINTS_COPY.label} of ${row.name}`} title={POINTS_COPY.labelHint}
      onChange={event => setLabel(event.target.value)} onBlur={commitLabel}
      onKeyDown={event => { if (event.key === 'Enter') commitLabel(); if (event.key === 'Escape') setLabel(saved); }} />
    <span>{row.name}</span>
    {editing
      ? <><input autoFocus className="snmp-edit mono" defaultValue={row.oid} spellCheck={false}
        aria-label={`${POINTS_COPY.changeOid} of ${row.name}`}
        onBlur={event => done(event.target.value)}
        onKeyDown={event => {
          if (event.key === 'Enter') done((event.target as HTMLInputElement).value);
          if (event.key === 'Escape') done(null);
        }} /><span className="small muted">{POINTS_COPY.oidHint}</span></>
      : <span className="row snmp-oid"><span className="small muted mono">{row.oid}</span>
        <button type="button" className="ghost icon-button" title={POINTS_COPY.changeOid}
          aria-label={`${POINTS_COPY.changeOid} of ${row.name}`} onClick={() => setEditing(true)}><Pencil size={13} /></button></span>}
  </div>;
}
