import { Cpu, LayoutTemplate, Upload } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { Project } from '../../api/types';
import { EmptyState } from '../../ui/EmptyState';
import { StatusPill } from '../../ui/StatusPill';
import { roleStatus } from '../../copy/status';
import { DeviceDetail } from './DeviceDetail';

export function DevicesPage({ project, locked, values, perform, prepare, importFile, addTemplate }: {
  project: Project; locked: boolean; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
  importFile: () => void; addTemplate: (name: string) => void;
}) {
  const [selected, setSelected] = useState('');
  useEffect(() => {
    if (!project.ieds.some(item => item.id === selected)) setSelected(project.ieds[0]?.id || '');
  }, [project.ieds, selected]);
  const ied = project.ieds.find(item => item.id === selected);
  if (!project.ieds.length) return <div className="content"><div className="card"><EmptyState icon={<Cpu size={26} />}
    title="Import your first IED" hintId="devices.empty" action={<div className="row" style={{ justifyContent: 'center' }}>
      <button className="primary large" disabled={locked} onClick={importFile}><Upload size={18} />Import CID / SCD file</button>
      <button className="large" disabled={locked} onClick={() => addTemplate('Feeder01')}><LayoutTemplate size={18} />Use a sample IED</button>
    </div>} /></div></div>;
  return <div className="content">
    <div className="master-detail">
      <div className="stack" style={{ gap: 12 }}>
        <button className="primary large" disabled={locked} onClick={importFile}><Upload size={18} />Import CID / SCD</button>
        <div className="list" role="list">{project.ieds.map(item => <button key={item.id} role="listitem"
          className={`list-row${item.id === selected ? ' active' : ''}`} onClick={() => setSelected(item.id)}>
          <Cpu size={22} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>{item.name}</strong><small>{item.signal_count.toLocaleString()} data points</small></span>
          <StatusPill status={roleStatus(item.role, item.state)} /></button>)}</div>
        <button className="ghost" disabled={locked} onClick={() => addTemplate(`Sample${project.ieds.length + 1}`)}>
          <LayoutTemplate size={16} />Add a sample IED</button>
      </div>
      {ied && <DeviceDetail ied={ied} project={project} locked={locked} values={values} perform={perform} prepare={prepare} />}
    </div>
  </div>;
}
