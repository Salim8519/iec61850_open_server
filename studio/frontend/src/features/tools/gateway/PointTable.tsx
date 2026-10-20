import { Download, Lock, Plus, Save, Sparkles, Trash2, Undo2, Upload, X } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { gatewayApi, gatewayPointsPath, problemsOf, type GatewayPoint, type PointKind, type Protocol } from '../../../api/gateway';
import { liveFeed } from '../../../api/polls';
import { ADDRESS_LABEL, GATEWAY_COPY, KIND_LABELS, KIND_WHERE, OWN_COPY } from '../../../copy/gateway';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { choicesOf, effective } from '../../../ui/filterState';
import { HelpTip } from '../../../ui/HelpTip';
import { AddOwnPoint } from './AddOwnPoint';
import { CommandButtons, CommandWarning, type PendingCommand } from './CommandSend';
import { blockedReason } from './gatewayCommands';
import { OwnReference, OwnValue } from './OwnCells';
import { isOwn, liveEdit, ownName } from './ownPoints';
import { GATEWAY_POINT_FILTER, gatewayPointMatches } from './pointFilters';
import { pointKey, mergeValues } from './pointRows';
import { Pager } from './Pager';

const PAGE = 100;

/** The SCADA point list: proposed, imported or made up (own points), edited here, saved to the project. */
export function PointTable({ protocol, locked, fail }: { protocol: Protocol; locked: boolean; fail: (message: string) => void }) {
  const [rows, setRows] = useState<GatewayPoint[] | null>(null);
  const [dirty, setDirty] = useState(false);
  const [problems, setProblems] = useState<string[]>([]);
  const [notes, setNotes] = useState<string[]>([]);
  const [page, setPage] = useState(0);
  const [adding, setAdding] = useState(false);
  const [pending, setPending] = useState<PendingCommand | null>(null);
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState<{ ok: boolean; text: string } | null>(null);
  const filter = useListFilter('scada-points', GATEWAY_POINT_FILTER);
  const file = useRef<HTMLInputElement>(null);
  const dirtyRef = useRef(false);
  dirtyRef.current = dirty;
  const path = gatewayPointsPath(protocol);
  const load = () => liveFeed.refresh(path);
  // Shared live feed every second: an unchanged list re-renders nothing (speed lane F).
  useEffect(() => {
    setRows(null); setDirty(false); setProblems([]); setNotes([]); setAdding(false);
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
    .then(value => { edit(value.points); setProblems(value.problems); setNotes([`${value.points.length} points read from ${picked.name}.`]); })
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
  return <>
    <div className="panel-toolbar">
      <button onClick={propose} disabled={locked}><Sparkles size={16} />{GATEWAY_COPY.propose}</button>
      <button onClick={() => setAdding(true)} disabled={locked || adding} title={OWN_COPY.addHint}><Plus size={16} />{OWN_COPY.add}</button>
      <button onClick={() => file.current?.click()} disabled={locked}><Upload size={16} />{GATEWAY_COPY.importCsv}</button>
      <a className="button" href={gatewayApi.csvUrl(protocol)} download><Download size={16} />{GATEWAY_COPY.exportCsv}</a>
      <input ref={file} type="file" accept=".csv,text/csv" hidden aria-label="CSV point list"
        onChange={event => { const picked = event.target.files?.[0]; if (picked) importFile(picked); event.target.value = ''; }} />
      <HelpTip text={GATEWAY_COPY.pointsHint} label="What is the point list?" />
      <span className="gateway-spacer" />
      {dirty && <><span className="small gateway-warning">{GATEWAY_COPY.unsaved}</span>
        <button className="ghost" onClick={() => { setDirty(false); load(); }}><Undo2 size={16} />{GATEWAY_COPY.discard}</button>
        <button className="primary" onClick={save}><Save size={16} />{GATEWAY_COPY.save}</button></>}
      {locked && <span className="small muted row"><Lock size={14} />{GATEWAY_COPY.lockedPoints}</span>}
      {!locked && !!rows?.length && <button className="ghost" onClick={() => edit([])}><Trash2 size={16} />{GATEWAY_COPY.removeAll}</button>}
    </div>
    {adding && rows && <AddOwnPoint protocol={protocol} rows={rows} fail={fail} close={() => setAdding(false)}
      add={point => { edit([...rows, point]); setAdding(false); setNotes([OWN_COPY.saveFirst]); }} />}
    <FilterBar label="Filter points" spec={GATEWAY_POINT_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={rows?.length ?? 0} options={options} />
    {(pending || sent || notes.length > 0 || problems.length > 0) && <div className="guide-section panel-messages">
      {pending && <CommandWarning pending={pending} real={pending.row.ied_role === 'real'} own={isOwn(pending.row)} busy={sending}
        cancel={() => setPending(null)} send={() => send(pending)} />}
      {sent && <p className={`small gateway-sent${sent.ok ? ' ok' : ''}`} role="status">{sent.text}</p>}
      {notes.map(note => <p key={note} className="small muted" style={{ margin: 0 }}>{note}</p>)}
      {problems.length > 0 && <ul className="gateway-problems" role="alert">{problems.slice(0, 20).map(problem =>
        <li key={problem}>{problem}</li>)}{problems.length > 20 && <li>…and {problems.length - 20} more</li>}</ul>}
    </div>}
    {rows && rows.length === 0 ? <p className="muted watch-empty">{GATEWAY_COPY.emptyPoints}</p> :
      <div className="table-scroll"><table className="gateway-points"><thead><tr><th>Type</th><th>{ADDRESS_LABEL[protocol]}</th>
        <th>IED · reference</th><th>Scale</th><th>Value now</th><th>Description</th><th aria-label="Remove" /></tr></thead>
        <tbody>{shown.slice(first, first + PAGE).map(({ row, index }) => {
          const own = isOwn(row);
          const command = row.kind.startsWith('command');
          return <tr key={pointKey(row, index)}>
            <td><span className="type-cell"><select value={row.kind} disabled={locked} aria-label="Point type"
              onChange={event => update(index, own ? { kind: event.target.value as PointKind, value: undefined, target: undefined }
                : { kind: event.target.value as PointKind })}>
              {(Object.keys(KIND_LABELS) as PointKind[]).map(kind => <option key={kind} value={kind}>{KIND_LABELS[kind]}</option>)}
            </select><span className="small muted">{KIND_WHERE[protocol][row.kind]}</span></span></td>
            <td><input className="gateway-number" inputMode="numeric" value={row.address} disabled={locked} aria-label="Address"
              onChange={event => update(index, { address: Number(event.target.value.replace(/\D/g, '')) })} /></td>
            {own ? <OwnReference row={row} rows={rows ?? []} locked={locked} change={target => update(index, { target })} />
              : <td className="reference-cell" title={row.ref}><strong>{row.ied_name ?? row.ied}</strong>
                <div className="small mono">{row.ref}</div>{row.monitor_only && command &&
                  <div className="small gateway-warning">Real IED: commands are refused</div>}</td>}
            <td><input className="gateway-number" value={row.scale} disabled={locked} aria-label="Scale"
              onChange={event => update(index, { scale: Number(event.target.value) || 0 })} /></td>
            <td className="small value-cell" title={command && !own ? undefined : row.value_text}>{command
              ? <div className="own-command"><CommandButtons row={row} blocked={blockedReason(row, dirty)}
                ask={choice => { setSent(null); setPending(choice); }} />
                {own && <span className="small muted">{row.value_text ?? 'No command yet'}</span>}</div>
              : own ? <OwnValue row={row} set={value => setOwn(index, row, value)} fail={fail} />
                : <span className={row.online ? '' : 'muted'}>{row.online === false && '⚠ '}{row.value_text ?? '—'}</span>}</td>
            <td><input value={row.description} disabled={locked} aria-label="Description" placeholder={own ? OWN_COPY.namePlaceholder : 'optional'}
              onChange={event => update(index, { description: event.target.value })} /></td>
            <td><button className="ghost icon-button" disabled={locked} aria-label={`Remove ${own ? ownName(row) : row.ref}`} title="Remove"
              onClick={() => rows && edit(rows.filter((_, at) => at !== index))}><X size={15} /></button></td>
          </tr>;
        })}</tbody></table>
        {rows && rows.length > 0 && shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
    <Pager count={shown.length} page={page} size={PAGE} setPage={setPage} noun="points" />
  </>;
}
