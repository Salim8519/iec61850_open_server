import { Check, ChevronRight, Wrench } from 'lucide-react';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { api } from '../../api/client';
import { LAB, blocking, type LabStatus } from '../../api/lab';
import { liveFeed } from '../../api/polls';
import { LAB_COPY, labSummary, realCardLine } from '../../copy/labNetwork';
import { SpinIcon } from '../../ui/Spinner';
import { LabRowView } from './LabRows';
import './lab.css';

type Typed = Record<string, string>;

/** The ScadaBench Lab network (ADR-0120, ADR-0121). `choose`: shown in the network card list once "ScadaBench Lab"
 *  is picked: one line, the folded addresses and one button (create, repair or use). `active`: under the banner
 *  while the IEDs use it: the folded addresses, to move one or ping it. The engine decides every address and clash.
 *  `extra` (the list's Cancel) shares the button row. */
export function LabNetwork({ locked, perform, mode, onUsed, extra }: {
  locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>; mode: 'choose' | 'active'; onUsed?: () => void;
  extra?: ReactNode;
}) {
  const [status, setStatus] = useState<LabStatus | null>(null);
  const [typed, setTyped] = useState<Typed>({});
  const [working, setWorking] = useState<'' | 'create' | 'use' | 'move' | 'remove'>('');
  const [error, setError] = useState('');
  const typedRef = useRef(typed);
  typedRef.current = typed;
  const active = !!status?.active;
  // The shared live feed keeps it current; while addresses are typed before "Use", the typed check owns it.
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
  // `onUsed` closes the network card list; it runs after this work so the page reads the project once, in order.
  const use = async () => {
    let used = false;
    await perform(async () => {
      if (await run('use', () => api<LabStatus>(`${LAB}/use`, 'POST', { addresses: typed }))) { setTyped({}); used = true; }
    });
    if (used) onUsed?.();
  };
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
  return <div className="lab-body">
    {mode === 'choose' && <p className="small" style={{ margin: 0 }}>{status.installed ? labSummary(status) : LAB_COPY.intro}</p>}
    {working === 'create' && <p className="small muted" style={{ margin: 0 }}>{LAB_COPY.creating}</p>}
    {error && <div className="error-banner" role="alert"><span>{error}</span></div>}
    {status.installed && status.problem && <p className="small" style={{ margin: 0 }}>{status.problem}</p>}
    {status.installed && !status.problem && status.service_update && <p className="small" style={{ margin: 0 }}>
      {LAB_COPY.serviceUpdate}{' '}<button className="link-button" disabled={busy || locked} onClick={create}
        title={locked ? LAB_COPY.serviceUpdateStopFirst : LAB_COPY.serviceUpdateHint}>
        {working === 'create' && <SpinIcon size={14} />}{LAB_COPY.repair}</button></p>}
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
    {mode === 'choose' && <div className="row">
      {!status.installed && <button className="primary" disabled={busy} onClick={create} title={LAB_COPY.createHint}>
        {working === 'create' ? <SpinIcon /> : <Wrench size={16} />}{LAB_COPY.create}</button>}
      {status.installed && status.problem && <button className="primary" disabled={busy} onClick={create} title={status.problem}>
        {working === 'create' ? <SpinIcon /> : <Wrench size={16} />}{LAB_COPY.repair}</button>}
      {status.installed && !status.problem && !active && status.rows.length > 0 &&
        <button className="primary" disabled={busy || locked || status.rows.some(blocking)} onClick={use}
          title={locked ? LAB_COPY.stopFirst : LAB_COPY.useHint}>
          {working === 'use' ? <SpinIcon /> : <Check size={16} />}{LAB_COPY.use}</button>}
      {extra}
      {!status.installed && <span className="small muted">{LAB_COPY.onceHint}</span>}
    </div>}
  </div>;
}
