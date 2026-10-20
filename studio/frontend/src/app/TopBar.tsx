import { Download, FileUp, FolderOpen, Laptop, MoreHorizontal, Network, Pause, Play, Radio, Save, Square } from 'lucide-react';
import type { Project } from '../api/types';
import { Menu } from '../ui/Menu';
import { runStateWords } from '../copy/running';
import { RUN_MARK, runSummary } from '../copy/runShape';
import { REACH_COPY } from '../copy/gooseReach';
import { onLabNetwork } from '../features/network/endpoints';
import { SpinIcon } from '../ui/Spinner';
import { useDelayed } from '../ui/useDelayed';

// A download link, not a page navigation, so the app window's exit prompt does not appear.
function downloadProject() {
  const link = document.createElement('a');
  link.href = '/api/v1/project/download';
  link.download = '';
  link.click();
}

export function TopBar({ title, subtitle, project, busy, working = '', saveState, run, openFile, save, openNetwork }: {
  title: string; subtitle: string; project: Project | null; busy: boolean; working?: string; saveState: string;
  run: (action: 'start' | 'stop' | 'pause') => void; openFile: () => void; save: () => void; openNetwork: () => void;
}) {
  // Words appear only when a task takes longer than a blink, so quick clicks do not flicker.
  const slow = useDelayed(working, 400);
  const runtime = project?.runtime || 'stopped';
  const running = runtime !== 'stopped';
  const active = project?.ieds.filter(ied => ied.role !== 'off').length || 0;
  const inRun = project?.running_ieds?.length ?? active;
  const state = !project ? 'Connecting…' : runtime === 'running' || runtime === 'paused'
    ? runStateWords(inRun, active, runtime === 'paused') : runtime === 'stopped' ? 'Stopped' : runtime;
  // ○ stopped, ◐ some IEDs, ● all IEDs: green only when everything runs (change 0106).
  // The way to start the rest sits on Devices and the SLD, so the top bar keeps its width at 1366 px.
  const { shape } = runSummary(project);
  return <header className="topbar">
    <div className="topbar-title"><h1>{title}</h1><p>{subtitle}</p></div>
    {/* While IEDs run the drawing is locked: the idle "All changes saved" yields its room so the page
        subtitle stays whole at 1366 px; "Saving..." and "Not saved" always show (change 0106). */}
    {!(running && saveState === 'All changes saved') && <span className="small muted" aria-live="polite">{saveState}</span>}
    {project && <ReachChip project={project} open={openNetwork} />}
    {/* Only while running (nothing leaves the PC when stopped). Icon-sized so the page subtitle stays whole
        at 1366 px; the words are in the tooltip, for screen readers, and in Settings (ADR-0073). */}
    {running && project?.network.goose_reach === 'outside' && <button className="goose-pill" title={REACH_COPY.pillHint}
      aria-label={`${REACH_COPY.pill}. ${REACH_COPY.pillHint}`} onClick={openNetwork}><Radio size={16} aria-hidden /></button>}
    <Menu label="Project file" align="right" trigger={<><FileUp size={17} />File</>} items={[
      { label: 'Open file…', icon: <FolderOpen size={16} />, onSelect: openFile, disabled: running || busy },
      { label: 'Save project', icon: <Save size={16} />, onSelect: save, disabled: running || busy || !project },
      { label: 'Download project', icon: <Download size={16} />,
        onSelect: downloadProject, disabled: !project },
    ]} />
    <span className={`run-state shape-${shape}`} role="status">{slow ? <><SpinIcon size={14} />{slow}</>
      : <><span className="run-mark" aria-hidden>{RUN_MARK[shape]}</span>{state}</>}</span>
    {running && <Menu label="Run options" trigger={<MoreHorizontal size={17} />} items={[
      { label: runtime === 'paused' ? 'Resume automation' : 'Pause automation', icon: <Pause size={16} />,
        onSelect: () => run('pause'), disabled: busy },
    ]} />}
    {running
      ? <button className={`run-pill ${shape === 'all' ? 'running' : 'partial'}`} data-tour="run" disabled={busy}
        title="Stop every running IED" onClick={() => run('stop')}>{slow ? <SpinIcon size={15} /> : <Square size={15} />}
        {shape === 'partial' ? 'Stop all' : 'Stop'}</button>
      : <button className="run-pill primary" data-tour="run" disabled={busy || !active} title={active ? 'Start every IED that is not skipped' : 'Add an IED first'}
        onClick={() => run('start')}>{slow ? <SpinIcon size={16} /> : <Play size={16} />}Run</button>}
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
