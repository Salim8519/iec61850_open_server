import { Cpu, HeartPulse, LayoutTemplate, Share2, Upload } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { Project } from '../../api/types';
import { EmptyState } from '../../ui/EmptyState';
import { StatusPill } from '../../ui/StatusPill';
import { roleStatus } from '../../copy/status';
import { deviceBadge, modeStatus } from '../../copy/testMode';
import { useModes } from './useModes';
import { DeviceDetail } from './DeviceDetail';
import { HealthView } from './HealthView';
import { SignalFlowView } from './SignalFlowView';
import { GooseHint } from './GooseHint';
import { useGooseLinks } from './useGooseLinks';

const FLOW = 'station-flow';
const HEALTH = 'station-health';

export function DevicesPage({ project, locked, values, perform, prepare, importFile, addTemplate, flowIed }: {
  project: Project; locked: boolean; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
  importFile: () => void; addTemplate: (name: string) => void; flowIed?: string;
}) {
  const [selected, setSelected] = useState(flowIed ?? '');
  useEffect(() => {
    if (selected !== FLOW && selected !== HEALTH && !project.ieds.some(item => item.id === selected)) setSelected(project.ieds[0]?.id || '');
  }, [project.ieds, selected]);
  useEffect(() => { if (flowIed) setSelected(flowIed); }, [flowIed]);
  const ied = project.ieds.find(item => item.id === selected);
  const { modes } = useModes(project.runtime !== 'stopped', project.revision);
  const gooseLinks = useGooseLinks(project.revision, project.runtime !== 'stopped');
  if (!project.ieds.length) return <div className="content"><div className="card"><EmptyState icon={<Cpu size={26} />}
    title="Import your first IED" hintId="devices.empty" action={<div className="row" style={{ justifyContent: 'center' }}>
      <button className="primary large" data-tour="devices-import" disabled={locked} onClick={importFile}><Upload size={18} />Import CID / SCD file</button>
      <button className="large" disabled={locked} onClick={() => addTemplate('Feeder01')}><LayoutTemplate size={18} />Use a sample IED</button>
    </div>} /></div></div>;
  return <div className="content">
    <div className="master-detail">
      <div className="stack" style={{ gap: 12 }}>
        <button className="primary large" data-tour="devices-import" disabled={locked} onClick={importFile}><Upload size={18} />Import CID / SCD</button>
        <div className="list" role="list">{project.ieds.map(item => <button key={item.id} role="listitem"
          className={`list-row${item.id === selected ? ' active' : ''}`} onClick={() => setSelected(item.id)}>
          <Cpu size={22} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>{item.name}</strong><small>{item.signal_count.toLocaleString()} data points</small>
            <GooseHint map={gooseLinks} name={item.name} /></span>
          <span className="stack" style={{ gap: 4, alignItems: 'flex-end' }}><StatusPill status={roleStatus(item.role, item.state)} />
            {deviceBadge(modes.ieds[item.id]) && <StatusPill status={modeStatus(deviceBadge(modes.ieds[item.id])!)} />}</span></button>)}</div>
        <button className="ghost" disabled={locked} onClick={() => addTemplate(`Sample${project.ieds.length + 1}`)}>
          <LayoutTemplate size={16} />Add a sample IED</button>
        <div className="section-label">Whole station</div>
        <button className={`list-row${selected === HEALTH ? ' active' : ''}`} onClick={() => setSelected(HEALTH)}>
          <HeartPulse size={22} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>Health check</strong><small>Does the real station match the SCD?</small></span></button>
        <button className={`list-row${selected === FLOW ? ' active' : ''}`} onClick={() => setSelected(FLOW)}>
          <Share2 size={22} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>Signal flow</strong><small>Who sends GOOSE and reports to whom</small></span></button>
      </div>
      {selected === FLOW && <SignalFlowView running={project.runtime !== 'stopped'} />}
      {selected === HEALTH && <HealthView running={project.runtime !== 'stopped'} />}
      {ied && <DeviceDetail ied={ied} project={project} locked={locked} values={values} perform={perform} prepare={prepare}
        initialTab={ied.id === flowIed ? 'flow' : undefined} mode={modes.ieds[ied.id]} selectDevice={setSelected} />}
    </div>
  </div>;
}
