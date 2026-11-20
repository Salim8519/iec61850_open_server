import { Send } from 'lucide-react';
import { memo } from 'react';
import type { MasterPoint } from '../../../../api/master';
import { MASTER_COPY } from '../../../../copy/master';
import { kindOfMasterGroup } from '../../../../points/pointKind';
import { EditableText } from '../../../../ui/EditableText';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { sameData } from '../../../../ui/sameProps';

/** One point the device reported (moved out of MasterPointTable, change 0224). Memoised: when the
 *  table's list changes, only the rows whose point object changed render again (the push stream's
 *  unchanged rows keep their objects, api/rowParse.ts). The handlers act on the latest list. */
export const MasterPointRow = memo(function MasterPointRow({ row, linkOpen, toggle, rename, command }: {
  row: MasterPoint; linkOpen: boolean; toggle: (row: MasterPoint) => void; rename: (row: MasterPoint, label: string) => void;
  command: (row: MasterPoint) => void;
}) {
  const kind = kindOfMasterGroup(row.group);
  return <tr {...kindRow(kind)} className={row.ticked ? '' : 'unwatched'}>
    <td><input type="checkbox" checked={row.ticked} aria-label={`Watch ${row.label || row.key}`}
      onChange={() => toggle(row)} /></td>
    <td><span className="type-cell"><KindBadge kind={kind} raw={`${row.group_name} · ${row.type}`} />
      <span className="small">{row.group_name}</span></span></td>
    <td className="mono">{row.address}</td>
    <td><EditableText value={row.label} label={`Name of ${row.key}`} placeholder={MASTER_COPY.labelPlaceholder}
      onSave={text => rename(row, text.trim())} /></td>
    <td className={row.stale ? 'muted' : ''}><strong>{row.value_text}</strong>
      {row.changes > 0 && <span className="small muted"> · {row.changes} {row.changes === 1 ? 'change' : 'changes'}</span>}</td>
    <td><span className={row.good ? 'quality-good' : 'quality-bad'}>{row.good ? '✓ ' : '⚠ '}{row.quality_text}</span></td>
    <td className="small muted">{row.cause}</td>
    <td>{row.command && <button className="ghost" disabled={!linkOpen} onClick={() => command(row)}>
      <Send size={14} />{MASTER_COPY.sendTo}</button>}</td>
  </tr>;
}, sameData);
