import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { LiveSession } from '../../api/network';
import type { Ied, IedRole, Project } from '../../api/types';
import { hint } from '../../copy/hints';
import { HelpTip } from '../../ui/HelpTip';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { TabBar } from '../../ui/TabBar';
import { noteCount } from './deviceFilters';
import { ComparisonView } from './ComparisonView';
import { DeviceFiles } from './DeviceFiles';
import { DeviceOverview } from './DeviceOverview';
import { DeviceSignals } from './DeviceSignals';
import { LiveValueTable } from './LiveValueTable';
import { SignalFlowView } from './SignalFlowView';
import type { ModeSummary } from './useModes';
import { RoleLine } from './RoleLine';
import { RunThisIed } from './RunThisIed';

type Tab = 'overview' | 'signals' | 'compare' | 'flow' | 'files';

export function DeviceDetail({ ied, project, locked, busy = false, values, perform, prepare, initialTab, mode, selectDevice, openNetwork }: {
  ied: Ied; project: Project; locked: boolean; busy?: boolean; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>; initialTab?: Tab;
  mode?: ModeSummary; selectDevice?: (iedId: string) => void; openNetwork?: () => void;
}) {
  const [tab, setTab] = useState<Tab>(initialTab ?? 'overview');
  useEffect(() => { if (initialTab) setTab(initialTab); }, [initialTab, ied.id]);
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
    ...(ied.role === 'real' ? [{ id: 'compare' as const, label: 'Compare with SCD' }] : []),
    { id: 'flow', label: 'Signal flow' }, { id: 'files', label: 'Files' }];
  const notes = noteCount(ied);
  const facts = [`${ied.signal_count.toLocaleString()} points`, `${ied.controls.length.toLocaleString()} controls`,
    ...(notes ? [`${notes} ${notes === 1 ? 'note' : 'notes'}`] : [])].join(' · ');
  return <div className="card fill-panel device-detail">
    <div className="detail-head compact">
      <h2>{ied.name}</h2>
      <span className="small muted detail-facts">{facts}</span>
      <HelpTip text={hint(`live.role.${ied.role}`)} label="What does this mode mean?" />
      <span className="gateway-spacer" />
      <RunThisIed ied={ied} project={project} busy={busy} perform={perform} />
      <SegmentedControl<IedRole> label="What ScadaBench does with this device" value={ied.role} disabled={locked}
        onChange={role => perform(() => api(`/ieds/${ied.id}/settings`, 'PUT', { role }))} options={[
          { value: 'simulated', label: 'Be this IED' }, { value: 'real', label: 'Read real IED' }, { value: 'off', label: 'Leave out' }]} />
    </div>
    <RoleLine ied={ied} network={project.network} openNetwork={openNetwork} />
    <TabBar label="Device sections" value={tab} change={setTab} tabs={tabs.map(item => [item.id, item.label])} />
    <div className="panel-body">
      {tab === 'overview' && <div className="fill-scroll">
        <DeviceOverview ied={ied} project={project} locked={locked} session={session} perform={perform} mode={mode}
          selectDevice={selectDevice} /></div>}
      {tab === 'signals' && (ied.role === 'real'
        ? <LiveValueTable iedId={ied.id} live={session?.state === 'live'} values={values} />
        : <DeviceSignals ied={ied} project={project} values={values} perform={perform} prepare={prepare} />)}
      {tab === 'compare' && <ComparisonView iedId={ied.id} ready={session?.state === 'live'} />}
      {tab === 'flow' && <div className="fill-scroll"><SignalFlowView iedName={ied.name} running={running} /></div>}
      {tab === 'files' && <div className="fill-scroll"><DeviceFiles ied={ied} locked={locked} perform={perform} prepare={prepare} /></div>}
    </div>
  </div>;
}
