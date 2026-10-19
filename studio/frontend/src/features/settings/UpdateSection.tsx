import { useState } from 'react';
import { Download, RefreshCw } from 'lucide-react';
import { updateProblem, type UpdateStatus } from '../../api/updates';
import { SpinIcon } from '../../ui/Spinner';

/** Settings › About: version, last check, Check now and the once-a-day switch (ADR-0071). */
export function UpdateSection({ status, change, checkNow }: {
  status: UpdateStatus | null; change: (changes: { auto?: boolean }) => Promise<unknown>; checkNow: () => Promise<unknown>;
}) {
  const [busy, setBusy] = useState(false);
  if (!status) return null;
  const checked = status.checked_at ? new Date(status.checked_at * 1000).toLocaleString() : 'not yet';
  const latest = status.latest;
  const run = async () => { setBusy(true); try { await checkNow(); } finally { setBusy(false); } };
  return <div className="card-body update-section">
    <h3 style={{ margin: 0 }}>Updates</h3>
    <p className="small" style={{ margin: 0 }}>
      {latest ? <>Version <strong>{latest.version}</strong> is ready ({latest.date}). You have {status.current}.</>
        : <>You have {status.current}{status.checked_at && !status.error ? ', the newest version' : ''}.</>}
      {' '}<span className="muted">Last checked: {checked}.</span></p>
    {latest?.notes && <p className="small muted" style={{ margin: 0 }}>What's new: {latest.notes}</p>}
    {status.error && <p className="small muted" style={{ margin: 0 }}>{updateProblem(status.error)}</p>}
    <div className="row">
      {latest && <a className="button primary" href={latest.page} target="_blank" rel="noreferrer">
        <Download size={16} aria-hidden />Open the download page</a>}
      <button onClick={run} disabled={busy}>{busy ? <SpinIcon /> : <RefreshCw size={16} />}
        {busy ? 'Checking…' : 'Check now'}</button>
    </div>
    <label className="row small" style={{ gap: 8 }}>
      <input type="checkbox" checked={status.auto} onChange={event => change({ auto: event.target.checked })} />
      Check for a new version once a day (only the version number is asked; nothing about this PC is sent)</label>
  </div>;
}
