import { Laptop, Network, Radio, Undo2 } from 'lucide-react';
import { useState } from 'react';
import { api } from '../../api/client';
import type { Project } from '../../api/types';
import { hint } from '../../copy/hints';
import { TOOLS_COPY } from '../../copy/tools';
import { InfoTip } from '../../ui/InfoTip';
import { CleanupCard } from '../network/CleanupCard';
import { GooseReach } from '../network/GooseReach';
import { LabNetwork } from '../network/LabNetwork';
import { NetworkSetup } from '../network/NetworkSetup';
import { NoPublishers } from '../network/NoPublishers';
import { snmpFollowLine } from '../network/setupWords';
import { useSnmpCount } from '../network/useSnmpCount';

const LAB_CARD = 'ScadaBench Lab';

export function NetworkSettings({ project, locked, perform, openTraffic }: {
  project: Project; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>; openTraffic: () => void;
}) {
  const [wizard, setWizard] = useState(false);
  const lab = project.network.mode === 'lab';
  // The ScadaBench Lab card (ADR-0120) keeps everything on this PC; the stored label names it.
  const labCard = lab && project.network.adapter_label === LAB_CARD;
  const snmp = useSnmpCount(project.revision);
  const publishing = project.ieds.some(ied => ied.role === 'simulated') || snmp > 0;
  if (wizard) return <NetworkSetup close={() => setWizard(false)} locked={locked} perform={perform}
    finished={() => { setWizard(false); perform(async () => {}); }} />;
  return <>
    <div className="mode-banner">
      <div className="row" style={{ alignItems: 'flex-start', gap: 14 }}>
        <span className="guide-empty-icon" style={{ margin: 0, width: 44, height: 44 }}>{lab ? <Network size={22} /> : <Laptop size={22} />}</span>
        <div><h2>{labCard ? <>Your published IEDs are on the ScadaBench Lab network (this PC only)</> : lab ? <>Your published IEDs are on “{project.network.adapter_label}”</> : <>Your published IEDs are reachable from <InfoTip term="loopback" label="this PC only" /></>}</h2>
          <p className="muted" style={{ margin: 0, maxWidth: 620 }}>{hint(labCard ? 'network.mode.labcard' : lab ? 'network.mode.lab' : 'network.mode.loopback')}</p>
          {snmp > 0 && <p className="muted small" style={{ margin: '6px 0 0' }}>{snmpFollowLine(snmp, lab)}</p>}</div>
      </div>
      <div className="row">
        {lab && <button disabled={locked} onClick={() => perform(() => api('/network', 'PUT', { mode: 'loopback' }))}>
          <Undo2 size={16} />Back to this PC only</button>}
        <button className="primary" disabled={locked || !publishing} onClick={() => setWizard(true)}>
          <Network size={16} />{labCard ? 'Change network' : lab ? 'Change network card or addresses' : snmp ? 'Put my devices on a network' : 'Put my IEDs on a network'}</button>
      </div>
      {labCard && publishing && <LabNetwork mode="active" locked={locked} perform={perform} />}
    </div>
    {!publishing && <div className="card"><div className="guide-section">
      <NoPublishers ieds={project.ieds} locked={locked} perform={perform} /></div></div>}
    {locked && <p className="small muted" style={{ margin: 0 }}>{hint('network.locked')}</p>}
    <GooseReach revision={project.revision} perform={perform} />
    <div className="card"><div className="guide-section row" style={{ justifyContent: 'space-between' }}>
      <span className="muted">{TOOLS_COPY.settingsLink}</span>
      <button onClick={openTraffic}><Radio size={16} />{TOOLS_COPY.openTraffic}</button></div></div>
    <CleanupCard locked={locked} revision={project.revision} perform={perform} />
  </>;
}
