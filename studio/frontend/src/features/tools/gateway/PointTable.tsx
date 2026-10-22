import { Fragment, useEffect, useMemo, useRef, useState } from 'react';
import { gatewayApi, gatewayPointsPath, problemsOf, type GatewayPoint, type PointKind, type Protocol } from '../../../api/gateway';
import { liveFeed } from '../../../api/polls';
import type { Ied } from '../../../api/types';
import { ADDRESS_LABEL, GATEWAY_COPY, OWN_COPY } from '../../../copy/gateway';
import { kindOfGatewayPoint } from '../../../points/pointKind';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { choicesOf, effective } from '../../../ui/filterState';
import { stillThere, toggleShown, togglePick } from '../../../ui/pickRows';
import { SelectBar } from '../../../ui/SelectBar';
import { AddOwnPoint } from './AddOwnPoint';
import { CommandWarning, type PendingCommand } from './CommandSend';
import { kindCounts } from './kindCounts';
import { isOwn, liveEdit, ownName } from './ownPoints';
import { typeChoices } from './pointEdits';
import { GATEWAY_POINT_FILTER, gatewayPointMatches } from './pointFilters';
import { pointKey, mergeValues } from './pointRows';
import { Pager } from './Pager';
import { PointRow } from './PointRow';
import { PointToolbar } from './PointToolbar';
import { SignalChange } from './SignalChange';

const PAGE = 100;

