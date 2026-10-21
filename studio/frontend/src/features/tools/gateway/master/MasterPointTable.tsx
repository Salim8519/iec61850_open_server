import { ChevronDown, Download, PlugZap, Send, Upload } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { MASTER_POINTS, masterApi, type MasterCommand, type MasterPoint, type MasterStatus } from '../../../../api/master';
import { liveFeed } from '../../../../api/polls';
import { GATEWAY_COPY } from '../../../../copy/gateway';
import { COMMANDS, MASTER_COPY } from '../../../../copy/master';
import { kindOfMasterGroup } from '../../../../points/pointKind';
import { useListFilter } from '../../../../state/useListFilter';
import { EditableText } from '../../../../ui/EditableText';
import { FilterBar } from '../../../../ui/FilterBar';
import { choicesOf, effective } from '../../../../ui/filterState';
import { HelpTip } from '../../../../ui/HelpTip';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { Menu } from '../../../../ui/Menu';
import { kindCounts } from '../kindCounts';
import { Pager } from '../Pager';
import { MASTER_POINT_FILTER, masterPointMatches } from '../pointFilters';
import { saveFile } from '../saveFile';
import { MasterCommandForm, startValue } from './MasterCommandForm';

const PAGE = 100;

/** Every point the device reported: watched (ticked) by default, with names the user gives. */
export function MasterPointTable({ status, fail }: { status: MasterStatus; fail: (message: string) => void }) {
  const [rows, setRows] = useState<MasterPoint[]>([]);
  const [page, setPage] = useState(0);
  const [draft, setDraft] = useState<MasterCommand | null>(null);
  const [note, setNote] = useState('');
  const filter = useListFilter('master-points', MASTER_POINT_FILTER);
  // A command drafted for another protocol must not survive a protocol switch (change 0089).
  useEffect(() => setDraft(null), [status.protocol]);
  const file = useRef<HTMLInputElement>(null);
  useEffect(() => liveFeed.subscribe<{ points: MasterPoint[] }>(MASTER_POINTS, 1000, value => setRows(value.points),
    error => fail(error.message)), []);
  useEffect(() => setPage(0), [filter.values]);
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
  const options = useMemo(() => ({ group: choicesOf(rows.map(row => row.group_name)) }), [rows]);
  const values = useMemo(() => effective(MASTER_POINT_FILTER, filter.values, options), [filter.values, options]);
  const shown = useMemo(() => rows.filter(row => masterPointMatches(row, values)), [rows, values]);
  const counts = useMemo(() => kindCounts(rows, row => kindOfMasterGroup(row.group)), [rows]);
  const first = Math.min(page, Math.max(0, Math.ceil(shown.length / PAGE) - 1)) * PAGE;
  const watched = rows.filter(row => row.ticked).length;
  const connected = status.state === 'connected' || status.state === 'lost';
  return <>
    <div className="panel-toolbar master-toolbar">
      <button onClick={() => { const kind = COMMANDS[status.protocol][0]; setDraft({ kind, address: undefined,
        value: startValue(kind), select: status.protocol !== 'modbus', code: kind === 'binary_output' ? 'latch' : undefined }); }}
        disabled={!status.link_open || COMMANDS[status.protocol].length === 0}><Send size={16} />{MASTER_COPY.openForm}</button>
      <button onClick={() => untick([])}>{MASTER_COPY.tickAll}</button>
      <button onClick={() => untick(rows.map(row => row.key))}>{MASTER_COPY.untickAll}</button>
      <Menu label={MASTER_COPY.moreLabel} align="left" trigger={<>{GATEWAY_COPY.more}<ChevronDown size={15} aria-hidden /></>} items={[
        { label: MASTER_COPY.exportCsv, icon: <Download size={16} />, onSelect: () => saveFile(masterApi.csvUrl()) },
        { label: MASTER_COPY.importLabels, icon: <Upload size={16} />, onSelect: () => file.current?.click() }]} />
      <input ref={file} type="file" accept=".csv,text/csv" hidden aria-label="CSV point names"
        onChange={event => { const picked = event.target.files?.[0]; if (picked) importNames(picked); event.target.value = ''; }} />
      <HelpTip text={MASTER_COPY.pointsHint} label="What is this list?" />
      <span className="gateway-spacer" />
      <strong className="small">{MASTER_COPY.points(rows.length)} · {watched} watched</strong>
    </div>
    <FilterBar label="Filter points" spec={MASTER_POINT_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={rows.length} options={options} counts={counts} />
    {(draft || note || (rows.length > 0 && rows[0].stale)) && <div className="guide-section panel-messages">
      {draft && <MasterCommandForm key={`${draft.kind}-${draft.address}`} draft={draft} host={status.host} protocol={status.protocol}
        close={() => setDraft(null)} fail={fail} />}
      {note && <p className="small muted" style={{ margin: 0 }}>{note}</p>}
      {rows.length > 0 && rows[0].stale && <p className="small gateway-warning" style={{ margin: 0 }}>{MASTER_COPY.stale}</p>}
    </div>}
    {rows.length === 0 ? <div className="panel-empty"><div className="guide-empty">
      <span className="guide-empty-icon"><PlugZap size={24} /></span>
      <h3>{connected ? MASTER_COPY.points(0) : MASTER_COPY.emptyNotConnected}</h3>
      <p>{status.protocol === 'modbus' ? MASTER_COPY.noModbusPoints : MASTER_COPY.noPoints}</p></div></div> :
      <div className="table-scroll"><table className="gateway-points master-points compact-rows"><thead><tr>
        <th aria-label="Watched" /><th>Type</th><th>Address</th><th>Name</th><th>Value</th><th>Quality</th><th>Last cause</th>
        <th aria-label="Command" /></tr></thead>
        <tbody>{shown.slice(first, first + PAGE).map(row => {
          const kind = kindOfMasterGroup(row.group);
          return <tr key={row.key} {...kindRow(kind)} className={row.ticked ? '' : 'unwatched'}>
            <td><input type="checkbox" checked={row.ticked} aria-label={`Watch ${row.label || row.key}`}
              onChange={() => toggle(row)} /></td>
            <td><span className="type-cell"><KindBadge kind={kind} raw={`${row.group_name} · ${row.type}`} />
              <span className="small">{row.group_name}</span></span></td>
            <td className="mono">{row.address}</td>
            <td><EditableText value={row.label} label={`Name of ${row.key}`} placeholder={MASTER_COPY.labelPlaceholder}
              onSave={text => rename(row, text.trim())} /></td>
            <td className={row.stale ? 'muted' : ''}><strong>{row.value_text}</strong>
              {row.changes > 0 && <span className="small muted"> · {row.changes} {row.changes === 1 ? 'change' : 'changes'}</span>}</td>
            <td><span className={row.good ? 'quality-good' : 'quality-bad'}>{row.good ? '✓ ' : '⚠ '}{row.quality_text}</span></td>
            <td className="small muted">{row.cause}</td>
            <td>{row.command && <button className="ghost" disabled={!status.link_open} onClick={() => setDraft({
              kind: row.command!, address: row.address, value: startValue(row.command!), select: status.protocol !== 'modbus',
              code: row.command === 'binary_output' ? 'latch' : undefined })}>
              <Send size={14} />{MASTER_COPY.sendTo}</button>}</td>
          </tr>;
        })}</tbody></table>
        {shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
    <Pager count={shown.length} page={page} size={PAGE} setPage={setPage} noun="points" />
  </>;
}
