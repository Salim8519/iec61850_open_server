import { Check, ChevronRight, MonitorSmartphone, Wrench } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { api } from '../../api/client';
import { LAB, blocking, type LabStatus } from '../../api/lab';
import { liveFeed } from '../../api/polls';
import { LAB_COPY, labSummary, realCardLine } from '../../copy/labNetwork';
import { SpinIcon } from '../../ui/Spinner';
import { LabRowView } from './LabRows';
import './lab.css';

type Typed = Record<string, string>;

/** "This PC lab network" (ADR-0120): one compact card. One line and one button; the addresses are folded away
 *  and open by themselves when an IED needs attention. The engine decides every address and clash. */
export function LabNetwork({ locked, perform }: { locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void> }) {
  const [status, setStatus] = useState<LabStatus | null>(null);
  const [typed, setTyped] = useState<Typed>({});
  const [working, setWorking] = useState<'' | 'create' | 'use' | 'move' | 'remove'>('');
  const [error, setError] = useState('');
  const typedRef = useRef(typed);
  typedRef.current = typed;
  const active = !!status?.active;
  // The shared live feed keeps the card current; while addresses are typed before "Use", the typed check owns it.
  useEffect(() => liveFeed.subscribe<LabStatus>(LAB, 3000, answer => {
    if (answer.active || !Object.keys(typedRef.current).length) setStatus(answer);
  }, reason => setError(reason.message)), []);
  useEffect(() => {
    if (active || !Object.keys(typed).length) return undefined;
    const timer = window.setTimeout(() => {
      api<LabStatus>(`${LAB}/check`, 'POST', { addresses: typed }).then(setStatus).catch(reason => setError(reason.message));
    }, status?.checking ? 2000 : 300);
    return () => window.clearTimeout(timer);
  }, [typed, active, status?.checking]);
  const run = async (kind: typeof working, task: () => Promise<LabStatus>) => {
    setWorking(kind); setError('');
    try { const answer = await task(); setStatus(answer); return answer; } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason)); return null;
    } finally { setWorking(''); }
  };
  const create = () => run('create', () => api<LabStatus>(`${LAB}/install`, 'POST'));
  const use = () => perform(async () => {
    if (await run('use', () => api<LabStatus>(`${LAB}/use`, 'POST', { addresses: typed }))) setTyped({});
  });
  const remove = () => perform(() => run('remove', () => api<LabStatus>(`${LAB}/remove`, 'POST')));
  const commit = (id: string, address: string) => {
    if (!active) { setTyped({ ...typed, [id]: address }); return; }
    perform(async () => {
      const answer = await run('move', () => api<LabStatus>(`${LAB}/address`, 'PUT', { ied_id: id, ip: address }));
      const rest = { ...typedRef.current };
      delete rest[id];
      if (answer) setTyped(rest);
    });
  };
  const edit = (id: string, value: string | undefined) => {
    const next = { ...typed };
    if (value === undefined) delete next[id]; else next[id] = value;
    setTyped(next);
  };
  if (!status) return null;
  const busy = working !== '';
  const attention = status.rows.some(row => !['ok', 'same_range', 'checking'].includes(row.status));
  return <div className={`card lab-card${active ? ' is-active' : ''}`}><div className="guide-section">
    <div className="lab-head">
      <span className="lab-icon"><MonitorSmartphone size={20} aria-hidden /></span>
      <div className="lab-text">
        <h2>{LAB_COPY.title}{!active && <span className="lab-new">New</span>}</h2>
        <p className="muted">{status.installed ? labSummary(status) : LAB_COPY.intro}</p>
      </div>
      {!status.installed && <button className="primary" disabled={busy} onClick={create} title={LAB_COPY.createHint}>
        {working === 'create' ? <SpinIcon /> : <Wrench size={16} />}{LAB_COPY.create}</button>}
      {status.installed && status.problem && <button disabled={busy} onClick={create} title={status.problem}>
        {working === 'create' ? <SpinIcon /> : <Wrench size={16} />}{LAB_COPY.repair}</button>}
      {status.installed && !status.problem && !active && status.rows.length > 0 &&
        <button className="primary" disabled={busy || locked || status.rows.some(blocking)} onClick={use}
          title={locked ? LAB_COPY.stopFirst : LAB_COPY.useHint}>
          {working === 'use' ? <SpinIcon /> : <Check size={16} />}{LAB_COPY.use}</button>}
    </div>
    {working === 'create' && <p className="small muted" style={{ margin: 0 }}>{LAB_COPY.creating}</p>}
    {error && <div className="error-banner" role="alert"><span>{error}</span></div>}
    {status.installed && status.problem && <p className="small" style={{ margin: 0 }}>{status.problem}</p>}
    {status.installed && status.rows.length > 0 && <details className="disclosure" open={attention || undefined}>
      <summary><ChevronRight size={16} aria-hidden />{LAB_COPY.show(status.rows.length)}</summary>
      <div className="disclosure-body lab-body">
        <div className="lab-table-wrap"><table className="lab-table">
          <thead><tr><th>IED</th><th>Address on this PC</th><th>Check</th><th /></tr></thead>
          <tbody>{status.rows.map(row => <LabRowView key={row.ied_id} row={row} typed={typed[row.ied_id]}
            setTyped={value => edit(row.ied_id, value)} commit={address => commit(row.ied_id, address)} busy={busy} />)}</tbody>
        </table></div>
        <p className="small muted" style={{ margin: 0 }}>{LAB_COPY.editHint}
          {status.real_cards.length > 0 && <> Real cards: {status.real_cards.map(realCardLine).join('; ')}.</>}</p>
        <div className="row"><button className="link-button" disabled={busy || locked} onClick={remove}
          title={LAB_COPY.removeHint}>{working === 'remove' && <SpinIcon size={14} />}{LAB_COPY.remove}</button></div>
      </div>
    </details>}
  </div></div>;
}
