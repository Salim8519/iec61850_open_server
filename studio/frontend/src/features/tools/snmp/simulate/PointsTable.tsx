import { useMemo, useState } from 'react';
import { snmpApi, snmpPointsPath, type SnmpDevice, type SnmpPoint, type ValueRow } from '../../../../api/snmp';
import { useLiveFeed } from '../../../../api/useLiveFeed';
import { POINTS_COPY } from '../../../../copy/snmp';
import { useListFilter } from '../../../../state/useListFilter';
import { FilterBar } from '../../../../ui/FilterBar';
import { HelpTip } from '../../../../ui/HelpTip';
import { stillThere, toggleShown, togglePick } from '../../../../ui/pickRows';
import { SelectBar } from '../../../../ui/SelectBar';
import { Pager } from '../../gateway/Pager';
import { kindCounts, simShown, SIM_FILTER } from '../valueFilter';
import { AddPoint } from './AddPoint';
import { oidOf } from './oidText';
import { PointRow, type RowActions } from './PointRow';
import { PointsActions } from './PointsActions';

const PAGE = 100;

/** The device's values: live while it runs, editable at any time; settings of each value saved with the project.
 *  One line per value; the buttons, search, type chips and count share the bar above the table. */
export function PointsTable({ device, running, refresh, fail }: {
  device: SnmpDevice; running: boolean; refresh: () => void; fail: (message: string) => void;
}) {
  const [again, setAgain] = useState(0);
  const [adding, setAdding] = useState(false);
  const [page, setPage] = useState(0);
  const [picked, setPicked] = useState<Set<string> | null>(null);
  const feed = useLiveFeed<{ points: ValueRow[]; live: boolean }>(snmpPointsPath(device.id), running ? 1000 : null, again);
  const filter = useListFilter('snmp-points', SIM_FILTER);
  const rows = useMemo(() => feed.value?.points ?? [], [feed.value]);
  const counts = useMemo(() => kindCounts(rows), [rows]);
  const shown = useMemo(() => rows.filter(row => simShown(row, filter.values)), [rows, filter.values]);
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
  const actions: RowActions = {
    rename: (row, label) => change(row.oid, { label: label || undefined }),
    // A new OID keeps the value's settings; a name a MIB knows works too.
    move: (row, text) => { oidOf(text, device.points).then(oid => change(row.oid, { oid })).catch(reason => fail(reason.message)); },
    setValue: (row, value) => snmpApi.setValue(device.id, row.oid, value, row.hex).then(reread).catch(reason => fail(reason.message)),
    change: (row, update) => change(row.oid, update),
    remove: row => savePoints(device.points.filter(point => point.oid !== row.oid)),
  };
  const live = feed.value?.live;
  return <>
    <div className="snmp-bar">
      <PointsActions device={device.id} adding={adding} toggleAdding={() => setAdding(!adding)} picking={Boolean(picked)}
        togglePicking={() => setPicked(picked ? null : new Set())} />
      <FilterBar label="Filter values" spec={SIM_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
        shown={shown.length} total={rows.length} counts={counts}
        extra={<span className="small muted snmp-live-note">{live ? POINTS_COPY.live : POINTS_COPY.stored}
          {!live && <HelpTip text={POINTS_COPY.storedHint} label="Why are these values stored?" />}</span>} />
    </div>
    {adding && <AddPoint points={device.points} add={point => savePoints([...device.points, point]).then(() => setAdding(false))}
      close={() => setAdding(false)} fail={fail} />}
    {picked && <SelectBar count={chosen.length} shown={shown.length} noun="value" cancel={() => setPicked(null)}
      selectShown={() => setPicked(toggleShown(new Set(), shownKeys))} remove={removeChosen} />}
    <div className="table-scroll"><table className={`gateway-points snmp-values compact-rows${picked ? ' picking' : ''}`}><thead><tr>
      {picked && <th className="pick-col"><input type="checkbox" aria-label="Tick all values shown"
        checked={shownKeys.length > 0 && shownKeys.every(key => picked.has(key))}
        onChange={() => setPicked(toggleShown(picked, shownKeys))} /></th>}
      <th>{POINTS_COPY.name}</th><th className="col-oid">{POINTS_COPY.oid}</th><th className="col-type">{POINTS_COPY.type}</th>
      <th>{POINTS_COPY.value}</th><th className="col-changes">{POINTS_COPY.behaviour}</th>
      <th className="col-writable">{POINTS_COPY.writable}</th><th className="col-remove" aria-label={POINTS_COPY.remove} /></tr></thead>
      <tbody>{shown.slice(first, first + PAGE).map(row => <PointRow key={row.oid} row={row} actions={actions}
        writable={Boolean(device.points.find(point => point.oid === row.oid)?.writable)}
        picked={picked?.has(row.oid)} pick={picked ? () => setPicked(togglePick(picked, row.oid)) : undefined} />)}</tbody></table>
      {rows.length > 0 && shown.length === 0 && <p className="muted watch-empty">{POINTS_COPY.none}</p>}</div>
    <Pager count={shown.length} page={page} size={PAGE} setPage={setPage} noun="values" />
  </>;
}
