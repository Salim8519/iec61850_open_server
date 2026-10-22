import { Send, X } from 'lucide-react';
import type { MasterProtocol } from '../../../../../api/master';
import type { ComparedRow, ListPoint, MasterType } from '../../../../../api/masterList';
import { LIST_ADDRESS, LIST_COPY, STATE_WORDS } from '../../../../../copy/masterList';
import { EditableText } from '../../../../../ui/EditableText';
import { KindBadge, kindRow } from '../../../../../ui/KindBadge';
import { checkText, kindOfListType, stateText, typedNumber } from './listRows';

const time = (ms?: number | null) => ms ? new Date(ms).toLocaleTimeString() : '';

/** The listed points with what the engine found for each: the result in words and a symbol (never colour
 *  alone), the value received with its time, and the optional IED check. Names, scale and unit are edited
 *  in place; a command point has a Send button that opens the confirmation with its saved command. */
export function ListTable({ rows, protocol, types, check, linkOpen, change, remove, send }: {
  rows: ComparedRow[]; protocol: MasterProtocol; types: Map<string, MasterType>; check: boolean; linkOpen: boolean;
  change: (id: string, patch: Partial<ListPoint>) => void; remove: (id: string) => void; send: (row: ComparedRow) => void;
}) {
  return <table className="gateway-points master-points master-list compact-rows"><thead><tr>
    <th>{LIST_COPY.columns.state}</th><th>{LIST_COPY.columns.type}</th><th>{LIST_ADDRESS[protocol]}</th>
    <th>{LIST_COPY.columns.name}</th><th>{LIST_COPY.columns.value}</th><th>{LIST_COPY.columns.scale} · {LIST_COPY.columns.unit}</th>
    {check && <th title={LIST_COPY.checkHint}>{LIST_COPY.checkColumn}</th>}<th aria-label="Actions" /></tr></thead>
    <tbody>{rows.map(row => {
      const type = types.get(row.point.type);
      const kind = kindOfListType(type);
      const numeric = !!type?.group && !['single', 'double', 'binary', 'double_binary', 'binary_output', 'coil', 'discrete_input']
        .includes(type.group);
      return <tr key={row.point.id} {...kindRow(kind)} className={`list-${row.state}`}>
        <td title={STATE_WORDS[row.state].hint}><strong className={`list-state ${row.state}`}>{stateText(row)}</strong>
          {row.reason && <span className="small muted list-reason">{row.reason}</span>}</td>
        <td><span className="type-cell"><KindBadge kind={kind} raw={type?.type_id} />
          <span className="small">{type?.name ?? row.point.type}</span></span></td>
        <td className="mono">{row.point.address}</td>
        <td title={row.point.from ? LIST_COPY.copiedFrom(row.point.from.address) : undefined}><EditableText value={row.point.name} label={`${LIST_COPY.columns.name} ${row.point.address}`}
          placeholder={LIST_COPY.namePlaceholder} onSave={text => change(row.point.id, { name: text.trim() })} /></td>
        <td className={row.stale ? 'muted' : ''}>{row.engineering !== undefined && <strong>{row.engineering}</strong>}
          {row.quality_text && <span className="small muted list-reason">{row.good ? row.quality_text : `⚠ ${row.quality_text}`}
            {row.time_ms ? ` · ${time(row.time_ms)}` : ''}</span>}</td>
        <td className="list-scale">{numeric ? <>
          <EditableText mono inputMode="decimal" value={String(row.point.scale)} label={`${LIST_COPY.columns.scale} ${row.point.address}`}
            onSave={text => { const scale = typedNumber(text); if (scale) change(row.point.id, { scale }); }} />
          <EditableText value={row.point.unit} label={`${LIST_COPY.columns.unit} ${row.point.address}`} placeholder="unit"
            onSave={text => change(row.point.id, { unit: text.trim().slice(0, 16) })} /></> : <span className="muted">—</span>}</td>
        {check && <td className="small list-check">{checkText(row)}</td>}
        <td className="list-actions">{row.command && <button className="ghost" disabled={!linkOpen} title={LIST_COPY.sendHint}
          onClick={() => send(row)}><Send size={14} />{LIST_COPY.send}</button>}
          <button className="ghost icon-button" aria-label={`${LIST_COPY.remove} ${row.point.name || row.point.address}`}
            title={LIST_COPY.remove} onClick={() => remove(row.point.id)}><X size={15} /></button></td>
      </tr>;
    })}</tbody></table>;
}
