import { Trash2 } from 'lucide-react';
import type { Behaviour, ValueRow } from '../../../../api/snmp';
import { BEHAVIOURS, POINTS_COPY, TYPE_NAMES } from '../../../../copy/snmp';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { kindOfValue } from '../valueFilter';
import { PointLabel, PointOid } from './PointName';
import { ValueEditor } from './ValueEditor';

export type RowActions = {
  rename: (row: ValueRow, label: string) => void; move: (row: ValueRow, text: string) => void;
  setValue: (row: ValueRow, value: string) => void; change: (row: ValueRow, update: { behaviour?: Behaviour;
    rate?: number; writable?: boolean }) => void; remove: (row: ValueRow) => void;
};

/** One simulated value on one line: label, OID, type badge, value, how it changes, writable, delete. */
export function PointRow({ row, writable, picked, pick, actions }: {
  row: ValueRow; writable: boolean; picked?: boolean; pick?: () => void; actions: RowActions;
}) {
  const kind = kindOfValue(row);
  return <tr {...kindRow(kind)} className={picked ? 'picked' : undefined}>
    {pick && <td><input type="checkbox" checked={Boolean(picked)} aria-label={`Tick ${row.name}`} onChange={pick} /></td>}
    <td><PointLabel row={row} rename={label => actions.rename(row, label)} /></td>
    <td><PointOid row={row} move={text => actions.move(row, text)} /></td>
    <td><KindBadge kind={kind} raw={TYPE_NAMES[row.type] ?? row.type} /></td>
    <td className="snmp-value"><ValueEditor row={row} commit={value => actions.setValue(row, value)} /></td>
    <td><span className="row snmp-behaviour">
      <select aria-label={POINTS_COPY.changesOf(row.name)} value={row.behaviour || 'fixed'}
        onChange={event => actions.change(row, { behaviour: event.target.value as Behaviour })}>
        {(Object.keys(BEHAVIOURS) as Behaviour[]).map(key => <option key={key} value={key}>{BEHAVIOURS[key]}</option>)}
      </select>
      {row.behaviour === 'counter' && <input className="gateway-number" inputMode="decimal" aria-label={`Rate of ${row.name}`}
        defaultValue={row.rate ?? 0} title={POINTS_COPY.rate}
        onBlur={event => actions.change(row, { rate: Number(event.target.value) || 0 })} />}
    </span></td>
    <td><input type="checkbox" checked={writable} aria-label={`${row.name} is writable`}
      onChange={event => actions.change(row, { writable: event.target.checked })} /></td>
    <td><button className="ghost icon-button" aria-label={`${POINTS_COPY.remove} ${row.name}`} title={POINTS_COPY.remove}
      onClick={() => actions.remove(row)}><Trash2 size={14} /></button></td>
  </tr>;
}
