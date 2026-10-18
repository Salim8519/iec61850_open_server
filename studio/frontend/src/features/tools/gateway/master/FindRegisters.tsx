import { ListPlus, Radar } from 'lucide-react';
import { useState } from 'react';
import { masterApi, type MasterStatus } from '../../../../api/master';
import { DATA_TYPE_NAMES, RANGES_COPY, TABLE_NAMES } from '../../../../copy/master';
import { probeCost } from './probeCost';

/** Find registers: a bounded, read-only probe with a preview of how many requests it sends. */
export function FindRegisters({ status, added, fail }: { status: MasterStatus; added: () => void; fail: (message: string) => void }) {
  const [table, setTable] = useState('holding_registers');
  const [start, setStart] = useState(0);
  const [end, setEnd] = useState(999);
  const [dataType, setDataType] = useState('uint16');
  const probe = status.probe;
  const span = end - start + 1;
  const valid = span >= 1 && span <= 10000 && start >= 0 && end <= 65535;
  const cost = probeCost(table, start, end);
  const running = probe.state === 'running';
  const bits = table === 'coils' || table === 'discrete_inputs';
  const number = (value: string) => Number(value.replace(/\D/g, '') || 0);
  const addFound = () => masterApi.addFound(dataType).then(added).catch(error => fail(error.message));
  return <div className="master-command" role="group" aria-label={RANGES_COPY.findTitle}>
    <strong>{RANGES_COPY.findTitle}</strong>
    <div className="row master-command-fields">
      <label>{RANGES_COPY.table}<select value={table} disabled={running} onChange={event => setTable(event.target.value)}>
        {Object.keys(TABLE_NAMES).map(name => <option key={name} value={name}>{TABLE_NAMES[name]}</option>)}</select></label>
      <label>{RANGES_COPY.from}<input className="gateway-number" value={start} disabled={running}
        onChange={event => setStart(number(event.target.value))} /></label>
      <label>{RANGES_COPY.to}<input className="gateway-number" value={end} disabled={running}
        onChange={event => setEnd(number(event.target.value))} /></label>
      <button className="primary" disabled={!valid || running || !status.link_open}
        onClick={() => masterApi.probe(table, start, end).catch(error => fail(error.message))}>
        <Radar size={16} />{RANGES_COPY.find}</button>
    </div>
    <p className="small muted" style={{ margin: 0 }}>{RANGES_COPY.findHint} {valid
      ? RANGES_COPY.preview(cost.fewest, cost.most) : 'Choose 1 to 10000 addresses.'}</p>
    {running && <p className="small" role="status" style={{ margin: 0 }}>{RANGES_COPY.running(probe.requests ?? 0, probe.address ?? 0)}</p>}
    {probe.state === 'failed' && <p className="small gateway-warning" style={{ margin: 0 }}>{probe.message}</p>}
    {probe.state === 'done' && <div className="row master-command-fields">
      <span className="small">{RANGES_COPY.found(probe.found?.length ?? 0)} {(probe.found ?? []).slice(0, 8).map(span =>
        `${span.start}–${span.start + span.count - 1}`).join(', ')}{(probe.found?.length ?? 0) > 8 ? ' …' : ''}</span>
      {(probe.found?.length ?? 0) > 0 && <>
        {!bits && <label>{RANGES_COPY.addFound}<select value={dataType} onChange={event => setDataType(event.target.value)}>
          {['uint16', 'int16', 'uint32', 'int32', 'float32'].map(type => <option key={type} value={type}>{DATA_TYPE_NAMES[type]}</option>)}
        </select></label>}
        <button onClick={addFound}><ListPlus size={16} />{RANGES_COPY.addFoundButton}</button></>}
    </div>}
  </div>;
}
