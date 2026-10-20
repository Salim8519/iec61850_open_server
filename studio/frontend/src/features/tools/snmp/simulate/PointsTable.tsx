import { Download, ListChecks, Plus, Trash2 } from 'lucide-react';
import { useMemo, useState } from 'react';
import { snmpApi, snmpPointsPath, type Behaviour, type SnmpDevice, type SnmpPoint, type ValueRow } from '../../../../api/snmp';
import { useLiveFeed } from '../../../../api/useLiveFeed';
import { BEHAVIOURS, POINTS_COPY, TYPE_NAMES } from '../../../../copy/snmp';
import { useListFilter } from '../../../../state/useListFilter';
import { FilterBar } from '../../../../ui/FilterBar';
import type { FilterSpec } from '../../../../ui/filterState';
import { HelpTip } from '../../../../ui/HelpTip';
import { stillThere, toggleShown, togglePick } from '../../../../ui/pickRows';
import { SelectBar } from '../../../../ui/SelectBar';
import { Pager } from '../../gateway/Pager';
import { rowMatches } from '../rowMatch';
import { AddPoint } from './AddPoint';
import { ValueEditor } from './ValueEditor';

const PAGE = 100;
const FILTER: FilterSpec = { search: POINTS_COPY.search, toggles: [{ key: 'writable', label: POINTS_COPY.writable }] };

/** The device's values: live while it runs, editable at any time; settings of each value saved with the project. */
export function PointsTable({ device, running, refresh, fail }: {
  device: SnmpDevice; running: boolean; refresh: () => void; fail: (message: string) => void;
}) {
  const [again, setAgain] = useState(0);
  const [adding, setAdding] = useState(false);
  const [page, setPage] = useState(0);
  const [picked, setPicked] = useState<Set<string> | null>(null);
  const feed = useLiveFeed<{ points: ValueRow[]; live: boolean }>(snmpPointsPath(device.id), running ? 1000 : null, again);
  const filter = useListFilter('snmp-points', FILTER);
  const rows = useMemo(() => feed.value?.points ?? [], [feed.value]);
  const shown = useMemo(() => rows.filter(row => rowMatches(row, String(filter.values.text ?? ''))
    && (!filter.values.writable || row.writable)), [rows, filter.values]);
  const first = Math.min(page, Math.max(0, Math.ceil(shown.length / PAGE) - 1)) * PAGE;
  const reread = () => setAgain(value => value + 1);
  const savePoints = (points: SnmpPoint[]) => snmpApi.savePoints(device.id, points)
    .then(() => { refresh(); reread(); }).catch(reason => fail(reason.message));
  const change = (oid: string, update: Partial<SnmpPoint>) =>
    savePoints(device.points.map(point => point.oid === oid ? { ...point, ...update } : point));
  const shownKeys = shown.map(row => row.oid);
  const chosen = picked ? stillThere(picked, rows.map(row => row.oid)) : [];
  const removeChosen = () => {
    if (chosen.length > 1 && !window.confirm(POINTS_COPY.removeMany(chosen.length))) return;
    const gone = new Set(chosen);
    savePoints(device.points.filter(point => !gone.has(point.oid))).then(() => setPicked(null));
  };
  const setValue = (row: ValueRow, value: string) => snmpApi.setValue(device.id, row.oid, value, row.hex)
    .then(reread).catch(reason => fail(reason.message));
  return <>
    <div className="panel-toolbar">
      <button onClick={() => setAdding(!adding)}><Plus size={16} />{POINTS_COPY.add}</button>
      <a className="button" href={snmpApi.csvUrl(device.id)} download><Download size={16} />{POINTS_COPY.exportCsv}</a>
      <button className={picked ? 'active-toggle' : ''} aria-pressed={!!picked} title={POINTS_COPY.selectHint}
        onClick={() => setPicked(picked ? null : new Set())}><ListChecks size={16} />{POINTS_COPY.select}</button>
      <HelpTip text={`${POINTS_COPY.editHint} ${POINTS_COPY.writableHint}`} label="How do I change values?" />
      <span className="gateway-spacer" />
      <span className="small muted">{feed.value?.live ? POINTS_COPY.live : POINTS_COPY.stored}</span>
    </div>
    {adding && <AddPoint points={device.points} add={point => savePoints([...device.points, point]).then(() => setAdding(false))}
      close={() => setAdding(false)} fail={fail} />}
    <FilterBar label="Filter values" spec={FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={rows.length} />
    {picked && <SelectBar count={chosen.length} shown={shown.length} noun="value" cancel={() => setPicked(null)}
      selectShown={() => setPicked(toggleShown(new Set(), shownKeys))} remove={removeChosen} />}
    <div className="table-scroll"><table className={`gateway-points snmp-values${picked ? ' picking' : ''}`}><thead><tr>
      {picked && <th className="pick-col"><input type="checkbox" aria-label="Tick all values shown"
        checked={shownKeys.length > 0 && shownKeys.every(key => picked.has(key))}
        onChange={() => setPicked(toggleShown(picked, shownKeys))} /></th>}
      <th>{POINTS_COPY.name}</th><th>{POINTS_COPY.value}</th><th className="col-type">{POINTS_COPY.type}</th>
      <th className="col-changes">{POINTS_COPY.behaviour}</th><th className="col-writable">{POINTS_COPY.writable}</th>
      <th className="col-remove" aria-label={POINTS_COPY.remove} /></tr></thead>
      <tbody>{shown.slice(first, first + PAGE).map(row => <tr key={row.oid} className={picked?.has(row.oid) ? 'picked' : undefined}>
        {picked && <td><input type="checkbox" checked={picked.has(row.oid)} aria-label={`Tick ${row.name}`}
          onChange={() => setPicked(togglePick(picked, row.oid))} /></td>}
        <td><div className="snmp-name"><span>{row.name}</span><span className="small muted mono">{row.oid}</span></div></td>
        <td className="snmp-value"><ValueEditor row={row} commit={value => setValue(row, value)} /></td>
        <td className="small">{TYPE_NAMES[row.type] ?? row.type}</td>
        <td><span className="row snmp-behaviour">
          <select aria-label={`How ${row.name} changes`} value={row.behaviour || 'fixed'}
            onChange={event => change(row.oid, { behaviour: event.target.value as Behaviour })}>
            {(Object.keys(BEHAVIOURS) as Behaviour[]).map(key => <option key={key} value={key}>{BEHAVIOURS[key]}</option>)}
          </select>
          {row.behaviour === 'counter' && <input className="gateway-number" inputMode="decimal" aria-label={`Rate of ${row.name}`}
            defaultValue={row.rate ?? 0} title={POINTS_COPY.rate}
            onBlur={event => change(row.oid, { rate: Number(event.target.value) || 0 })} />}
        </span></td>
        <td><input type="checkbox" checked={Boolean(device.points.find(point => point.oid === row.oid)?.writable)}
          aria-label={`${row.name} is writable`} onChange={event => change(row.oid, { writable: event.target.checked })} /></td>
        <td><button className="ghost icon" aria-label={`${POINTS_COPY.remove} ${row.name}`}
          onClick={() => savePoints(device.points.filter(point => point.oid !== row.oid))}><Trash2 size={14} /></button></td>
      </tr>)}</tbody></table>
      {rows.length > 0 && shown.length === 0 && <p className="muted watch-empty">{POINTS_COPY.none}</p>}</div>
    <Pager count={shown.length} page={page} size={PAGE} setPage={setPage} noun="values" />
  </>;
}
