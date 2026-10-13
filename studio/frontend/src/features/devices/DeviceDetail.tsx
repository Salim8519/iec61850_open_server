import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { LiveSession } from '../../api/network';
import type { Ied, IedRole, Project } from '../../api/types';
import { hint } from '../../copy/hints';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { ComparisonView } from './ComparisonView';
import { DeviceFiles } from './DeviceFiles';
import { DeviceOverview } from './DeviceOverview';
import { DeviceSignals } from './DeviceSignals';
import { LiveValueTable } from './LiveValueTable';

type Tab = 'overview' | 'signals' | 'compare' | 'files';

export function DeviceDetail({ ied, project, locked, values, perform, prepare }: {
  ied: Ied; project: Project; locked: boolean; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const [tab, setTab] = useState<Tab>('overview');
  const [session, setSession] = useState<LiveSession | undefined>();
  const running = project.runtime !== 'stopped';
  useEffect(() => { if (tab === 'compare' && ied.role !== 'real') setTab('overview'); }, [ied.role, tab]);
  useEffect(() => {
    setSession(undefined);
    if (!running || ied.role !== 'real') return;
    let cancelled = false;
    const load = () => api<{ sessions: LiveSession[] }>('/live').then(value => {
      if (!cancelled) setSession(value.sessions.find(item => item.ied_id === ied.id));
    }).catch(() => {});
    load();
    const timer = setInterval(load, 2000);
    return () => { cancelled = true; clearInterval(timer); };
  }, [running, ied.id, ied.role]);
  const tabs: { id: Tab; label: string }[] = [{ id: 'overview', label: 'Overview' }, { id: 'signals', label: 'Data points' },
    ...(ied.role === 'real' ? [{ id: 'compare' as const, label: 'Compare with SCD' }] : []), { id: 'files', label: 'Files' }];
  return <div className="card">
    <div className="detail-head">
      <div><h2>{ied.name}</h2><p className="muted" style={{ margin: 0 }}>{hint(`live.role.${ied.role}`)}</p></div>
      <SegmentedControl<IedRole> label="What Studio does with this device" value={ied.role} disabled={locked}
        onChange={role => perform(() => api(`/ieds/${ied.id}/settings`, 'PUT', { role }))} options={[
          { value: 'simulated', label: 'Simulate' }, { value: 'real', label: 'Real device' }, { value: 'off', label: 'Off' }]} />
    </div>
    <nav className="tabbar" aria-label="Device sections">{tabs.map(item => <button key={item.id}
      className={tab === item.id ? 'active' : ''} aria-current={tab === item.id ? 'page' : undefined}
      onClick={() => setTab(item.id)}>{item.label}</button>)}</nav>
    {tab === 'overview' && <DeviceOverview ied={ied} project={project} locked={locked} session={session} perform={perform} />}
    {tab === 'signals' && (ied.role === 'real'
      ? <LiveValueTable iedId={ied.id} live={session?.state === 'live'} values={values} />
      : <DeviceSignals ied={ied} project={project} values={values} perform={perform} prepare={prepare} />)}
    {tab === 'compare' && <ComparisonView iedId={ied.id} ready={session?.state === 'live'} />}
    {tab === 'files' && <DeviceFiles ied={ied} locked={locked} perform={perform} prepare={prepare} />}
  </div>;
}
