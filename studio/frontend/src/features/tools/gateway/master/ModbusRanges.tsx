import { Plus, Radar, Save, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { masterApi, type MasterStatus, type ModbusRange } from '../../../../api/master';
import { DATA_TYPE_NAMES, RANGES_COPY, TABLE_NAMES } from '../../../../copy/master';
import { FindRegisters } from './FindRegisters';

const BITS = new Set(['coils', 'discrete_inputs']);
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
    <div className="guide-section">
      <p className="small muted" style={{ margin: 0 }}>{RANGES_COPY.hint}</p>
      {(showFind ?? (ranges?.length === 0 || status.probe.state === 'running')) &&
        <FindRegisters status={status} added={() => { load(); setShowFind(false); }} fail={fail} />}
      <div className="row gateway-toolbar">
        <button onClick={add} disabled={(ranges?.length ?? 0) >= 20}><Plus size={16} />{RANGES_COPY.add}</button>
        <button onClick={() => setShowFind(!(showFind ?? ranges?.length === 0))}><Radar size={16} />{RANGES_COPY.find}…</button>
        <span className="gateway-spacer" />
        {dirty && <button className="primary" onClick={save}><Save size={16} />{RANGES_COPY.save}</button>}
      </div>
    </div>
    {ranges && ranges.length === 0 ? <p className="muted watch-empty">{RANGES_COPY.empty}</p> :
      <div className="table-scroll"><table className="gateway-points"><thead><tr><th>{RANGES_COPY.table}</th>
        <th>{RANGES_COPY.start}</th><th>{RANGES_COPY.count}</th><th>{RANGES_COPY.type}</th><th>{RANGES_COPY.order}</th>
        <th>Answer</th><th aria-label="Remove" /></tr></thead>
        <tbody>{(ranges ?? []).map((range, index) => <tr key={index}>
          <td><select value={range.table} aria-label={RANGES_COPY.table} onChange={event => edit(index, { table: event.target.value })}>
            {Object.keys(TABLE_NAMES).map(table => <option key={table} value={table}>{TABLE_NAMES[table]}</option>)}</select></td>
          <td><input className="gateway-number" value={range.start} aria-label={RANGES_COPY.start}
            onChange={event => edit(index, { start: number(event.target.value) })} /></td>
          <td><input className="gateway-number" value={range.count} aria-label={RANGES_COPY.count}
            onChange={event => edit(index, { count: number(event.target.value) })} /></td>
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
  </>;
}
