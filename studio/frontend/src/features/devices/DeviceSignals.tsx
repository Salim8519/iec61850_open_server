import { Search } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Ied, Project, Signal } from '../../api/types';
import { SignalTable } from './SignalTable';

const PAGE = 100;

export function DeviceSignals({ ied, project, values, perform, prepare }: {
  ied: Ied; project: Project; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const [query, setQuery] = useState('');
  const [fc, setFc] = useState('');
  const [report, setReport] = useState('');
  const [offset, setOffset] = useState(0);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [message, setMessage] = useState('');
  const running = project.runtime !== 'stopped';
  useEffect(() => { setOffset(0); }, [query, fc, report, ied.id]);
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => api<{ items: Signal[]; total: number }>(`/ieds/${ied.id}/signals?q=${encodeURIComponent(query)}`
      + `&fc=${fc}&offset=${offset}&report=${encodeURIComponent(report)}`)
      .then(result => { if (!cancelled) { setSignals(result.items); setTotal(result.total); } })
      .catch(reason => { if (!cancelled) setMessage(reason.message); }), 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [ied.id, query, fc, offset, report, running]);
  const addTest = (signal: Signal) => perform(async () => {
    await prepare();
    await api('/project/test-panel', 'PUT', [...project.test_panel, { id: crypto.randomUUID(),
      label: (signal.description || signal.ref).slice(0, 100), ied_id: ied.id, ref: signal.ref, report_ref: null,
      duration_ms: 1000, active_value: true }]);
    setMessage(`Added “${signal.description || signal.ref}” to Test points.`);
  });
  return <>
    <div className="card-heading" style={{ borderTop: 0 }}>
      <div className="row" style={{ flexWrap: 'wrap' }}>
        <div className="search-field"><Search size={16} /><input aria-label="Search data points" placeholder="Search data points…"
          value={query} onChange={event => setQuery(event.target.value)} /></div>
        <select style={{ width: 150 }} aria-label="Kind of data" value={fc} onChange={event => setFc(event.target.value)}>
          <option value="">All kinds</option><option value="ST">Status (ST)</option><option value="MX">Measurements (MX)</option>
          <option value="CF">Configuration (CF)</option><option value="SP">Settings (SP)</option><option value="CO">Controls (CO)</option>
          <option value="DC">Descriptions (DC)</option></select>
        {!!ied.reports.length && <select style={{ width: 220 }} aria-label="Only points in report" value={report}
          onChange={event => setReport(event.target.value)}><option value="">All points</option>
          {ied.reports.map(item => <option key={item.ref} value={item.ref}>Report {item.name}</option>)}</select>}
      </div>
      <span className="small muted" role="status">{message || `${total.toLocaleString()} data points`}</span>
    </div>
    <SignalTable signals={signals} values={values} iedId={ied.id} running={running} perform={perform} addTest={addTest} />
    <div className="pager"><span>{total ? offset + 1 : 0}–{Math.min(offset + PAGE, total)} of {total.toLocaleString()}</span>
      <div className="row"><button disabled={!offset} onClick={() => setOffset(Math.max(0, offset - PAGE))}>Previous</button>
        <button disabled={offset + PAGE >= total} onClick={() => setOffset(offset + PAGE)}>Next</button></div></div>
  </>;
}
