import { RefreshCw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Signal } from '../../api/types';
import { hint } from '../../copy/hints';
import { UI_STATUS } from '../../copy/status';
import { useListFilter } from '../../state/useListFilter';
import { FilterBar } from '../../ui/FilterBar';
import { HelpTip } from '../../ui/HelpTip';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { LIVE_FILTER, signalQuery } from './deviceFilters';

type Read = { values: Record<string, { value: unknown; live_state?: string }>; errors: Record<string, string> };

/** The Data points tab of a real IED: the first 50 matching references, read on request. */
export function LiveValueTable({ iedId, live, values }: { iedId: string; live: boolean; values: Record<string, unknown> }) {
  const filter = useListFilter('live-points', LIVE_FILTER);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [read, setRead] = useState<Read | null>(null);
  const [error, setError] = useState('');
  const query = signalQuery(filter.values, 0);
  useEffect(() => {
    const timer = setTimeout(() => api<{ items: Signal[]; total: number }>(`/ieds/${iedId}/signals?${query}`)
      .then(result => { setSignals(result.items.slice(0, 50)); setTotal(result.total); })
      .catch(reason => setError(reason.message)), 150);
    return () => clearTimeout(timer);
  }, [iedId, query]);
  useEffect(() => setRead(null), [iedId, query]);
  const readNow = () => api<Read>(`/live/${iedId}/read`, 'POST', { refs: signals.map(item => item.ref) })
    .then(result => { setRead(result); setError(''); }).catch(reason => setError(reason.message));
  return <>
    <FilterBar label="Filter live values" spec={LIVE_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={Math.min(total, 50)} total={total} note={total > 50 ? '(first 50)' : ''} extra={<>
        <HelpTip text={hint(live ? 'live.values' : 'live.not_running')} label="How are live values read?" />
        <button disabled={!live || !signals.length} onClick={readNow}><RefreshCw size={14} />Read these now</button></>} />
    {error && <p role="alert" className="panel-note side-error">{error}</p>}
    <div className="table-scroll"><table><thead><tr><th>Reference</th><th>FC</th><th>Value</th><th><InfoTip term="quality" label="Quality" /></th></tr></thead>
      <tbody>{signals.map(signal => {
        const value = read?.values[signal.ref]?.value ?? values[`${iedId}:${signal.ref}`];
        const stale = read?.values[signal.ref]?.live_state === 'stale';
        const quality = signal.quality_ref ? values[`${iedId}:${signal.quality_ref}`] : undefined;
        return <tr key={signal.ref}><td className="reference-cell" title={signal.ref}><code>{signal.ref}</code></td>
          <td><span className="badge">{signal.fc}</span></td>
          <td>{read?.errors[signal.ref] ? <span style={{ color: 'var(--error)' }}>{read.errors[signal.ref]}</span>
            : value === undefined ? <span className="muted">Not read yet</span> : <code>{String(value)}</code>}
            {stale && <StatusPill status={UI_STATUS.stale} />}</td>
          <td className="muted">{quality === undefined ? '—' : Number(quality) === 0 ? 'Good' : `Flags ${quality}`}</td></tr>;
      })}</tbody></table>{!signals.length && <div className="empty">No data points match your search.</div>}</div>
  </>;
}
