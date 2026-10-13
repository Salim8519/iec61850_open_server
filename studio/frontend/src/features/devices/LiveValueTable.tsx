import { RefreshCw, Search } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Signal } from '../../api/types';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { UI_STATUS } from '../../copy/status';

type Read = { values: Record<string, { value: unknown; live_state?: string }>; errors: Record<string, string> };

export function LiveValueTable({ iedId, live, values }: { iedId: string; live: boolean; values: Record<string, unknown> }) {
  const [query, setQuery] = useState('');
  const [signals, setSignals] = useState<Signal[]>([]);
  const [read, setRead] = useState<Read | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    const timer = setTimeout(() => api<{ items: Signal[] }>(`/ieds/${iedId}/signals?q=${encodeURIComponent(query)}&offset=0`)
      .then(result => setSignals(result.items.slice(0, 50))).catch(reason => setError(reason.message)), 150);
    return () => clearTimeout(timer);
  }, [iedId, query]);
  useEffect(() => setRead(null), [iedId, query]);
  const readNow = () => api<Read>(`/live/${iedId}/read`, 'POST', { refs: signals.map(item => item.ref) })
    .then(result => { setRead(result); setError(''); }).catch(reason => setError(reason.message));
  return <div>
    <div className="card-heading"><h2>Live values</h2><div className="row">
      <div className="search-field"><Search size={14} /><input placeholder="Search references" value={query}
        onChange={event => setQuery(event.target.value)} /></div>
      <button disabled={!live || !signals.length} onClick={readNow}><RefreshCw size={14} />Read these now</button></div></div>
    <div className="guide-section"><Hint id={live ? 'live.values' : 'live.not_running'} />
      {error && <p role="alert" style={{ color: 'var(--error)', margin: 0 }}>{error}</p>}</div>
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
      })}</tbody></table></div>
  </div>;
}
