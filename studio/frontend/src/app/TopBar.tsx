import { FilePlus, FileUp, FileX, FolderOpen, Laptop, MoreHorizontal, Network, Pause, Play, Radio, Redo2, Save, SaveAll, Square, ToggleLeft, ToggleRight, Undo2 } from 'lucide-react';
import type { Project } from '../api/types';
import { Menu } from '../ui/Menu';
import { runStateWords } from '../copy/running';
import { RUN_MARK, runSummary } from '../copy/runShape';
import { REACH_COPY } from '../copy/gooseReach';
import { AUTOSAVE_COPY, PROJECT_FILE_COPY, saveWords, UNDO_COPY } from '../copy/projectFile';
import type { AutoSaveSwitch } from '../state/useAutoSaveSwitch';
import type { ProjectHistory } from '../state/useHistory';
import { onLabNetwork } from '../features/network/endpoints';
import { useGoesOut } from '../features/network/useGoesOut';
import { SpinIcon } from '../ui/Spinner';
import { useDelayed } from '../ui/useDelayed';

/** What the File menu does besides opening: the App owns the dialogs (ADR-0135). */
export type FileActions = { open: () => void; save: () => void; saveAs: () => void; newProject: () => void; close: () => void };

export function TopBar({ title, subtitle, project, busy, working = '', saveState, run, file, history, openNetwork, autoSave }: {
  title: string; subtitle: string; project: Project | null; busy: boolean; working?: string; saveState: string;
  run: (action: 'start' | 'stop' | 'pause') => void; file: FileActions; history: ProjectHistory; openNetwork: () => void;
  autoSave: AutoSaveSwitch;
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
  const goesOut = useGoesOut(running);
  const words = saveWords(saveState, autoSave);
  return <header className="topbar">
    <div className="topbar-title"><h1>{title}</h1><p>{subtitle}</p></div>
    {/* While IEDs run the drawing is locked: the idle "All changes saved" yields its room so the page
        subtitle stays whole at 1366 px; "Saving..." and "Not saved" always show (change 0106). */}
    {!(running && words.text === 'All changes saved') && <span className="small muted" aria-live="polite" title={words.tip}>{words.text}</span>}
    {project && <ReachChip project={project} open={openNetwork} />}
    {/* Only while the Go engine really sends GOOSE out (ADR-0119). Icon-sized so the page subtitle stays
        whole at 1366 px; the words are in the tooltip, for screen readers, and in Settings (ADR-0073). */}
    {goesOut && <button className="goose-pill" title={REACH_COPY.pillHint}
      aria-label={`${REACH_COPY.pill}. ${REACH_COPY.pillHint}`} onClick={openNetwork}><Radio size={16} aria-hidden /></button>}
    {history.available && <UndoRedo history={history} busy={busy || !project} />}
    <Menu label="Project file" align="right" trigger={<><FileUp size={17} />File</>} items={[
      { label: PROJECT_FILE_COPY.menu.open, icon: <FolderOpen size={16} />, onSelect: file.open, disabled: running || busy },
      { label: PROJECT_FILE_COPY.menu.new, icon: <FilePlus size={16} />, onSelect: file.newProject, disabled: running || busy || !project },
      { label: PROJECT_FILE_COPY.menu.save, icon: <Save size={16} />, onSelect: file.save, disabled: running || busy || !project },
      { label: PROJECT_FILE_COPY.menu.saveAs, icon: <SaveAll size={16} />, onSelect: file.saveAs, disabled: busy || !project },
      { label: PROJECT_FILE_COPY.menu.close, icon: <FileX size={16} />, onSelect: file.close, disabled: running || busy || !project },
      { label: autoSave.on ? AUTOSAVE_COPY.turnOff : AUTOSAVE_COPY.turnOn, onSelect: autoSave.toggle, disabled: busy || !project,
        icon: autoSave.on ? <ToggleRight size={16} /> : <ToggleLeft size={16} /> },
    ]} />
    <span className={`run-state shape-${shape}`} role="status">{slow ? <><SpinIcon size={14} />{slow}</>
      : <><span className="run-mark" aria-hidden>{RUN_MARK[shape]}</span>{state}</>}</span>
    {running && <Menu label="Run options" trigger={<MoreHorizontal size={17} />} items={[
      { label: runtime === 'paused' ? 'Resume automation' : 'Pause automation', icon: <Pause size={16} />,
        onSelect: () => run('pause'), disabled: busy },
    ]} />}
    {running
      // Stop is solid red like every Stop and Disconnect (owner, ADR-0135); the run state beside it keeps ○ ◐ ●.
      ? <button className="run-pill danger" data-tour="run" disabled={busy}
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

/** Small Undo and Redo for the whole project (ADR-0135); the tips say what they take back. */
function UndoRedo({ history, busy }: { history: ProjectHistory; busy: boolean }) {
  const status = history.status;
  const tip = (can: boolean, what: string, words: (what: string) => string, nothing: string) =>
    status?.running ? UNDO_COPY.running : can ? words(what) : nothing;
  return <div className="history-buttons" role="group" aria-label="Undo and redo">
    <button className="icon ghost" aria-label={UNDO_COPY.undo} disabled={busy || !history.canUndo} onClick={history.undo}
      title={tip(history.canUndo, status?.undo ?? '', UNDO_COPY.undoTip, UNDO_COPY.nothingUndo)}><Undo2 size={17} /></button>
    <button className="icon ghost" aria-label={UNDO_COPY.redo} disabled={busy || !history.canRedo} onClick={history.redo}
      title={tip(history.canRedo, status?.redo ?? '', UNDO_COPY.redoTip, UNDO_COPY.nothingRedo)}><Redo2 size={17} /></button>
  </div>;
}
