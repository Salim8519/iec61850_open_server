import { Cpu, HeartPulse, LayoutTemplate, ListChecks, Share2, Upload } from 'lucide-react';
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
import { GooseHint } from './GooseHint';
import { useGooseLinks } from './useGooseLinks';
import { DeleteIedsDialog } from './DeleteIedsDialog';
import { DeviceSelectBar } from './DeviceSelectBar';
import { DeviceRunSummary } from './DeviceRunSummary';
import { runGroups } from './runGroups';
import { runSummary } from '../../copy/runShape';
import { DEVICES_COPY } from '../../copy/devices';
import { freeSampleName, shownSelection } from './devicePick';

const FLOW = 'station-flow';
const HEALTH = 'station-health';

/** Devices: the IED list in its own scrolling column, the chosen IED (or whole-station view) beside it,
 *  both filling the window (ADR-0064). */
export function DevicesPage({ project, locked, perform, prepare, importFile, addTemplate, flowIed, openNetwork }: {
  project: Project; locked: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
  importFile: () => void; addTemplate: (name: string) => void; flowIed?: string; openNetwork?: () => void;
}) {
  const [selected, setSelected] = useState(flowIed ?? '');
  const filter = useListFilter('devices-list', DEVICE_FILTER);
  // Choosing several IEDs to delete: null when not choosing (change 0105).
  const [picked, setPicked] = useState<string[] | null>(null);
  const [deleting, setDeleting] = useState<string[] | null>(null);
  useEffect(() => { if (locked) setPicked(null); }, [locked]);
  const toggle = (id: string) => setPicked(current => current && (current.includes(id)
    ? current.filter(item => item !== id) : [...current, id]));
  useEffect(() => {
    if (selected !== FLOW && selected !== HEALTH && !project.ieds.some(item => item.id === selected)) setSelected(project.ieds[0]?.id || '');
  }, [project.ieds, selected]);
  useEffect(() => { if (flowIed) setSelected(flowIed); }, [flowIed]);
  // The detail never shows an IED the list filter hides, for example after a reload (change 0177).
  useEffect(() => {
    if (selected === FLOW || selected === HEALTH) return;
    const next = shownSelection(selected, project.ieds.filter(item => deviceMatches(item, filter.values)).map(item => item.id), flowIed);
    if (next !== selected) setSelected(next);
  }, [project.ieds, filter.values, selected, flowIed]);
  const ied = project.ieds.find(item => item.id === selected);
  const { modes } = useModes(project.runtime !== 'stopped', project.revision);
  const gooseLinks = useGooseLinks(project.revision, project.runtime !== 'stopped');
  if (!project.ieds.length) return <div className="content"><div className="card"><EmptyState icon={<Cpu size={26} />}
    title="Import your first IED" hintId="devices.empty" action={<div className="row" style={{ justifyContent: 'center' }}>
      <button className="primary large" data-tour="devices-import" disabled={locked} onClick={importFile}
        title={locked ? DEVICES_COPY.importLocked : undefined}><Upload size={18} />Import CID / SCD file</button>
      <button className="large" disabled={locked} title={locked ? DEVICES_COPY.sampleLocked : undefined}
        onClick={() => addTemplate('Feeder01')}><LayoutTemplate size={18} />Use a sample IED</button>
    </div>} /></div></div>;
  const shown = project.ieds.filter(item => deviceMatches(item, filter.values));
  const run = runSummary(project);
  const live = new Set(run.shape === 'partial' ? run.running.map(item => item.id) : []);
  return <div className="content fill-page devices-page">
    <DeviceRunSummary project={project} perform={perform} />
    <div className="fill-body fill-split devices-split">
      <aside className="card side-pane devices-list" aria-label="IEDs">
        <div className="devices-list-head">
          <button className="primary large" data-tour="devices-import" disabled={locked} onClick={importFile}
            title={locked ? DEVICES_COPY.importLocked : undefined}><Upload size={18} />
            <span>Import <span style={{ whiteSpace: 'nowrap' }}>CID / SCD</span></span></button>
          <button className={picked ? 'large active-toggle' : 'large'} disabled={locked} aria-pressed={!!picked}
            title={locked ? 'Stop the run to delete IEDs' : 'Choose several IEDs to delete'}
            onClick={() => setPicked(picked ? null : [])}><ListChecks size={18} />Select</button>
        </div>
        <FilterBar label="Filter IEDs" spec={DEVICE_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
          shown={shown.length} total={project.ieds.length} counts={{ role: roleCounts(project.ieds) }} />
        <div className="list" role="list">{runGroups(shown, run).map(group => [
          group.label && <div key={group.label} className="section-label">{group.label}</div>,
          ...group.ieds.map(item => <DeviceListRow key={item.id} ied={item} live={live.has(item.id)}
            active={item.id === selected} mode={modes.ieds[item.id]} checked={picked ? picked.includes(item.id) : undefined}
            select={() => picked ? toggle(item.id) : setSelected(item.id)}
            hints={<GooseHint map={gooseLinks} name={item.name} />} />)])}
          {shown.length === 0 && <p className="small muted" style={{ margin: 0 }}>No IED matches. Press Clear to see them all.</p>}</div>
        <button className="ghost" disabled={locked} title={locked ? DEVICES_COPY.sampleLocked : undefined}
          onClick={() => addTemplate(freeSampleName(project.ieds))}>
          <LayoutTemplate size={16} />Add a sample IED</button>
        <div className="section-label">Whole station</div>
        <button className={`list-row compact${selected === HEALTH ? ' active' : ''}`} onClick={() => setSelected(HEALTH)}>
          <HeartPulse size={18} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>Health check</strong><small>Does the real station match the SCD?</small></span></button>
        <button className={`list-row compact${selected === FLOW ? ' active' : ''}`} onClick={() => setSelected(FLOW)}>
          <Share2 size={18} style={{ color: 'var(--text-2)', flexShrink: 0 }} />
          <span className="list-text"><strong>Signal flow</strong><small>Who sends GOOSE and reports to whom</small></span></button>
        {picked && <DeviceSelectBar count={picked.length} shown={shown.length} cancel={() => setPicked(null)}
          selectShown={() => setPicked(shown.map(item => item.id))} remove={() => setDeleting(picked)} />}
      </aside>
      {selected === FLOW && <div className="fill-scroll"><SignalFlowView running={project.runtime !== 'stopped'} /></div>}
      {selected === HEALTH && <div className="fill-scroll"><HealthView running={project.runtime !== 'stopped'} /></div>}
      {ied && <DeviceDetail ied={ied} project={project} locked={locked} perform={perform} prepare={prepare}
        initialTab={ied.id === flowIed ? 'flow' : undefined} mode={modes.ieds[ied.id]} selectDevice={setSelected}
        openNetwork={openNetwork} />}
    </div>
    {deleting && <DeleteIedsDialog ids={deleting} close={() => setDeleting(null)} done={() => setPicked(null)}
      perform={perform} prepare={prepare} />}
  </div>;
}
