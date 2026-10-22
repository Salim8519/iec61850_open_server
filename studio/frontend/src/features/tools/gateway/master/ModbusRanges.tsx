import { Plus, Radar, Save, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { masterApi, type MasterStatus, type ModbusRange } from '../../../../api/master';
import { DATA_TYPE_NAMES, RANGES_COPY, TABLE_NAMES } from '../../../../copy/master';
import { kindOfMasterGroup } from '../../../../points/pointKind';
import { EditableText } from '../../../../ui/EditableText';
import { HelpTip } from '../../../../ui/HelpTip';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { ChoiceEdit } from '../ChoiceEdit';
import { FindRegisters } from './FindRegisters';
import { AutoRanges } from './list/AutoRanges';

const BITS = new Set(['coils', 'discrete_inputs']);
const TABLE_CHOICES = Object.entries(TABLE_NAMES).map(([value, label]) => ({ value, label }));
/** A Modbus table holds one kind of point: coils and discrete inputs are on/off, registers are values. */
const tableKind = (table: string) => kindOfMasterGroup(table.replace(/s$/, ''));
const REGISTER_TYPES = ['uint16', 'int16', 'uint32', 'int32', 'float32'];

/** Modbus has no discovery: the ranges ScadaBench reads every poll interval, and Find registers. */
export function ModbusRanges({ status, fail }: { status: MasterStatus; fail: (message: string) => void }) {
  const [ranges, setRanges] = useState<ModbusRange[] | null>(null);
  const [dirty, setDirty] = useState(false);
  const [showFind, setShowFind] = useState<boolean | null>(null);
  const load = () => masterApi.config().then(config => { setRanges(config.modbus_ranges); setDirty(false); })
    .catch(error => fail(error.message));
  useEffect(() => { load(); }, []);
  const edit = (index: number, change: Partial<ModbusRange>) => {
    if (!ranges) return;
    setRanges(ranges.map((range, at) => {
      if (at !== index) return range;
      const next = { ...range, ...change };
      if (change.table) next.data_type = BITS.has(change.table) ? 'bool' : range.data_type === 'bool' ? 'uint16' : range.data_type;
      return next;
    }));
    setDirty(true);
  };
  const add = () => { setRanges([...(ranges ?? []), { table: 'holding_registers', start: 0, count: 10, data_type: 'uint16',
    word_order: 'big' }]); setDirty(true); };
  const save = () => ranges && masterApi.saveRanges(ranges).then(answer => { setRanges(answer.modbus_ranges); setDirty(false); })
    .catch(error => fail(error.message));
  const failed = new Map(status.poll_failures.map(item => [item.range, item.text]));
  const number = (value: string) => Number(value.replace(/\D/g, '') || 0);
  return <>
    <div className="panel-toolbar">
      <button onClick={add} disabled={(ranges?.length ?? 0) >= 20}><Plus size={16} />{RANGES_COPY.add}</button>
      <button onClick={() => setShowFind(!(showFind ?? ranges?.length === 0))}><Radar size={16} />{RANGES_COPY.find}…</button>
      <HelpTip text={RANGES_COPY.hint} label="What is this?" />
      <span className="gateway-spacer" />
      {dirty && <button className="primary" onClick={save}><Save size={16} />{RANGES_COPY.save}</button>}
    </div>
    {(showFind ?? (ranges?.length === 0 || status.probe.state === 'running')) && <div className="guide-section panel-messages find-registers">
      <FindRegisters status={status} added={() => { load(); setShowFind(false); }} fail={fail} /></div>}
    {ranges && ranges.length === 0 ? <p className="muted watch-empty">{RANGES_COPY.empty}</p> :
      <div className="table-scroll"><table className="gateway-points compact-rows"><thead><tr><th>{RANGES_COPY.table}</th>
        <th>{RANGES_COPY.start}</th><th>{RANGES_COPY.count}</th><th>{RANGES_COPY.type}</th><th>{RANGES_COPY.order}</th>
        <th>Answer</th><th aria-label="Remove" /></tr></thead>
        <tbody>{(ranges ?? []).map((range, index) => <tr key={index} {...kindRow(tableKind(range.table))}>
          <td><ChoiceEdit label={RANGES_COPY.table} value={range.table} choices={TABLE_CHOICES}
            shown={<span className="type-cell"><KindBadge kind={tableKind(range.table)} raw={TABLE_NAMES[range.table]} />
              <span className="small">{TABLE_NAMES[range.table] ?? range.table}</span></span>}
            onChange={table => edit(index, { table })} /></td>
          <td className="address-cell"><EditableText mono inputMode="numeric" value={String(range.start)} label={RANGES_COPY.start}
            onSave={text => edit(index, { start: number(text) })} /></td>
          <td className="address-cell"><EditableText mono inputMode="numeric" value={String(range.count)} label={RANGES_COPY.count}
            onSave={text => edit(index, { count: number(text) })} /></td>
          <td>{BITS.has(range.table) ? DATA_TYPE_NAMES.bool : <select value={range.data_type} aria-label={RANGES_COPY.type}
            onChange={event => edit(index, { data_type: event.target.value })}>
            {REGISTER_TYPES.map(type => <option key={type} value={type}>{DATA_TYPE_NAMES[type]}</option>)}</select>}</td>
          <td>{['uint32', 'int32', 'float32'].includes(range.data_type) ? <select value={range.word_order}
            aria-label={RANGES_COPY.order} onChange={event => edit(index, { word_order: event.target.value })}>
            <option value="big">{RANGES_COPY.big}</option><option value="little">{RANGES_COPY.little}</option></select> : '—'}</td>
          <td className="small">{dirty ? '—' : failed.has(index + 1)
            ? <span className="gateway-warning">⚠ {failed.get(index + 1)}</span> : <span className="quality-good">✓ read</span>}</td>
          <td><button className="ghost icon-button" aria-label="Remove range" title="Remove"
            onClick={() => { setRanges(ranges!.filter((_, at) => at !== index)); setDirty(true); }}><X size={15} /></button></td>
        </tr>)}</tbody></table></div>}
    <AutoRanges refresh={status.ranges} />
  </>;
}
