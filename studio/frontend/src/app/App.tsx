import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Activity, Boxes, ChevronDown, FileUp, FolderOpen, Network, Pause, Play, Save, Square, Workflow, X, Zap, Undo2, Redo2 } from 'lucide-react';
import { useStudio } from '../state/useStudio';
import type { Diagram, EquipmentKind, EquipmentNode } from '../api/types';
import { api } from '../api/client';
import { Palette } from '../features/substation/Palette';
import { LEGACY_KINDS } from '../equipment/registry';
import { Canvas } from '../features/substation/Canvas';
import { LiveValues } from '../equipment/liveContext';
import { Inspector } from '../features/substation/Inspector';
import { IedWorkspace } from '../features/ieds/IedWorkspace';
import { ImportSelection } from '../features/ieds/ImportSelection';
import { Scenarios } from '../features/scenarios/Scenarios';
import { Connections } from '../features/connections/Connections';
import { EventLog } from '../features/event-log/EventLog';
import { RecoveryNotice } from '../components/RecoveryNotice';
import { MigrationNotice } from '../components/MigrationNotice';
import { SignalTest } from '../features/signal-test/SignalTest';
import { useCanvasShortcuts } from '../features/substation/useCanvasShortcuts';

const tabs = [
  { id: 'substation', title: 'Substation', icon: Workflow },
  { id: 'ieds', title: 'IEDs & Signals', icon: Boxes },
  { id: 'signal-test', title: 'Signal Test', icon: Zap },
  { id: 'scenarios', title: 'Scenarios', icon: Activity },
  { id: 'connections', title: 'Connections', icon: Network },
];

