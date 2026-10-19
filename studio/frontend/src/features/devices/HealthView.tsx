import { Download, FileText, HeartPulse, RefreshCw } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../../api/client';
import { areaInfo, verdictStatus } from '../../copy/health';
import { Disclosure } from '../../ui/Disclosure';
import { EmptyState } from '../../ui/EmptyState';
import { Spinner } from '../../ui/Spinner';
import { Hint } from '../../ui/Hint';
import { StatusPill } from '../../ui/StatusPill';

export type HealthCheck = { area: string; item: string; result: string; message: string;
  observed_at: string | null; source: string; stale: boolean };
export type HealthIed = { id: string; name: string; role: string; verdict: string; checks: HealthCheck[] };
export type Health = { verdict: string; running: boolean; checked_at: string; ieds: HealthIed[];
  summary: Record<string, number>; unknown: Record<string, unknown>[]; unknown_note: string;
  simulation_sources?: string[] };

const ORDER = ['problem', 'warning', 'not_checked', 'ok'];

/** Does the real station match its SCD? One verdict per IED, worst first, with reasons. */
export function HealthView({ running }: { running: boolean }) {
  const [health, setHealth] = useState<Health | null>(null);
  const [open, setOpen] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const load = useCallback(() => api<Health>('/health').then(setHealth).catch(reason => setError(reason.message)), []);
  useEffect(() => {
    load();
    if (!running) return;
    const timer = setInterval(load, 3000);
    return () => clearInterval(timer);
  }, [load, running]);
  if (!health) return <div className="card"><EmptyState icon={<HeartPulse size={22} />} title="Health check"
    hintId="health.intro" action={error ? undefined : <Spinner label="Checking the station…" />} />{error && <p role="alert" className="import-unavailable">{error}</p>}</div>;
  const ieds = [...health.ieds].sort((a, b) => ORDER.indexOf(a.verdict) - ORDER.indexOf(b.verdict));
  return <div className="card">
    <div className="card-heading"><h2>Health check</h2><span className="row">
      <button type="button" disabled={!running || busy} onClick={async () => {
        setBusy(true); setError('');
        try { setHealth(await api<Health>('/health/refresh', 'POST')); }
        catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
        finally { setBusy(false); }
      }}><RefreshCw size={14} />{busy ? 'Reading…' : 'Run check again'}</button>
      <a className="button" href="/api/v1/health.csv"><Download size={14} />CSV</a>
      <a className="button" href="/api/v1/health/report.html"><FileText size={14} />Report</a></span></div>
    <div className="guide-section">
      <Hint id={running ? 'health.intro' : 'health.not_running'} />
      {!!health.simulation_sources?.length && <div className="warning-list" role="alert">
        <strong>Simulation traffic present</strong><Hint id="mode.simulation_traffic" compact />
        <ul className="small" style={{ margin: 0 }}>{health.simulation_sources.map(source => <li key={source}>{source}</li>)}</ul></div>}
      <div className="row"><strong>Whole station:</strong><StatusPill status={verdictStatus(health.verdict)} showHint /></div>
      <div className="facts">{ORDER.map(key => <div className="fact" key={key}>
        <span>{verdictStatus(key).label}</span><strong>{health.summary[key] ?? 0}</strong></div>)}</div>
      {error && <p role="alert" className="import-unavailable">{error}</p>}
    </div>
    <div className="list" role="list" style={{ padding: '0 20px 20px' }}>{ieds.map(ied => <div key={ied.id} role="listitem">
      <button type="button" className={`list-row${open === ied.id ? ' active' : ''}`} aria-expanded={open === ied.id}
        onClick={() => setOpen(open === ied.id ? '' : ied.id)}>
        <span className="list-text"><strong>{ied.name}</strong><small>{ied.role === 'real' ? 'Reads real IED' :
          ied.role === 'simulated' ? 'Simulated by ScadaBench' : 'Off'} · {ied.checks.length} checks</small></span>
        <StatusPill status={verdictStatus(ied.verdict)} /></button>
      {open === ied.id && <div className="table-scroll"><table><thead><tr><th>Area</th><th>What</th><th>Result</th>
        <th>Why / what to do</th><th>Seen</th></tr></thead><tbody>
        {[...ied.checks].sort((a, b) => ORDER.indexOf(a.result) - ORDER.indexOf(b.result)).map((check, index) =>
          <tr key={`${check.area}-${check.item}-${index}`}>
            <td title={areaInfo(check.area).hint}>{areaInfo(check.area).label}</td>
            <td className="reference-cell"><code title={check.item}>{check.item}</code></td>
            <td><StatusPill status={verdictStatus(check.result)} /></td>
            <td>{check.message}</td>
            <td className="small muted">{check.observed_at ? `${check.observed_at.slice(11, 19)} · ${check.source}` : '—'}
              {check.stale && <div>Old result</div>}</td></tr>)}</tbody></table></div>}
    </div>)}</div>
    {health.unknown.length > 0 && <div className="guide-section"><Disclosure title={`GOOSE not in the project (${health.unknown.length})`}>
      <Hint id="goose.unknown" compact />{health.unknown_note && <p className="small">{health.unknown_note}</p>}
      <div className="table-scroll"><table><thead><tr><th>GOOSE block</th><th>APPID</th><th>Destination</th><th>Source</th>
        <th>VLAN</th><th>GOOSE ID</th><th>confRev</th><th>stNum</th></tr></thead><tbody>
        {health.unknown.map(item => <tr key={String(item.gocb_ref)}>{['gocb_ref', 'app_id', 'mac', 'src_mac', 'vlan_id',
          'go_id', 'conf_rev', 'st_num'].map(key => <td key={key}><code>{item[key] === null || item[key] === undefined
            ? '—' : String(item[key])}</code></td>)}</tr>)}</tbody></table></div></Disclosure></div>}
  </div>;
}