/** The SCADA point list: proposed, imported or made up (own points), edited here, saved to the project. */
export function PointTable({ protocol, ieds, locked, fail }: {
  protocol: Protocol; ieds: Ied[]; locked: boolean; fail: (message: string) => void;
}) {
  const [rows, setRows] = useState<GatewayPoint[] | null>(null);
  const [dirty, setDirty] = useState(false);
  const [problems, setProblems] = useState<string[]>([]);
  const [notes, setNotes] = useState<string[]>([]);
  const [page, setPage] = useState(0);
  const [adding, setAdding] = useState(false);
  const [pending, setPending] = useState<PendingCommand | null>(null);
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState<{ ok: boolean; text: string } | null>(null);
  const [picked, setPicked] = useState<Set<string> | null>(null);
  const [changing, setChanging] = useState<string | null>(null);
  const filter = useListFilter('scada-points', GATEWAY_POINT_FILTER);
  const dirtyRef = useRef(false);
  dirtyRef.current = dirty;
  const path = gatewayPointsPath(protocol);
  const load = () => liveFeed.refresh(path);
  // Shared live feed every second: an unchanged list re-renders nothing (speed lane F).
  useEffect(() => {
    setRows(null); setDirty(false); setProblems([]); setNotes([]); setAdding(false); setPicked(null); setChanging(null);
    return liveFeed.subscribe<{ points: GatewayPoint[] }>(path, 1000, value => {
      setRows(current => dirtyRef.current && current ? mergeValues(current, value.points) : value.points);
    }, error => fail(error.message));
  }, [path]);
  useEffect(() => setPage(0), [filter.values]);
  const edit = (next: GatewayPoint[]) => { setRows(next); setDirty(true); setProblems([]); };
  const update = (index: number, change: Partial<GatewayPoint>) => rows && edit(rows.map((row, at) => at === index ? { ...row, ...change } : row));
  const save = () => rows && gatewayApi.savePoints(protocol, rows)
    .then(value => { setRows(value.points); setDirty(false); setProblems([]); setNotes([]); })
    .catch(error => { setProblems(problemsOf(error)); fail(error.message); });
  // Own points stay when the IED points are proposed again.
  const propose = () => gatewayApi.propose(protocol, (rows ?? []).filter(isOwn))
    .then(value => { edit(value.points); setNotes(value.notes); }).catch(error => fail(error.message));
  const importFile = (picked: File) => picked.text().then(text => gatewayApi.importCsv(protocol, text))
    .then(value => {
      // A file with no usable rows leaves the list as it was; only its problems are shown.
      if (value.points.length) edit(value.points);
      setProblems(value.problems); setNotes([value.points.length ? `${value.points.length} points read from ${picked.name}.`
        : `No points could be read from ${picked.name}; the list is unchanged.`]);
    })
    .catch(error => fail(error.message));
  // A saved own point changes at once (SCADA sees it while the gateway runs); a draft waits for Save.
  const setOwn = (index: number, row: GatewayPoint, value: number) => liveEdit(row, dirty)
    ? gatewayApi.setOwnValue(row.id!, value).then(answer => {
      setRows(current => current && current.map(item => item.id === row.id ? { ...item, ...answer.point } : item)); load();
    }).catch(error => fail(error.message))
    : update(index, { value });
  const send = (command: PendingCommand) => {
    setSending(true); setSent(null);
    const own = isOwn(command.row);
    gatewayApi.send(command.row.id!, command.value)
      .then(answer => setSent({ ok: answer.ok, text: own ? OWN_COPY.sentResult(command.label, ownName(command.row), answer.ok, answer.reason)
        : GATEWAY_COPY.sentResult(command.label, command.row.ref, answer.ok, answer.reason) }))
      .catch(error => fail(error.message)).finally(() => { setSending(false); setPending(null); load(); });
  };
  const options = useMemo(() => ({ ied: choicesOf((rows ?? []).map(row => row.ied_name ?? row.ied)) }), [rows]);
  const values = useMemo(() => effective(GATEWAY_POINT_FILTER, filter.values, options), [filter.values, options]);
  const shown = useMemo(() => (rows ?? []).map((row, index) => ({ row, index }))
    .filter(({ row }) => gatewayPointMatches(row, values)), [rows, values]);
  const first = Math.min(page, Math.max(0, Math.ceil(shown.length / PAGE) - 1)) * PAGE;
  const shownKeys = shown.map(({ row, index }) => pointKey(row, index));
  // Signals other IED points of the same type already serve: the Change row offers them as taken.
  const takenFor = (index: number, kind: PointKind) => new Set((rows ?? [])
    .filter((other, at) => at !== index && !isOwn(other) && other.kind === kind).map(other => `${other.ied}:${other.ref}`));
  const chosen = picked && rows ? stillThere(picked, rows.map(pointKey)) : [];
  const choices = useMemo(() => typeChoices(protocol), [protocol]);
  const counts = useMemo(() => kindCounts(rows ?? [], row => kindOfGatewayPoint(row.kind)), [rows]);
  const removeAll = () => {
    if (!rows) return;
    edit([]);
    setNotes([GATEWAY_COPY.removedNote(rows.length)]);
  };
  const removeChosen = () => {
    if (!rows) return;
    const gone = new Set(chosen);
    edit(rows.filter((row, index) => !gone.has(pointKey(row, index))));
    setNotes([GATEWAY_COPY.removedNote(gone.size)]);
    setPicked(null);
  };
  return <>
    <PointToolbar protocol={protocol} locked={locked} dirty={dirty} adding={adding} hasRows={!!rows?.length} picking={!!picked}
      propose={propose} add={() => setAdding(true)} togglePicking={() => setPicked(picked ? null : new Set())}
      removeAll={removeAll} importFile={importFile} discard={() => { setDirty(false); load(); }} save={save} />
    {adding && rows && <AddOwnPoint protocol={protocol} rows={rows} fail={fail} close={() => setAdding(false)}
      add={point => { edit([...rows, point]); setAdding(false); setNotes([OWN_COPY.saveFirst]); }} />}
    <FilterBar label="Filter points" spec={GATEWAY_POINT_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={rows?.length ?? 0} options={options} counts={counts} />
    {picked && !locked && <SelectBar count={chosen.length} shown={shown.length} noun="point" cancel={() => setPicked(null)}
      selectShown={() => setPicked(toggleShown(new Set(), shownKeys))} remove={removeChosen} />}
    {(pending || sent || notes.length > 0 || problems.length > 0) && <div className="guide-section panel-messages">
      {pending && <CommandWarning pending={pending} real={pending.row.ied_role === 'real'} own={isOwn(pending.row)} busy={sending}
        cancel={() => setPending(null)} send={() => send(pending)} />}
      {sent && <p className={`small gateway-sent${sent.ok ? ' ok' : ''}`} role="status">{sent.text}</p>}
      {notes.map(note => <p key={note} className="small muted" style={{ margin: 0 }}>{note}</p>)}
      {problems.length > 0 && <ul className="gateway-problems" role="alert">{problems.slice(0, 20).map(problem =>
        <li key={problem}>{problem}</li>)}{problems.length > 20 && <li>…and {problems.length - 20} more</li>}</ul>}
    </div>}
    {rows && rows.length === 0 ? <p className="muted watch-empty">{GATEWAY_COPY.emptyPoints}</p> :
      <div className="table-scroll"><table className="gateway-points compact-rows"><thead><tr>
        {picked && !locked && <th className="pick-col"><input type="checkbox" aria-label="Tick all points shown"
          checked={shownKeys.length > 0 && shownKeys.every(key => picked.has(key))}
          onChange={() => setPicked(toggleShown(picked, shownKeys))} /></th>}<th>{GATEWAY_COPY.name}</th><th>Type</th>
        <th>{ADDRESS_LABEL[protocol]}</th><th>IED · path</th><th>Scale</th><th>Value now</th><th aria-label="Remove" /></tr></thead>
        <tbody>{shown.slice(first, first + PAGE).map(({ row, index }) => {
          const key = pointKey(row, index);
          return <Fragment key={key}><PointRow row={row} protocol={protocol} rows={rows ?? []} locked={locked} dirty={dirty}
            picking={!!picked && !locked} picked={!!picked?.has(key)} changing={changing === key} choices={choices}
            togglePicked={() => picked && setPicked(togglePick(picked, key))} toggleChange={() => setChanging(changing === key ? null : key)}
            update={change => update(index, change)} remove={() => rows && edit(rows.filter((_, at) => at !== index))}
            setOwn={value => setOwn(index, row, value)} ask={choice => { setSent(null); setPending(choice); }} fail={fail} />
            {changing === key && !isOwn(row) && !locked && <SignalChange row={row} ieds={ieds} taken={takenFor(index, row.kind)}
              columns={picked ? 8 : 7} close={() => setChanging(null)} choose={(ied, ref) => {
                update(index, { ied: ied.id, ref, ied_name: ied.name, value_text: undefined, online: undefined });
                setChanging(null);
              }} />}</Fragment>;
        })}</tbody></table>
        {rows && rows.length > 0 && shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
    <Pager count={shown.length} page={page} size={PAGE} setPage={setPage} noun="points" />
  </>;
}
