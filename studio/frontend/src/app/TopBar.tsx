import { Download, FileUp, FolderOpen, Laptop, MoreHorizontal, Network, Pause, Play, Save, Square } from 'lucide-react';
import type { Project } from '../api/types';
import { Menu } from '../ui/Menu';
import { runStateWords } from '../copy/running';
import { onLabNetwork } from '../features/network/endpoints';

// A download link, not a page navigation, so the app window's exit prompt does not appear.
function downloadProject() {
  const link = document.createElement('a');
  link.href = '/api/v1/project/download';
  link.download = '';
  link.click();
}

export function TopBar({ title, subtitle, project, busy, saveState, run, openFile, save, openNetwork }: {
  title: string; subtitle: string; project: Project | null; busy: boolean; saveState: string;
  run: (action: 'start' | 'stop' | 'pause') => void; openFile: () => void; save: () => void; openNetwork: () => void;
}) {
  const runtime = project?.runtime || 'stopped';
  const running = runtime !== 'stopped';
  const active = project?.ieds.filter(ied => ied.role !== 'off').length || 0;
  const inRun = project?.running_ieds?.length ?? active;
  const state = !project ? 'Connecting…' : runtime === 'running' || runtime === 'paused'
    ? runStateWords(inRun, active, runtime === 'paused') : runtime === 'stopped' ? 'Stopped' : runtime;
  return <header className="topbar">
    <div className="topbar-title"><h1>{title}</h1><p>{subtitle}</p></div>
    <span className="small muted" aria-live="polite">{saveState}</span>
    {project && <ReachChip project={project} open={openNetwork} />}
    <Menu label="Project file" align="right" trigger={<><FileUp size={17} />File</>} items={[
      { label: 'Open file…', icon: <FolderOpen size={16} />, onSelect: openFile, disabled: running || busy },
      { label: 'Save project', icon: <Save size={16} />, onSelect: save, disabled: running || busy || !project },
      { label: 'Download project', icon: <Download size={16} />,
        onSelect: downloadProject, disabled: !project },
    ]} />
    <span className="run-state" role="status"><span className={`dot${running ? ' on' : ''}`} />{state}</span>
    {running && <Menu label="Run options" trigger={<MoreHorizontal size={17} />} items={[
      { label: runtime === 'paused' ? 'Resume automation' : 'Pause automation', icon: <Pause size={16} />,
        onSelect: () => run('pause'), disabled: busy },
    ]} />}
    {running
      ? <button className="run-pill running" data-tour="run" disabled={busy} onClick={() => run('stop')}><Square size={15} />Stop</button>
      : <button className="run-pill primary" data-tour="run" disabled={busy || !active} title={active ? 'Start every IED that is not left out' : 'Add an IED first'}
        onClick={() => run('start')}><Play size={16} />Run</button>}
  </header>;
}

/** Who can reach the IEDs ScadaBench publishes as (change 0098); opens the network setup. */
function ReachChip({ project, open }: { project: Project; open: () => void }) {
  const published = project.ieds.filter(ied => ied.role === 'simulated');
  if (!published.length) return null;
  const lab = project.network.mode === 'lab';
  // Only IEDs with a lab address are on the adapter; one added later answers on this PC only (change 0102).
  const onNetwork = published.filter(ied => onLabNetwork(ied, project.network)).length;
  const count = lab && onNetwork < published.length ? `${onNetwork} of ${published.length}` : String(published.length);
  return <button className="reach-chip" onClick={open}
    title={lab ? `Other PCs and devices on this adapter can reach ${onNetwork < published.length
      ? `${onNetwork} of your IEDs; the others answer on this PC only until you add them to the network`
      : 'your IEDs'}. Click to change.` : 'Only programs on this PC can reach your IEDs. Click to put them on a network.'}>
    {lab ? <Network size={15} aria-hidden /> : <Laptop size={15} aria-hidden />}
    {count} {published.length === 1 ? 'IED' : 'IEDs'} reachable from {lab ? <>“{project.network.adapter_label}”</> : 'this PC only'}
  </button>;
}
