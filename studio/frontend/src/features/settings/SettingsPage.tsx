import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { NpcapStatus } from '../../api/network';
import type { Project } from '../../api/types';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { StatusPill } from '../../ui/StatusPill';
import { UI_STATUS } from '../../copy/status';
import { NetworkSettings } from './NetworkSettings';
import { LicenseSection } from './LicenseSection';
import './settings.css';

export function SettingsPage({ project, locked, perform }: {
  project: Project; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [view, setView] = useState<'network' | 'license' | 'about'>('network');
  const [npcap, setNpcap] = useState<NpcapStatus | null>(null);
  useEffect(() => { api<{ npcap: NpcapStatus }>('/network').then(value => setNpcap(value.npcap)).catch(() => {}); }, []);
  const rows: [string, string][] = [
    ['Project', project.name], ['Project format', 'ScadaBench project (.subsim)'],
    ['Session', `${project.session_id.slice(0, 8)} · revision ${project.revision}`],
    ['IEDs', String(project.ieds.length)], ['Equipment on the diagram', String(project.diagram.nodes.length)],
  ];
  return <div className="content">
    <SegmentedControl label="Settings sections" value={view} onChange={setView}
      options={[{ value: 'network', label: 'Network & GOOSE' }, { value: 'license', label: 'License' },
        { value: 'about', label: 'About' }]} />
    {view === 'network' && <NetworkSettings project={project} locked={locked} perform={perform} />}
    {view === 'license' && <LicenseSection />}
    {view === 'about' && <div className="card"><div className="card-heading"><h2>About this app</h2>
      {npcap && <StatusPill status={npcap.installed ? UI_STATUS.npcap_ready : UI_STATUS.npcap_missing} />}</div>
      <table><tbody>{rows.map(([label, value]) => <tr key={label}><td className="muted" style={{ width: 240 }}>{label}</td>
        <td><strong>{value}</strong></td></tr>)}</tbody></table>
      <div className="card-body"><p className="small muted" style={{ margin: 0 }}>ScadaBench is a laboratory tool. Do not connect it
        to an operational substation network. Logs are saved in the data folder next to the app.</p></div></div>}
  </div>;
}
