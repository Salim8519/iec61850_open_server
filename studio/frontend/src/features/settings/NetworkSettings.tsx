import { Laptop, Network, Radio, Undo2 } from 'lucide-react';
import { useState } from 'react';
import { api } from '../../api/client';
import type { Project } from '../../api/types';
import { hint } from '../../copy/hints';
import { TOOLS_COPY } from '../../copy/tools';
import { InfoTip } from '../../ui/InfoTip';
import { CleanupCard } from '../network/CleanupCard';
import { LabWizard } from '../network/LabWizard';

export function NetworkSettings({ project, locked, perform, openTraffic }: {
  project: Project; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>; openTraffic: () => void;
}) {
  const [wizard, setWizard] = useState(false);
  const lab = project.network.mode === 'lab';
  if (wizard) return <LabWizard close={() => setWizard(false)} finished={() => { setWizard(false); perform(async () => {}); }} />;
  return <>
    <div className="mode-banner">
      <div className="row" style={{ alignItems: 'flex-start', gap: 14 }}>
        <span className="guide-empty-icon" style={{ margin: 0, width: 44, height: 44 }}>{lab ? <Network size={22} /> : <Laptop size={22} />}</span>
        <div><h2>{lab ? <>Lab network on “{project.network.adapter_label}”</> : <InfoTip term="loopback" label="This PC only" />}</h2>
          <p className="muted" style={{ margin: 0, maxWidth: 620 }}>{hint(lab ? 'network.mode.lab' : 'network.mode.loopback')}</p></div>
      </div>
      <div className="row">
        {lab && <button disabled={locked} onClick={() => perform(() => api('/network', 'PUT', { mode: 'loopback' }))}>
          <Undo2 size={16} />Back to this PC only</button>}
        <button className="primary" disabled={locked} onClick={() => setWizard(true)}>
          <Network size={16} />{lab ? 'Change setup' : 'Set up lab network'}</button>
      </div>
    </div>
    {locked && <p className="small muted" style={{ margin: 0 }}>{hint('network.locked')}</p>}
    <div className="card"><div className="guide-section row" style={{ justifyContent: 'space-between' }}>
      <span className="muted">{TOOLS_COPY.settingsLink}</span>
      <button onClick={openTraffic}><Radio size={16} />{TOOLS_COPY.openTraffic}</button></div></div>
    <CleanupCard locked={locked} revision={project.revision} perform={perform} />
  </>;
}
