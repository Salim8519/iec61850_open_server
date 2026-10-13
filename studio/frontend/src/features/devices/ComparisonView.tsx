import { Download, GitCompare } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Comparison } from '../../api/network';
import { EmptyState } from '../../ui/EmptyState';
import { Hint } from '../../ui/Hint';
import { StatusPill } from '../../ui/StatusPill';
import { COMPARE } from '../../copy/compare';

const ORDER = ['missing_on_device', 'type_mismatch', 'extra_on_device', 'match'];

export function ComparisonView({ iedId, ready }: { iedId: string; ready: boolean }) {
  const [result, setResult] = useState<Comparison | null>(null);
  const [filter, setFilter] = useState('problems');
  const [error, setError] = useState('');
  useEffect(() => {
    setResult(null);
    if (!ready) return;
    api<Comparison>(`/live/${iedId}/comparison`).then(setResult).catch(reason => setError(reason.message));
  }, [iedId, ready]);
  if (!ready || !result) return <div><EmptyState icon={<GitCompare size={20} />}
    title="SCD compared with the device" hintId={error ? 'live.compare' : 'live.compare_wait'} /></div>;
  const differences = result.signals.filter(row => row.category !== 'match').length
    + result.summary.datasets_different + result.summary.reports_missing;
  const rows = result.signals.filter(row => filter === 'all' || (filter === 'problems' ? row.category !== 'match' : row.category === filter));
  return <div>
    <div className="card-heading"><h2>SCD compared with the device</h2>
      <a className="button" href={`/api/v1/live/${iedId}/comparison.csv`}><Download size={14} />Download CSV</a></div>
    <div className="guide-section">
      <Hint id="live.compare" />
      <div className="tile-row">{ORDER.map(key => <button key={key} type="button" className={`tile choice${filter === key ? ' selected' : ''}`}
        onClick={() => setFilter(key)}><strong>{result.summary[key] ?? 0}</strong><span>{COMPARE[key].label}</span></button>)}
        <button type="button" className={`tile choice${filter === 'problems' ? ' selected' : ''}`} onClick={() => setFilter('problems')}>
          <strong>{differences}</strong><span>All differences</span></button>
      </div>
      {filter !== 'all' && filter !== 'match' && COMPARE[filter] && <StatusPill status={COMPARE[filter]} showHint />}
    </div>
    <div className="table-scroll"><table><thead><tr><th>Reference</th><th>FC</th><th>Result</th><th>SCD type</th><th>Device type</th></tr></thead>
      <tbody>{rows.slice(0, 400).map(row => <tr key={`${row.ref}-${row.fc}`}>
        <td className="reference-cell" title={row.ref}><code>{row.ref}</code></td><td><span className="badge">{row.fc}</span></td>
        <td><StatusPill status={COMPARE[row.category]} /></td><td className="muted">{row.scd_type ?? '—'}</td>
        <td className="muted">{row.device_type ?? '—'}</td></tr>)}</tbody></table>
      {rows.length > 400 && <p className="small muted" style={{ padding: 12 }}>Showing 400 of {rows.length}. Download the CSV for all rows.</p>}
      {!rows.length && <EmptyState icon={<GitCompare size={20} />} title={filter === 'problems' && !differences
        ? 'Everything matches the SCD' : 'Nothing to show'} hintId={filter === 'problems' && !differences ? 'live.compare_match' : 'live.compare_clean'} />}
    </div>
    {result.datasets.filter(item => item.status !== 'match').map(item => <div key={item.ref} className="guide-section">
      <StatusPill status={COMPARE[item.status === 'different' ? 'dataset_different' : 'missing_on_device']} />
      <code>{item.ref}</code>
      {!!item.missing.length && <span className="small">Missing on device: {item.missing.join(', ')}</span>}
      {!!item.extra.length && <span className="small">Extra on device: {item.extra.join(', ')}</span>}
    </div>)}
  </div>;
}
