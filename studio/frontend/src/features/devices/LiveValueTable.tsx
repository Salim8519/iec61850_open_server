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
import { SpinIcon, Spinner } from '../../ui/Spinner';
import { LIVE_FILTER, signalQuery } from './deviceFilters';
import { useSignalValue } from '../../state/useLive';

type Read = { values: Record<string, { value: unknown; live_state?: string }>; errors: Record<string, string> };

/** One reference; it reads its own live value, so only changed rows repaint (speed lane F). */
function LiveRow({ signal, iedId, read }: { signal: Signal; iedId: string; read: Read | null }) {
  const reported = useSignalValue(`${iedId}:${signal.ref}`);
  const quality = useSignalValue(signal.quality_ref ? `${iedId}:${signal.quality_ref}` : null);
  const value = read?.values[signal.ref]?.value ?? reported;
  const stale = read?.values[signal.ref]?.live_state === 'stale';
  return <tr><td className="reference-cell" title={signal.ref}><code>{signal.ref}</code></td>
    <td><span className="badge">{signal.fc}</span></td>
    <td>{read?.errors[signal.ref] ? <span style={{ color: 'var(--error)' }}>{read.errors[signal.ref]}</span>
      : value === undefined ? <span className="muted">Not read yet</span> : <code>{String(value)}</code>}
      {stale && <StatusPill status={UI_STATUS.stale} />}</td>
    <td className="muted">{quality === undefined ? '—' : Number(quality) === 0 ? 'Good' : `Flags ${quality}`}</td></tr>;
}

/** The Data points tab of a real IED: the first 50 matching references, read on request. */
export function LiveValueTable({ iedId, live }: { iedId: string; live: boolean }) {
  const filter = useListFilter('live-points', LIVE_FILTER);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [read, setRead] = useState<Read | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [reading, setReading] = useState(false);
  const query = signalQuery(filter.values, 0);
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const timer = setTimeout(() => api<{ items: Signal[]; total: number }>(`/ieds/${iedId}/signals?${query}`)
      .then(result => { if (!cancelled) { setSignals(result.items.slice(0, 50)); setTotal(result.total); } })
      .catch(reason => { if (!cancelled) setError(reason.message); })
      .finally(() => { if (!cancelled) setLoading(false); }), 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [iedId, query]);
  useEffect(() => setRead(null), [iedId, query]);
  const readNow = () => {
    setReading(true);
    api<Read>(`/live/${iedId}/read`, 'POST', { refs: signals.map(item => item.ref) })
      .then(result => { setRead(result); setError(''); }).catch(reason => setError(reason.message))
      .finally(() => setReading(false));
  };
  return <>
    <FilterBar label="Filter live values" spec={LIVE_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={Math.min(total, 50)} total={total} note={total > 50 ? '(first 50)' : ''} extra={<>
        <HelpTip text={hint(live ? 'live.values' : 'live.not_running')} label="How are live values read?" />
        <button disabled={!live || !signals.length || reading} onClick={readNow}>
          {reading ? <SpinIcon size={14} /> : <RefreshCw size={14} />}{reading ? 'Reading from the IED…' : 'Read these now'}</button></>} />
    {error && <p role="alert" className="panel-note side-error">{error}</p>}
    <div className="table-scroll"><table><thead><tr><th>Reference</th><th>FC</th><th>Value</th><th><InfoTip term="quality" label="Quality" /></th></tr></thead>
      <tbody>{signals.map(signal => <LiveRow key={signal.ref} signal={signal} iedId={iedId} read={read} />)}</tbody></table>{!signals.length && (loading ? <div className="empty"><Spinner label="Loading data points…" /></div>
        : <div className="empty">No data points match your search.</div>)}</div>
  </>;
}
