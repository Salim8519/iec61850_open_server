import { Download, GitCompare } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Comparison } from '../../api/network';
import { hint } from '../../copy/hints';
import { FcTag } from '../../components/TypeCell';
import { COMPARE } from '../../copy/compare';
import { KindBadge, kindRow } from '../../ui/KindBadge';
import { countKinds, kindOf61850 } from '../../points/pointKind';
import { useDataClasses } from '../../points/dataClasses';
import { useListFilter } from '../../state/useListFilter';
import { EmptyState } from '../../ui/EmptyState';
import { Spinner } from '../../ui/Spinner';
import { FilterBar } from '../../ui/FilterBar';
import { HelpTip } from '../../ui/HelpTip';
import { StatusPill } from '../../ui/StatusPill';
import { kindMatches } from '../../ui/kindFilter';
import { COMPARE_FILTER, compareKind, compareMatches, POINT_KIND } from './deviceFilters';

const LIMIT = 400;

/** One side of the comparison: the kind badge and the exact data type, which is what differs on a mismatch. */
function typeOf(fc: string, type: string | null, dataClass?: string) {
  return type ? <span className="type-cell"><KindBadge kind={kindOf61850(fc, type, dataClass)} raw={type} />
    <code className="small muted">{type}</code></span> : <span className="muted">—</span>;
}

/** SCD compared with the real device: "Differences only" first, the table fills the tab (change 0093). */
export function ComparisonView({ iedId, ready }: { iedId: string; ready: boolean }) {
  const [result, setResult] = useState<Comparison | null>(null);
  const filter = useListFilter('compare', COMPARE_FILTER);
  const classOf = useDataClasses(iedId);
  const [error, setError] = useState('');
  useEffect(() => {
    setResult(null);
    if (!ready) return;
    api<Comparison>(`/live/${iedId}/comparison`).then(setResult).catch(reason => setError(reason.message));
  }, [iedId, ready]);
  // While running, an empty result means the device model is still being read: say so, not "press Run".
  if (!ready || !result) return <div className="fill-scroll"><EmptyState icon={<GitCompare size={20} />}
    title="SCD compared with the device" hintId={error || ready ? 'live.compare' : 'live.compare_wait'}
    action={ready && !error ? <Spinner label="Reading the device and comparing it with the SCD…"
      slowHint="Still reading. Large devices can take a minute." /> : undefined} /></div>;
  const differences = result.signals.filter(row => row.category !== 'match').length
    + result.summary.datasets_different + result.summary.reports_missing;
  const counts = { problems: differences, all: result.signals.length, missing_on_device: result.summary.missing_on_device ?? 0,
    extra_on_device: result.summary.extra_on_device ?? 0, type_mismatch: result.summary.type_mismatch ?? 0 };
  // Chip counts follow the result and search choices; the point-type chip then narrows the rows (ADR-0095).
  const inResult = result.signals.map(row => ({ row, kind: compareKind(row, classOf) }))
    .filter(item => compareMatches(item.row, filter.values));
  const rows = inResult.filter(item => kindMatches(filter.values[POINT_KIND], item.kind));
  const kindCounts = countKinds(inResult, item => item.kind);
  const show = String(filter.values.show);
  const datasets = result.datasets.filter(item => item.status !== 'match');
  return <>
    <FilterBar label="Filter the comparison" spec={COMPARE_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={rows.length} total={result.signals.length} counts={{ show: counts, [POINT_KIND]: kindCounts }} extra={<>
        <HelpTip text={`${hint('live.compare')}${COMPARE[show] ? ` ${COMPARE[show].label}: ${COMPARE[show].hint}` : ''}`}
          label="How do I read the comparison?" />
        <a className="button" href={`/api/v1/live/${iedId}/comparison.csv`}><Download size={14} />Download CSV</a></>} />
    <div className="table-scroll"><table className="compact-rows"><thead><tr><th>Reference</th><th>FC</th><th>Result</th><th>SCD type</th><th>Device type</th></tr></thead>
      <tbody>{rows.slice(0, LIMIT).map(({ row, kind }) => <tr key={`${row.ref}-${row.fc}`} {...kindRow(kind)}>
        <td className="reference-cell point-cell" title={row.ref}><code>{row.ref}</code></td><td><FcTag fc={row.fc} /></td>
        <td><StatusPill status={COMPARE[row.category]} /></td>
        <td>{typeOf(row.fc, row.scd_type, classOf(row.ref))}</td><td>{typeOf(row.fc, row.device_type, classOf(row.ref))}</td></tr>)}</tbody></table>
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
