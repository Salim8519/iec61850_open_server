import { Download, FileUp, FolderOpen, MoreHorizontal, Pause, Play, Save, Square } from 'lucide-react';
import type { Project } from '../api/types';
import { Menu } from '../ui/Menu';

export function TopBar({ title, subtitle, project, busy, saveState, run, openFile, save }: {
  title: string; subtitle: string; project: Project | null; busy: boolean; saveState: string;
  run: (action: 'start' | 'stop' | 'pause') => void; openFile: () => void; save: () => void;
}) {
  const runtime = project?.runtime || 'stopped';
  const running = runtime !== 'stopped';
  const active = project?.ieds.filter(ied => ied.role !== 'off').length || 0;
  const state = !project ? 'Connecting…' : runtime === 'running' ? `Running · ${active} IED${active === 1 ? '' : 's'}`
    : runtime === 'paused' ? 'Running · automation paused' : runtime === 'stopped' ? 'Stopped' : runtime;
  return <header className="topbar">
    <div className="topbar-title"><h1>{title}</h1><p>{subtitle}</p></div>
    <span className="small muted" aria-live="polite">{saveState}</span>
    <Menu label="Project file" align="right" trigger={<><FileUp size={17} />File</>} items={[
      { label: 'Open file…', icon: <FolderOpen size={16} />, onSelect: openFile, disabled: running || busy },
      { label: 'Save project', icon: <Save size={16} />, onSelect: save, disabled: running || busy || !project },
      { label: 'Download project', icon: <Download size={16} />,
        onSelect: () => { window.location.href = '/api/v1/project/download'; }, disabled: !project },
    ]} />
    <span className="run-state" role="status"><span className={`dot${running ? ' on' : ''}`} />{state}</span>
    {running && <Menu label="Run options" trigger={<MoreHorizontal size={17} />} items={[
      { label: runtime === 'paused' ? 'Resume automation' : 'Pause automation', icon: <Pause size={16} />,
        onSelect: () => run('pause'), disabled: busy },
    ]} />}
    {running
      ? <button className="run-pill running" disabled={busy} onClick={() => run('stop')}><Square size={15} />Stop</button>
      : <button className="run-pill primary" disabled={busy || !active} title={active ? 'Start all IEDs' : 'Add an IED first'}
        onClick={() => run('start')}><Play size={16} />Run</button>}
  </header>;
}
