import { Download, Search, Send, Upload } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { masterApi, type MasterCommand, type MasterPoint, type MasterStatus } from '../../../../api/master';
import { COMMANDS, MASTER_COPY } from '../../../../copy/master';
import { MasterCommandForm, startValue } from './MasterCommandForm';

const PAGE = 100;

/** Every point the device reported: watched (ticked) by default, with names the user gives. */
export function MasterPointTable({ status, fail }: { status: MasterStatus; fail: (message: string) => void }) {
  const [rows, setRows] = useState<MasterPoint[]>([]);
  const [filter, setFilter] = useState('');
  const [onlyWatched, setOnlyWatched] = useState(false);
  const [page, setPage] = useState(0);
  const [draft, setDraft] = useState<MasterCommand | null>(null);
  const [note, setNote] = useState('');
  // A command drafted for another protocol must not survive a protocol switch (change 0089).
  useEffect(() => setDraft(null), [status.protocol]);
  const file = useRef<HTMLInputElement>(null);
  useEffect(() => {
    const load = () => masterApi.points().then(value => setRows(value.points)).catch(error => fail(error.message));
    load();
    const timer = setInterval(load, 1000);
    return () => clearInterval(timer);
  }, []);
  const untick = (keys: string[]) => {
    const hidden = new Set(keys);
    setRows(current => current.map(row => ({ ...row, ticked: !hidden.has(row.key) })));
    masterApi.untick(keys).catch(error => fail(error.message));
  };
  const toggle = (row: MasterPoint) => untick(rows.filter(item => item.key === row.key ? row.ticked : !item.ticked)
    .map(item => item.key));
  const rename = (row: MasterPoint, label: string) => label !== row.label && masterApi.label(row.key, label)
    .then(() => setRows(current => current.map(item => item.key === row.key ? { ...item, label } : item)))
    .catch(error => fail(error.message));
  const importNames = (picked: File) => picked.text().then(masterApi.importLabels)
    .then(answer => setNote(`${answer.imported} names read from ${picked.name}.${answer.problems.length
      ? ` ${answer.problems.slice(0, 3).join(' ')}` : ''}`)).catch(error => fail(error.message));
  const needle = filter.trim().toLowerCase();
  const shown = useMemo(() => rows.filter(row => (!onlyWatched || row.ticked) && (!needle ||
    `${row.label} ${row.address} ${row.group_name} ${row.type} ${row.value_text}`.toLowerCase().includes(needle))),
  [rows, needle, onlyWatched]);
  const pages = Math.max(1, Math.ceil(shown.length / PAGE));
  const first = Math.min(page, pages - 1) * PAGE;
  const watched = rows.filter(row => row.ticked).length;
  return <>
    <div className="guide-section">
      <p className="small muted row master-hint" style={{ margin: 0 }}><span>{MASTER_COPY.pointsHint}</span>
        <span className="gateway-spacer" /><strong>{MASTER_COPY.points(rows.length)} · {watched} watched</strong></p>
      {draft && <MasterCommandForm key={`${draft.kind}-${draft.address}`} draft={draft} host={status.host} protocol={status.protocol}
        close={() => setDraft(null)} fail={fail} />}
      <div className="row gateway-toolbar master-toolbar">
        <button onClick={() => { const kind = COMMANDS[status.protocol][0]; setDraft({ kind, address: undefined,
          value: startValue(kind), select: status.protocol !== 'modbus', code: kind === 'binary_output' ? 'latch' : undefined }); }}
          disabled={!status.link_open || COMMANDS[status.protocol].length === 0}><Send size={16} />{MASTER_COPY.openForm}</button>
        <button onClick={() => untick([])}>{MASTER_COPY.tickAll}</button>
        <button onClick={() => untick(rows.map(row => row.key))}>{MASTER_COPY.untickAll}</button>
        <label className="row gateway-check small"><input type="checkbox" checked={onlyWatched}
          onChange={event => { setOnlyWatched(event.target.checked); setPage(0); }} />{MASTER_COPY.onlyWatched}</label>
        <a className="button" href={masterApi.csvUrl()} download><Download size={16} />{MASTER_COPY.exportCsv}</a>
        <button onClick={() => file.current?.click()}><Upload size={16} />{MASTER_COPY.importLabels}</button>
        <input ref={file} type="file" accept=".csv,text/csv" hidden aria-label="CSV point names"
          onChange={event => { const picked = event.target.files?.[0]; if (picked) importNames(picked); event.target.value = ''; }} />
        <label className="search-field"><Search size={16} /><input value={filter} placeholder="Filter points"
          aria-label="Filter points" onChange={event => { setFilter(event.target.value); setPage(0); }} /></label>
      </div>
      {note && <p className="small muted" style={{ margin: 0 }}>{note}</p>}
      {rows.length > 0 && rows[0].stale && <p className="small gateway-warning" style={{ margin: 0 }}>{MASTER_COPY.stale}</p>}
    </div>
    {rows.length === 0 ? <p className="muted watch-empty">{status.protocol === 'modbus' ? MASTER_COPY.noModbusPoints : MASTER_COPY.noPoints}</p> :
      <div className="table-scroll"><table className="gateway-points master-points"><thead><tr>
        <th aria-label="Watched" /><th>Type</th><th>Address</th><th>Name</th><th>Value</th><th>Quality</th><th>Last cause</th>
        <th aria-label="Command" /></tr></thead>
        <tbody>{shown.slice(first, first + PAGE).map(row => <tr key={row.key} className={row.ticked ? '' : 'unwatched'}>
          <td><input type="checkbox" checked={row.ticked} aria-label={`Watch ${row.label || row.key}`}
            onChange={() => toggle(row)} /></td>
          <td><span className="type-cell">{row.group_name}<span className="small muted">{row.type}</span></span></td>
          <td className="mono">{row.address}</td>
          <td><input defaultValue={row.label} key={`${row.key}-${row.label}`} placeholder={MASTER_COPY.labelPlaceholder}
            aria-label={`Name of ${row.key}`} onBlur={event => rename(row, event.target.value.trim())} /></td>
          <td className={row.stale ? 'muted' : ''}><strong>{row.value_text}</strong>
            {row.changes > 0 && <span className="small muted"> · {row.changes} {row.changes === 1 ? 'change' : 'changes'}</span>}</td>
          <td><span className={row.good ? 'quality-good' : 'quality-bad'}>{row.good ? '✓ ' : '⚠ '}{row.quality_text}</span></td>
          <td className="small muted">{row.cause}</td>
          <td>{row.command && <button className="ghost" disabled={!status.link_open} onClick={() => setDraft({
            kind: row.command!, address: row.address, value: startValue(row.command!), select: status.protocol !== 'modbus',
            code: row.command === 'binary_output' ? 'latch' : undefined })}>
            <Send size={14} />{MASTER_COPY.sendTo}</button>}</td>
        </tr>)}</tbody></table></div>}
    {pages > 1 && <div className="pager"><span>{shown.length} points · page {Math.min(page, pages - 1) + 1} of {pages}</span>
      <span className="row"><button disabled={page === 0} onClick={() => setPage(page - 1)}>Previous</button>
        <button disabled={page >= pages - 1} onClick={() => setPage(page + 1)}>Next</button></span></div>}
  </>;
}
