import { Plus } from 'lucide-react';
import type { MasterProtocol } from '../../../../../api/master';
import type { ExtraRow } from '../../../../../api/masterList';
import { LIST_ADDRESS, LIST_COPY } from '../../../../../copy/masterList';
import { kindOfMasterGroup } from '../../../../../points/pointKind';
import { KindBadge, kindRow } from '../../../../../ui/KindBadge';

/** Points the device sent that the list does not have, each with one-click Add (the engine plans the point
 *  from what arrived: its type, address and the name given in Everything received). */
export function NotInList({ rows, protocol, add }: {
  rows: ExtraRow[]; protocol: MasterProtocol; add: (keys: string[]) => void;
}) {
  const addable = rows.filter(row => row.list_type);
  return <>
    <div className="row list-extra-head"><strong>{LIST_COPY.notInList}</strong>
      <span className="small muted">{LIST_COPY.notInListHint}</span><span className="gateway-spacer" />
      <button type="button" disabled={!addable.length} onClick={() => add(addable.map(row => row.key))}>
        <Plus size={16} />{LIST_COPY.addAll(addable.length)}</button></div>
    <table className="gateway-points master-points compact-rows"><thead><tr><th>{LIST_COPY.columns.type}</th>
      <th>{LIST_ADDRESS[protocol]}</th><th>{LIST_COPY.columns.name}</th><th>{LIST_COPY.columns.value}</th><th aria-label="Add" /></tr></thead>
      <tbody>{rows.map(row => {
        const kind = kindOfMasterGroup(row.group);
        return <tr key={row.key} {...kindRow(kind)}>
          <td><span className="type-cell"><KindBadge kind={kind} raw={`${row.group_name} · ${row.type}`} />
            <span className="small">{row.group_name}</span></span></td>
          <td className="mono">{row.address}</td>
          <td>{row.label || <span className="muted">—</span>}</td>
          <td><strong>{row.value_text}</strong>
            <span className="small muted list-reason">{row.good ? row.quality_text : `⚠ ${row.quality_text}`}</span></td>
          <td>{row.list_type ? <button type="button" className="ghost" onClick={() => add([row.key])}>
            <Plus size={14} />{LIST_COPY.addOne}</button> : <span className="small muted">{LIST_COPY.noType}</span>}</td>
        </tr>;
      })}</tbody></table>
  </>;
}
