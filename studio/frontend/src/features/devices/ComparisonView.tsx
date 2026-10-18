import { Download, GitCompare } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Comparison } from '../../api/network';
import { hint } from '../../copy/hints';
import { COMPARE } from '../../copy/compare';
import { useListFilter } from '../../state/useListFilter';
import { EmptyState } from '../../ui/EmptyState';
import { FilterBar } from '../../ui/FilterBar';
import { HelpTip } from '../../ui/HelpTip';
import { StatusPill } from '../../ui/StatusPill';
import { COMPARE_FILTER, compareMatches } from './deviceFilters';

const LIMIT = 400;

/** SCD compared with the real device: "Differences only" first, the table fills the tab (change 0093). */
export function ComparisonView({ iedId, ready }: { iedId: string; ready: boolean }) {
  const [result, setResult] = useState<Comparison | null>(null);
  const filter = useListFilter('compare', COMPARE_FILTER);
  const [error, setError] = useState('');
  useEffect(() => {
    setResult(null);
    if (!ready) return;
    api<Comparison>(`/live/${iedId}/comparison`).then(setResult).catch(reason => setError(reason.message));
  }, [iedId, ready]);
  if (!ready || !result) return <div className="fill-scroll"><EmptyState icon={<GitCompare size={20} />}
    title="SCD compared with the device" hintId={error ? 'live.compare' : 'live.compare_wait'} /></div>;
  const differences = result.signals.filter(row => row.category !== 'match').length
    + result.summary.datasets_different + result.summary.reports_missing;
  const counts = { problems: differences, all: result.signals.length, missing_on_device: result.summary.missing_on_device ?? 0,
    extra_on_device: result.summary.extra_on_device ?? 0, type_mismatch: result.summary.type_mismatch ?? 0 };
  const rows = result.signals.filter(row => compareMatches(row, filter.values));
  const show = String(filter.values.show);
  const datasets = result.datasets.filter(item => item.status !== 'match');
  return <>
    <FilterBar label="Filter the comparison" spec={COMPARE_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={rows.length} total={result.signals.length} counts={{ show: counts }} extra={<>
        <HelpTip text={`${hint('live.compare')}${COMPARE[show] ? ` ${COMPARE[show].label}: ${COMPARE[show].hint}` : ''}`}
          label="How do I read the comparison?" />
        <a className="button" href={`/api/v1/live/${iedId}/comparison.csv`}><Download size={14} />Download CSV</a></>} />
    <div className="table-scroll"><table><thead><tr><th>Reference</th><th>FC</th><th>Result</th><th>SCD type</th><th>Device type</th></tr></thead>
      <tbody>{rows.slice(0, LIMIT).map(row => <tr key={`${row.ref}-${row.fc}`}>
        <td className="reference-cell" title={row.ref}><code>{row.ref}</code></td><td><span className="badge">{row.fc}</span></td>
        <td><StatusPill status={COMPARE[row.category]} /></td><td className="muted">{row.scd_type ?? '—'}</td>
        <td className="muted">{row.device_type ?? '—'}</td></tr>)}</tbody></table>
      {rows.length > LIMIT && <p className="small muted" style={{ padding: 12 }}>Showing {LIMIT} of {rows.length}. Download the CSV for all rows.</p>}
      {!rows.length && <EmptyState icon={<GitCompare size={20} />} title={show === 'problems' && !differences
        ? 'Everything matches the SCD' : 'Nothing to show'} hintId={show === 'problems' && !differences ? 'live.compare_match' : 'live.compare_clean'} />}
      {datasets.map(item => <div key={item.ref} className="guide-section">
        <StatusPill status={COMPARE[item.status === 'different' ? 'dataset_different' : 'missing_on_device']} />
        <code>{item.ref}</code>
        {!!item.missing.length && <span className="small">Missing on device: {item.missing.join(', ')}</span>}
        {!!item.extra.length && <span className="small">Extra on device: {item.extra.join(', ')}</span>}
      </div>)}
    </div>
  </>;
}
