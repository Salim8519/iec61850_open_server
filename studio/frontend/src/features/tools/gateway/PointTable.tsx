import { Download, Lock, Save, Search, Sparkles, Trash2, Undo2, Upload, X } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { gatewayApi, problemsOf, type GatewayPoint, type PointKind, type Protocol } from '../../../api/gateway';
import { ADDRESS_LABEL, GATEWAY_COPY, KIND_LABELS, KIND_WHERE } from '../../../copy/gateway';
import { pointKey, mergeValues } from './pointRows';

const PAGE = 100;

/** The SCADA point list: proposed or imported, edited here, saved to the project. */
export function PointTable({ protocol, locked, fail }: { protocol: Protocol; locked: boolean; fail: (message: string) => void }) {
  const [rows, setRows] = useState<GatewayPoint[] | null>(null);
  const [dirty, setDirty] = useState(false);
  const [problems, setProblems] = useState<string[]>([]);
  const [notes, setNotes] = useState<string[]>([]);
  const [filter, setFilter] = useState('');
  const [page, setPage] = useState(0);
  const file = useRef<HTMLInputElement>(null);
  const dirtyRef = useRef(false);
  dirtyRef.current = dirty;
  const load = () => gatewayApi.points(protocol).then(value => {
    setRows(current => dirtyRef.current && current ? mergeValues(current, value.points) : value.points);
  }).catch(error => fail(error.message));
  useEffect(() => { setRows(null); setDirty(false); setProblems([]); setNotes([]); load(); }, [protocol]);
  useEffect(() => { const timer = setInterval(load, 1000); return () => clearInterval(timer); }, [protocol]);
  const edit = (next: GatewayPoint[]) => { setRows(next); setDirty(true); setProblems([]); };
  const update = (index: number, change: Partial<GatewayPoint>) => rows && edit(rows.map((row, at) => at === index ? { ...row, ...change } : row));
  const save = () => rows && gatewayApi.savePoints(protocol, rows)
    .then(value => { setRows(value.points); setDirty(false); setProblems([]); setNotes([]); })
    .catch(error => { setProblems(problemsOf(error)); fail(error.message); });
  const propose = () => gatewayApi.propose(protocol).then(value => { edit(value.points); setNotes(value.notes); })
    .catch(error => fail(error.message));
  const importFile = (picked: File) => picked.text().then(text => gatewayApi.importCsv(protocol, text))
    .then(value => { edit(value.points); setProblems(value.problems); setNotes([`${value.points.length} points read from ${picked.name}.`]); })
    .catch(error => fail(error.message));
  const needle = filter.trim().toLowerCase();
  const shown = useMemo(() => (rows ?? []).map((row, index) => ({ row, index })).filter(({ row }) => !needle ||
    `${row.ref} ${row.ied_name ?? ''} ${row.address} ${KIND_LABELS[row.kind]} ${row.description}`.toLowerCase().includes(needle)),
  [rows, needle]);
  const pages = Math.max(1, Math.ceil(shown.length / PAGE));
  const visible = shown.slice(Math.min(page, pages - 1) * PAGE, Math.min(page, pages - 1) * PAGE + PAGE);
  return <>
    <div className="guide-section">
      <p className="small muted" style={{ margin: 0 }}>{GATEWAY_COPY.pointsHint}</p>
      <div className="row gateway-toolbar">
        <button onClick={propose} disabled={locked}><Sparkles size={16} />{GATEWAY_COPY.propose}</button>
        <button onClick={() => file.current?.click()} disabled={locked}><Upload size={16} />{GATEWAY_COPY.importCsv}</button>
        <a className="button" href={gatewayApi.csvUrl(protocol)} download><Download size={16} />{GATEWAY_COPY.exportCsv}</a>
        <input ref={file} type="file" accept=".csv,text/csv" hidden aria-label="CSV point list"
          onChange={event => { const picked = event.target.files?.[0]; if (picked) importFile(picked); event.target.value = ''; }} />
        <label className="search-field"><Search size={16} /><input value={filter} placeholder="Filter points"
          aria-label="Filter points" onChange={event => { setFilter(event.target.value); setPage(0); }} /></label>
        <span className="gateway-spacer" />
        {dirty && <><span className="small gateway-warning">{GATEWAY_COPY.unsaved}</span>
          <button className="ghost" onClick={() => { setDirty(false); load(); }}><Undo2 size={16} />{GATEWAY_COPY.discard}</button>
          <button className="primary" onClick={save}><Save size={16} />{GATEWAY_COPY.save}</button></>}
        {locked && <span className="small muted row"><Lock size={14} />{GATEWAY_COPY.lockedPoints}</span>}
      </div>
      {notes.map(note => <p key={note} className="small muted" style={{ margin: 0 }}>{note}</p>)}
      {problems.length > 0 && <ul className="gateway-problems" role="alert">{problems.slice(0, 20).map(problem =>
        <li key={problem}>{problem}</li>)}{problems.length > 20 && <li>…and {problems.length - 20} more</li>}</ul>}
    </div>
    {rows && rows.length === 0 ? <p className="muted watch-empty">{GATEWAY_COPY.emptyPoints}</p> :
      <div className="table-scroll"><table className="gateway-points"><thead><tr><th>Type</th><th>{ADDRESS_LABEL[protocol]}</th>
        <th>IED · reference</th><th>Scale</th><th>Value now</th><th>Description</th><th aria-label="Remove" /></tr></thead>
        <tbody>{visible.map(({ row, index }) => <tr key={pointKey(row, index)}>
          <td><select value={row.kind} disabled={locked} aria-label="Point type"
            onChange={event => update(index, { kind: event.target.value as PointKind })}>
            {(Object.keys(KIND_LABELS) as PointKind[]).map(kind => <option key={kind} value={kind}>{KIND_LABELS[kind]}</option>)}
          </select><div className="small muted">{KIND_WHERE[protocol][row.kind]}</div></td>
          <td><input className="gateway-number" inputMode="numeric" value={row.address} disabled={locked} aria-label="Address"
            onChange={event => update(index, { address: Number(event.target.value.replace(/\D/g, '')) })} /></td>
          <td className="reference-cell" title={row.ref}><strong>{row.ied_name ?? row.ied}</strong>
            <div className="small mono">{row.ref}</div>{row.monitor_only && row.kind.startsWith('command') &&
              <div className="small gateway-warning">Real IED: commands are refused</div>}</td>
          <td><input className="gateway-number" value={row.scale} disabled={locked} aria-label="Scale"
            onChange={event => update(index, { scale: Number(event.target.value) || 0 })} /></td>
          <td className="small">{row.kind.startsWith('command') ? <span className="muted">command</span>
            : <span className={row.online ? '' : 'muted'}>{row.value_text ?? '—'}</span>}</td>
          <td><input value={row.description} disabled={locked} aria-label="Description" placeholder="optional"
            onChange={event => update(index, { description: event.target.value })} /></td>
          <td><button className="ghost icon-button" disabled={locked} aria-label={`Remove ${row.ref}`} title="Remove"
            onClick={() => rows && edit(rows.filter((_, at) => at !== index))}><X size={15} /></button></td>
        </tr>)}</tbody></table></div>}
    {pages > 1 && <div className="pager"><span>{shown.length} points · page {Math.min(page, pages - 1) + 1} of {pages}</span>
      <span className="row"><button disabled={page === 0} onClick={() => setPage(page - 1)}>Previous</button>
        <button disabled={page >= pages - 1} onClick={() => setPage(page + 1)}>Next</button></span></div>}
    {rows && rows.length > 0 && !locked && <div className="guide-section"><button className="ghost" onClick={() => edit([])}>
      <Trash2 size={16} />Remove all points</button></div>}
  </>;
}