export function App() {
  const studio = useStudio();
  const [tab, setTab] = useState('substation');
  const [selection, setSelection] = useState<string[]>([]);
  const [panelWidth, setPanelWidth] = useState<number | null>(null);
  const [logs, setLogs] = useState(false);
  const [operation, setOperation] = useState<{ nodeId: string; message: string } | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const project = studio.project;
  const locked = project?.runtime !== 'stopped';
  const diagram = project?.diagram || { nodes: [], edges: [] };
  const iedKey = JSON.stringify((project?.ieds || []).map(ied => [ied.id, ied.name]));
  const iedNames = useMemo(() => Object.fromEntries((project?.ieds || []).map(ied => [ied.id, ied.name])), [iedKey]);
  const selected = selection.length === 1 ? selection[0] : null;
  const node = diagram.nodes.find(item => item.id === selected);
  const select = useCallback((id: string | null) => setSelection(id ? [id] : []), []);
  useEffect(() => {
    const present = selection.filter(id => diagram.nodes.some(item => item.id === id));
    if (present.length !== selection.length) setSelection(present);
  }, [diagram.nodes, selection]);
  const edit = useCallback((next: Diagram, group?: string) => {
    studio.updateDiagram(next, group);
  }, [studio.updateDiagram]);
  const actions = useCanvasShortcuts({ active: tab === 'substation' && !studio.busy && !!project, locked, diagram,
    selection, edit, setSelection, undo: studio.undo, redo: studio.redo });
  const add = useCallback((kind: EquipmentKind) => {
    if ((LEGACY_KINDS as string[]).includes(kind)) return;
    const base = { kind, label: `${kind}-${diagram.nodes.filter(n => n.data.kind === kind).length + 1}`, rotation: 0 };
    const item: EquipmentNode = { id: crypto.randomUUID(), type: 'equipment',
      position: { x: 200 + (diagram.nodes.length % 4) * 140, y: 100 + Math.floor(diagram.nodes.length / 4) * 160 },
      data: kind === 'JUNCTION' || kind === 'RELAY' ? base :
        { ...base, trip_command_polarity: 'operate_false', trip_state_polarity: 'operated_true',
          push_mode: 'latching_status', push_value: true, initial_position: 'open' } };
    edit({ ...diagram, nodes: [...diagram.nodes, item] }); select(item.id);
  }, [diagram, edit]);
  const operate = useCallback((nodeId: string, action: string) => {
    setOperation(null);
    studio.perform(async () => {
      const result = await api<{ action: string; wire_value: boolean | null }>(
        `/equipment/${encodeURIComponent(nodeId)}/operate`, 'POST', { action });
      const sent = result.wire_value === null ? '' : ` · sent ${result.wire_value ? 1 : 0}`;
      setOperation({ nodeId,
        message: `${result.action[0].toUpperCase() + result.action.slice(1)} applied${sent}` });
    });
  }, [studio.perform]);
  return <div className="app">
    {studio.importPreview && <ImportSelection preview={studio.importPreview} busy={studio.busy}
      error={studio.error} confirm={studio.confirmImport} cancel={studio.cancelImport} />}
    <header className="header"><div className="row">
      <div className="brand"><span className="brand-mark"><Zap size={19} /></span><span>Substation Studio<small>IEC 61850 LABORATORY</small></span></div>
      <span className="header-divider" /><span className="project-title small">{project?.name || 'Connecting…'}</span>
    </div><div className="row">
      <span className="small muted save-label">{studio.saveState}</span>
      <button className="icon" title="Save project" onClick={studio.save} disabled={!project || studio.busy || locked}><Save size={15} /></button>
      <button className="icon" title="Open CID or project" onClick={() => fileInput.current?.click()} disabled={locked || studio.busy}><FolderOpen size={15} /></button>
      <span className="header-divider" />
      {project?.runtime === 'stopped' ? <button className="primary" disabled={studio.busy || !project.ieds.length} onClick={() => studio.runtime('start')}><Play size={14} />Run</button> :
        <><button disabled={studio.busy} onClick={() => studio.runtime('pause')}><Pause size={14} />{project?.runtime === 'paused' ? 'Resume' : 'Pause automation'}</button>
          <button disabled={studio.busy} onClick={() => studio.runtime('stop')}><Square size={13} />Stop</button></>}
    </div></header>
    <nav className="tabs">{tabs.map(item => <button key={item.id} className={tab === item.id ? 'active' : ''} onClick={() => setTab(item.id)}>
      <item.icon size={15} />{item.title}</button>)}</nav>
    <RecoveryNotice recovery={project?.startup_recovery} />
    <MigrationNotice notice={project?.migration_notice} />
    {studio.error && <div className="error-banner" role="alert"><span>{studio.error}</span><button className="ghost" onClick={() => studio.setError('')} aria-label="Dismiss error"><X size={16} /></button></div>}
    <input ref={fileInput} type="file" accept=".cid,.icd,.scd,.subsim" style={{ display: 'none' }} onChange={event => {
      const file = event.target.files?.[0]; if (file) studio.importFile(file); event.target.value = '';
    }} />
    <div className="workspace">
      {tab === 'substation' && <>
        <Palette locked={locked || !project} ieds={project?.ieds || []} add={add} openIeds={() => setTab('ieds')} />
        <main className="main-view"><div className="view-heading"><div><h1>Single-line diagram</h1>
          <p className="muted small">Build the bay. Map the signals. Run the IEDs.</p></div>
          <div className="row">
            <button className="icon" title="Undo diagram change" disabled={locked || !studio.canUndo} onClick={studio.undo}><Undo2 size={15} /></button>
            <button className="icon" title="Redo diagram change" disabled={locked || !studio.canRedo} onClick={studio.redo}><Redo2 size={15} /></button>
            <span className={`badge ${project?.runtime === 'running' ? 'live' : ''}`}><span className="dot" />{project?.runtime || 'Connecting'}</span>
            <a className="button" href="/api/v1/project/download"><FileUp size={14} />Export project</a></div></div>
          <Canvas diagram={diagram} change={edit} locked={locked} selection={selection} setSelection={setSelection} valueStore={studio.equipmentValues} iedNames={iedNames} operate={operate} />
        </main>
        {project && <LiveValues.Provider value={studio.equipmentValues}><Inspector node={node} selectionCount={selection.length} project={project} locked={locked} busy={studio.busy}
          values={studio.values} perform={studio.perform} operate={operate}
          panelWidth={panelWidth} setPanelWidth={setPanelWidth}
          operation={operation && operation.nodeId === node?.id ? operation.message : undefined}
          update={data => edit({ ...diagram, nodes: diagram.nodes.map(item => item.id === selected ? { ...item, data } : item) })}
          applyColors={(active, inactive) => node && edit({ ...diagram, nodes: diagram.nodes.map(item =>
            item.data.kind === node.data.kind ? { ...item, data: { ...item.data, active_color: active, inactive_color: inactive } } : item) })}
          remove={actions.remove} duplicate={actions.duplicate} /></LiveValues.Provider>}
      </>}
      {tab === 'ieds' && <IedWorkspace ieds={project?.ieds || []} running={locked} values={studio.values}
        addTemplate={studio.addTemplate} importFile={() => fileInput.current?.click()} perform={studio.perform} prepare={studio.prepare} />}
      {tab === 'signal-test' && project && <SignalTest project={project} values={studio.values} events={studio.events}
        perform={studio.perform} prepare={studio.prepare} />}
      {tab === 'scenarios' && <Scenarios ieds={project?.ieds || []} scenarios={project?.scenarios || []}
        running={locked} perform={studio.perform} />}
      {tab === 'connections' && <Connections ieds={project?.ieds || []} />}
    </div>
    {logs && <EventLog events={studio.events} />}
    <footer className="status-bar"><div className="row"><span className="dot" style={{ color: project ? '#15803d' : '#b45309' }} />
      {project ? 'Local engine connected' : 'Connecting to engine'}<span>·</span>{project?.ieds.length || 0} IEDs<span>·</span>{diagram.nodes.length} equipment
      {project && <><span>·</span><span title={project.session_id}>Session {project.session_id.slice(0, 8)} / revision {project.revision}</span></>}</div>
      <button className="ghost" onClick={() => setLogs(!logs)}><Activity size={12} />Event log ({studio.events.length})<ChevronDown size={12} style={{ transform: logs ? 'rotate(180deg)' : undefined }} /></button>
    </footer>
  </div>;
}
