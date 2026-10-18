import { Loader2, Radar, Telescope } from 'lucide-react';
import { useMemo } from 'react';
import type { ScanStatus } from '../../../api/tools';
import { FIND_COPY } from '../../../copy/tools';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { choicesOf, effective } from '../../../ui/filterState';
import { HelpTip } from '../../../ui/HelpTip';
import { StatusPill } from '../../../ui/StatusPill';
import { FIND_FILTER, foundMatches } from './findFilters';

/** The scan results: a filtered table that takes the rest of the window, or what will appear here. */
export function FindResults({ scan, percent, known, openExplorer }: {
  scan: ScanStatus | null; percent: number; known: string[]; openExplorer: (host: string, port: number) => void;
}) {
  const filter = useListFilter('find-results', FIND_FILTER);
  const found = scan?.found ?? [];
  const options = useMemo(() => ({ vendor: choicesOf(found.map(device => device.vendor)) }), [found]);
  const values = effective(FIND_FILTER, filter.values, options);
  const knownSet = useMemo(() => new Set(known), [known]);
  const shown = found.filter(device => foundMatches(device, values, knownSet));
  const scanning = scan?.state === 'scanning';
  if (!scan || scan.state === 'idle') return <div className="card fill-body panel-empty"><div className="guide-empty">
    <span className="guide-empty-icon"><Radar size={24} /></span><h3>{FIND_COPY.emptyTitle}</h3><p>{FIND_COPY.emptyHint}</p>
  </div></div>;
  return <div className="card fill-body fill-panel">
    <div className="panel-head"><h3>Results</h3>
      <StatusPill status={scanning ? { label: `Scanning ${percent}%`, tone: 'waiting', hint: scan.message }
        : scan.state === 'failed' ? { label: 'Scan failed', tone: 'problem', hint: scan.message }
        : { label: `${found.length} found`, tone: found.length ? 'good' : 'off', hint: scan.message }} />
      <span className="small muted" aria-live="polite">{scan.message}{!scanning && scan.seconds ? ` (${scan.seconds} s)` : ''}</span>
    </div>
    <FilterBar label="Filter results" spec={FIND_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={found.length} options={options}
      extra={found.length > 0 && !scanning ? <HelpTip text={FIND_COPY.addHint} label="How do I use a result?" /> : undefined} />
    <div className="panel-body">
      {found.length > 0 ? <div className="table-scroll"><table><thead><tr><th>Address</th><th>Vendor</th><th>Model</th>
        <th>Firmware</th><th>Logical devices</th><th>Answer</th><th aria-label="Open" /></tr></thead>
        <tbody>{shown.map(device => <tr key={device.ip}>
          <td><strong className="mono">{device.ip}{device.port !== 102 ? `:${device.port}` : ''}</strong>
            {knownSet.has(device.ip) && <div className="small muted">{FIND_COPY.inProject}</div>}</td>
          <td>{device.identified === null ? <span className="muted"><Loader2 size={13} className="spin" /> Asking…</span>
            : device.vendor || <span className="muted">—</span>}</td>
          <td>{device.model || <span className="muted">—</span>}</td><td>{device.revision || <span className="muted">—</span>}</td>
          <td className="small">{device.devices.length ? device.devices.join(', ') : <span className="muted">{device.reason}</span>}</td>
          <td className="small muted">{device.tcp_ms < 1 ? '<1' : Math.round(device.tcp_ms)} ms</td>
          <td><button disabled={device.identified !== true} onClick={() => openExplorer(device.ip, device.port)}>
            <Telescope size={15} />{FIND_COPY.open}</button></td></tr>)}</tbody></table>
        {shown.length === 0 && <p className="muted watch-empty">{FIND_COPY.noMatch}</p>}</div>
        : !scanning && <p className="muted watch-empty">{FIND_COPY.none}</p>}
    </div>
  </div>;
}
