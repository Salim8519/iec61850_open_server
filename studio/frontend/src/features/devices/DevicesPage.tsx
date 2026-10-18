import { Cpu, HeartPulse, LayoutTemplate, Share2, Upload } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { Project } from '../../api/types';
import { useListFilter } from '../../state/useListFilter';
import { EmptyState } from '../../ui/EmptyState';
import { FilterBar } from '../../ui/FilterBar';
import { useModes } from './useModes';
import { DeviceDetail } from './DeviceDetail';
import { DeviceListRow } from './DeviceListRow';
import { DEVICE_FILTER, deviceMatches, roleCounts } from './deviceFilters';
import { HealthView } from './HealthView';
import { SignalFlowView } from './SignalFlowView';
import './devices.css';

const FLOW = 'station-flow';
const HEALTH = 'station-health';

/** Devices: the IED list in its own scrolling column, the chosen IED (or whole-station view) beside it,
 *  both filling the window (ADR-0064). */
export function DevicesPage({ project, locked, values, perform, prepare, importFile, addTemplate, flowIed }: {
  project: Project; locked: boolean; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
  importFile: () => void; addTemplate: (name: string) => void; flowIed?: string;
}) {
  const [selected, setSelected] = useState(flowIed ?? '');
  const filter = useListFilter('devices-list', DEVICE_FILTER);
  useEffect(() => {
    if (selected !== FLOW && selected !== HEALTH && !project.ieds.some(item => item.id === selected)) setSelected(project.ieds[0]?.id || '');
  }, [project.ieds, selected]);
  useEffect(() => { if (flowIed) setSelected(flowIed); }, [flowIed]);
  const ied = project.ieds.find(item => item.id === selected);
  const { modes } = useModes(project.runtime !== 'stopped', project.revision);
  if (!project.ieds.length) return <div className="content"><div className="card"><EmptyState icon={<Cpu size={26} />}
    title="Import your first IED" hintId="devices.empty" action={<div className="row" style={{ justifyContent: 'center' }}>
      <button className="primary large" data-tour="devices-import" disabled={locked} onClick={importFile}><Upload size={18} />Import CID / SCD file</button>
      <button className="large" disabled={locked} onClick={() => addTemplate('Feeder01')}><LayoutTemplate size={18} />Use a sample IED</button>
    </div>} /></div></div>;
  const shown = project.ieds.filter(item => deviceMatches(item, filter.values));
  return <div className="content fill-page devices-page">
    <div className="fill-body fill-split devices-split">
      <aside className="card side-pane devices-list" aria-label="IEDs">
        <button className="primary large" data-tour="devices-import" disabled={locked} onClick={importFile}><Upload size={18} />Import CID / SCD</button>
        <FilterBar label="Filter IEDs" spec={DEVICE_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
          shown={shown.length} total={project.ieds.length} counts={{ role: roleCounts(project.ieds) }} />
        <div className="list" role="list">{shown.map(item => <DeviceListRow key={item.id} ied={item}
          active={item.id === selected} mode={modes.ieds[item.id]} select={() => setSelected(item.id)} />)}
          {shown.length === 0 && <p className="small muted" style={{ margin: 0 }}>No IED matches. Press Clear to see them all.</p>}</div>
        <button className="ghost" disabled={locked} onClick={() => addTemplate(`Sample${project.ieds.length + 1}`)}>
          <LayoutTemplate size={16} />Add a sample IED</button>
        <div className="section-label">Whole station</div>
        <button className={`list-row compact${selected === HEALTH ? ' active' : ''}`} onClick={() => setSelected(HEALTH)}>
          <HeartPulse size={18} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>Health check</strong><small>Does the real station match the SCD?</small></span></button>
        <button className={`list-row compact${selected === FLOW ? ' active' : ''}`} onClick={() => setSelected(FLOW)}>
          <Share2 size={18} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>Signal flow</strong><small>Who sends GOOSE and reports to whom</small></span></button>
      </aside>
      {selected === FLOW && <div className="fill-scroll"><SignalFlowView running={project.runtime !== 'stopped'} /></div>}
      {selected === HEALTH && <div className="fill-scroll"><HealthView running={project.runtime !== 'stopped'} /></div>}
      {ied && <DeviceDetail ied={ied} project={project} locked={locked} values={values} perform={perform} prepare={prepare}
        initialTab={ied.id === flowIed ? 'flow' : undefined} mode={modes.ieds[ied.id]} />}
    </div>
  </div>;
}
