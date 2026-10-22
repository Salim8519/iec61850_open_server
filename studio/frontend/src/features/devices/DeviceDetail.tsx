import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import { useLiveFeed } from '../../api/useLiveFeed';
import type { LiveSession } from '../../api/network';
import type { Ied, IedRole, Project } from '../../api/types';
import { ROLE_COPY, ROLE_HELP, ROLE_ORDER } from '../../copy/roles';
import { DEVICES_COPY } from '../../copy/devices';
import { HelpTip } from '../../ui/HelpTip';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { TabBar } from '../../ui/TabBar';
import { FullView, FullViewButton } from '../../ui/FullView';
import { useFullView } from '../../state/useFullView';
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
import { DeleteIed } from './DeleteIed';

type Tab = 'overview' | 'signals' | 'compare' | 'flow' | 'files';

export function DeviceDetail({ ied, project, locked, busy = false, perform, prepare, initialTab, mode, selectDevice, openNetwork }: {
  ied: Ied; project: Project; locked: boolean; busy?: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>; initialTab?: Tab;
  mode?: ModeSummary; selectDevice?: (iedId: string) => void; openNetwork?: () => void;
}) {
  const [tab, setTab] = useState<Tab>(initialTab ?? 'overview');
  const [full, setFull] = useFullView('devices');
  useEffect(() => { if (initialTab) setTab(initialTab); }, [initialTab, ied.id]);
  const running = project.runtime !== 'stopped';
  useEffect(() => { if (tab === 'compare' && ied.role !== 'real') setTab('overview'); }, [ied.role, tab]);
  const live = useLiveFeed<{ sessions: LiveSession[] }>(running && ied.role === 'real' ? '/live' : null, 2000).value;
  const session = live?.sessions.find(item => item.ied_id === ied.id);
  const tabs: { id: Tab; label: string }[] = [{ id: 'overview', label: 'Overview' }, { id: 'signals', label: 'Data points' },
    ...(ied.role === 'real' ? [{ id: 'compare' as const, label: 'Compare with SCD' }] : []),
    { id: 'flow', label: 'Signal flow' }, { id: 'files', label: 'Files' }];
  const notes = noteCount(ied);
  const facts = [`${ied.signal_count.toLocaleString()} points`, `${ied.controls.length.toLocaleString()} controls`,
    ...(notes ? [`${notes} ${notes === 1 ? 'note' : 'notes'}`] : [])].join(' · ');
  return <FullView full={full} change={setFull} place={`Devices › ${ied.name}`} detail={facts}
    status={<RunThisIed ied={ied} project={project} busy={busy} perform={perform} />}>
  <div className="card fill-panel device-detail">
    {/* In full view the dark bar already names the device and holds Run (ADR-0095). */}
    {!full && <div className="detail-head compact">
      <h2>{ied.name}</h2>
      <span className="small muted detail-facts">{facts}</span>
      <HelpTip text={ROLE_HELP} label="Which one should I choose?" />
      <span className="gateway-spacer" />
      <RunThisIed ied={ied} project={project} busy={busy} perform={perform} />
      <SegmentedControl<IedRole> label="What ScadaBench does with this device" value={ied.role} disabled={locked}
        onChange={role => perform(() => api(`/ieds/${ied.id}/settings`, 'PUT', { role }))} options={[
          ...ROLE_ORDER.map(role => ({ value: role, label: ROLE_COPY[role].label,
            title: locked ? DEVICES_COPY.roleLocked : undefined }))]} />
      <DeleteIed compact iedId={ied.id} iedName={ied.name} disabled={locked} perform={perform} prepare={prepare} />
    </div>}
    {!full && <RoleLine ied={ied} network={project.network} openNetwork={openNetwork} />}
    <TabBar label="Device sections" value={tab} change={setTab} tabs={tabs.map(item => [item.id, item.label])}
      end={<FullViewButton full={full} change={setFull} />} />
    <div className="panel-body">
      {tab === 'overview' && <div className="fill-scroll">
        <DeviceOverview ied={ied} project={project} locked={locked} session={session} perform={perform} mode={mode}
          selectDevice={selectDevice} /></div>}
      {tab === 'signals' && (ied.role === 'real'
        ? <LiveValueTable iedId={ied.id} live={session?.state === 'live'} controls={ied.controls} />
        : <DeviceSignals ied={ied} project={project} perform={perform} prepare={prepare} />)}
      {tab === 'compare' && <ComparisonView iedId={ied.id} ready={session?.state === 'live'} />}
      {tab === 'flow' && <div className="fill-scroll"><SignalFlowView iedName={ied.name} running={running} /></div>}
      {tab === 'files' && <div className="fill-scroll"><DeviceFiles ied={ied} locked={locked} perform={perform} prepare={prepare} /></div>}
    </div>
  </div>
  </FullView>;
}
