import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import { Copy, Cpu, Redo2, Trash2, Undo2 } from 'lucide-react';
import { api } from '../../api/client';
import type { CommandOptions, Diagram, EquipmentKind, EquipmentNode, Project } from '../../api/types';
import { commandBody } from '../../equipment/commandOptions';
import { operateWords } from '../../equipment/operateWords';
import { LiveValues } from '../../equipment/liveContext';
import { sectionSentence } from '../../equipment/energization';
import type { useStudio } from '../../state/useStudio';
import { AddMenu } from './AddMenu';
import { addBay as placeBay, type BayKind } from '../../equipment/bayTemplates';
import { defaultLabel } from '../../equipment/kindInfo';
import { Canvas } from './Canvas';
import { OpenPointPanel } from '../../equipment/pointPanelContext';
import { TapBusy, TapCommand } from '../../equipment/tapContext';
import { newTapChanger } from '../../equipment/tapChanger';
import { useTapCommands } from './tap/useTapCommands';
import { PointPanelDialog } from './points/PointPanelDialog';
import { EquipmentSheet } from './EquipmentSheet';
import { convertiblePushButtons, convertPushButtons } from '../../equipment/pushToPanel';
import { SldImport } from './SldImport';
import { TidyButton } from './TidyButton';
import { useCanvasShortcuts } from './useCanvasShortcuts';
import { useModeBadges } from '../devices/useModes';
import { createKeyedStore } from '../../equipment/keyedStore';
import { markClick } from '../../app/perf';
import { SignalLayerSwitches } from './signals/SignalLayerSwitches';
import { SignalOverlay, type SignalSelection } from './signals/SignalOverlay';
import { SignalSheet } from './signals/SignalSheet';
import { useSignalLayers } from './signals/useSignalLayers';
import { OutsideSenders } from './signals/OutsideSenders';
import { TestMenu, type MenuAt } from './TestMenu';
import { RunStrip } from './RunStrip';
import { runSummary } from '../../copy/runShape';
import './diagram.css';

type Studio = ReturnType<typeof useStudio>;

export function DiagramPage({ studio, project, locked, openDevices, openFile }: {
  studio: Studio; project: Project; locked: boolean; openDevices: (flowIedId?: string) => void;
  openFile: () => void;
}) {
  const [selection, setSelection] = useState<string[]>([]);
  const [operation, setOperation] = useState<{ nodeId: string; message: string } | null>(null);
  const [signal, setSignal] = useState<SignalSelection>(null);
  const [menu, setMenu] = useState<MenuAt | null>(null);
  const [tested, setTested] = useState<{ nodeId: string; message: string } | null>(null);
  const [panelId, setPanelId] = useState<string | null>(null);
  const runningKey = (project.running_ieds || []).join(',');
  const running = useMemo(() => new Set(runningKey ? runningKey.split(',') : []), [runningKey]);
  const diagram = project.diagram;
  // Only some IEDs run (change 0106): name them on the lock badge and grey the stopped IEDs' equipment.
  const summary = runSummary(project);
  const stoppedKey = summary.shape === 'partial' ? summary.stopped.map(ied => ied.id).join(',') : '';
  const stoppedIeds = useMemo(() => new Set(stoppedKey ? stoppedKey.split(',') : []), [stoppedKey]);
  const iedKey = JSON.stringify(project.ieds.map(ied => [ied.id, ied.name]));
  const iedNames = useMemo(() => Object.fromEntries(project.ieds.map(ied => [ied.id, ied.name])), [iedKey]);
  // Mode badges reach each equipment through a keyed store, never through this page's state.
  const badges = useMemo(() => createKeyedStore<string>(), []);
  useModeBadges(badges, project.runtime !== 'stopped', project.revision);
  const energy = useSyncExternalStore(studio.energization.subscribe, studio.energization.get, studio.energization.get);
  const layers = useSignalLayers(studio.signalLayers, project.revision, project.runtime !== 'stopped');
  const selectSignal = useCallback((next: SignalSelection) => { setSignal(next); if (next) setSelection([]); }, []);
  useEffect(() => { if (selection.length) setSignal(null); }, [selection]);
  const overlay = useMemo(() => <SignalOverlay store={studio.signalLayers} iedNames={iedNames} selection={signal}
    running={locked ? running : null} select={selectSignal} />, [studio.signalLayers, iedNames, signal, selectSignal, locked, running]);
  const selected = selection.length === 1 ? diagram.nodes.find(item => item.id === selection[0]) : undefined;
  useEffect(() => {
    const present = selection.filter(id => diagram.nodes.some(item => item.id === id));
    if (present.length !== selection.length) setSelection(present);
  }, [diagram.nodes, selection]);
  const edit = useCallback((next: Diagram, group?: string) => studio.updateDiagram(next, group), [studio.updateDiagram]);
  const actions = useCanvasShortcuts({ active: !studio.busy, locked, diagram, selection, edit, setSelection,
    undo: studio.undo, redo: studio.redo });
  // A click adds the symbol at a free spot in the visible middle of the canvas (change 0176).
  const add = useCallback((kind: EquipmentKind, at: { x: number; y: number }) => {
    const count = diagram.nodes.filter(node => node.data.kind === kind).length + 1;
    const base = { kind, label: defaultLabel(kind, count), rotation: 0 };
    const item: EquipmentNode = { id: crypto.randomUUID(), type: 'equipment', position: at,
      data: kind === 'INFEED' ? { ...base, infeed_type: 'line', infeed_state: 'live' } :
        kind === 'MEASUREMENT_LIST' ? { ...base, measurements: [] } :
        kind === 'POINT_PANEL' ? { ...base, ied_id: project.ieds.length === 1 ? project.ieds[0].id : null, points: [] } :
        kind === 'TAP_TRANSFORMER' ? { ...base, tap_changer: newTapChanger() } :
        kind === 'JUNCTION' || kind === 'RELAY' ? base : { ...base, trip_command_polarity: 'operate_false',
        trip_state_polarity: 'operated_true', push_mode: 'latching_status', push_value: true, initial_position: 'open' } };
    edit({ ...diagram, nodes: [...diagram.nodes, item] }); setSelection([item.id]);
  }, [diagram, edit, project.ieds]);
  const junctionKey = selection.filter(id => diagram.nodes.find(node => node.id === id)?.data.kind === 'JUNCTION').join(',');
  const junctions = useMemo(() => junctionKey ? junctionKey.split(',') : [], [junctionKey]);
  const addBay = useCallback((kind: BayKind) => {
    const result = placeBay(diagram, kind, { attach: junctions });
    edit(result.diagram); setSelection(result.ids);
  }, [diagram, edit, junctions]);
  const attachHint = junctions.length ? `New bays connect to the ${junctions.length} selected busbar point${junctions.length > 1 ? 's' : ''}.`
    : 'Tip: select a busbar junction first and the bay connects to it.';
  // Without options the body stays { action } exactly as before (ADR-0108).
  const operate = useCallback((nodeId: string, action: string, options?: CommandOptions) => {
    markClick(nodeId);
    setOperation(null);
    return studio.perform(async () => {
      const result = await api<{ action: string; wire_value: boolean | null; message?: string }>(
        `/equipment/${encodeURIComponent(nodeId)}/operate`, 'POST', commandBody(action, options));
      setOperation({ nodeId, message: operateWords(result) });
    });
  }, [studio.perform]);
  const report = useCallback((nodeId: string, message: string) => setOperation({ nodeId, message }), []);
  const tap = useTapCommands(studio.perform, report); // tap-changer transformers (ADR-0109)
  // Test buttons write the bound point in the simulated IED through the backend; no command is sent (ADR-0067).
  const test = useCallback((nodeId: string, action: string, extra?: { value?: number; row?: string }) => {
    setTested(null);
    studio.perform(async () => {
      await api(`/equipment/${encodeURIComponent(nodeId)}/test`, 'POST', { action, ...extra });
      setTested({ nodeId, message: `Test: ${action.replace('_', ' ')} set in the simulated IED` });
    });
  }, [studio.perform]);
  const openMenu = useCallback((nodeId: string, x: number, y: number) => setMenu({ nodeId, x, y }), []);
  const menuNode = menu ? diagram.nodes.find(item => item.id === menu.nodeId) : undefined;
  // The toolbar and lock note are made only when what they show changes, so Canvas (memo) skips
  // re-rendering when this page renders for an unrelated reason (speed lane F).
  const { canUndo, canRedo, undo, redo, prepare, runtime, busy } = studio;
  const noIeds = !project.ieds.length;
  const toolbar = useMemo(() => <div className="canvas-toolbar" role="toolbar" aria-label="Diagram tools">
    <AddMenu disabled={locked} add={add} addBay={addBay} attachHint={attachHint} />
    <SldImport disabled={locked} revision={project.revision} apply={(next: Diagram) => edit(next)} prepare={prepare} />
    <TidyButton diagram={diagram} selection={selection} locked={locked} apply={(next: Diagram) => edit(next)} />
    <span className="divider" />
    <button className="icon ghost" title="Undo" aria-label="Undo" disabled={locked || !canUndo} onClick={undo}><Undo2 size={18} /></button>
    <button className="icon ghost" title="Redo" aria-label="Redo" disabled={locked || !canRedo} onClick={redo}><Redo2 size={18} /></button>
    <span className="divider" />
    <SignalLayerSwitches layers={layers.layers} toggle={layers.toggle} />
    <OutsideSenders store={studio.signalLayers} importFile={openFile} />
    {noIeds && <><span className="divider" />
      <button className="ghost" onClick={() => openDevices()}><Cpu size={17} />Import an IED first</button></>}
  </div>, [locked, add, addBay, attachHint, project.revision, edit, prepare, diagram, selection, canUndo, canRedo, undo, redo,
    layers.layers, layers.toggle, studio.signalLayers, openFile, noIeds, openDevices]);
  const partial = summary.shape === 'partial';
  const summaryKey = partial ? `${summary.running.map(ied => ied.id)}|${summary.stopped.map(ied => ied.id)}` : '';
  const lockNote = useMemo(() => partial ? <RunStrip summary={summary} busy={busy} runAll={() => runtime('start')} /> : undefined,
    [partial, summaryKey, busy]);
  const panel = panelId ? diagram.nodes.find(item => item.id === panelId && item.data.kind === 'POINT_PANEL') : undefined;
  return <div className="diagram"><OpenPointPanel.Provider value={setPanelId}>
    <TapCommand.Provider value={tap.send}><TapBusy.Provider value={tap.busy}>
    <Canvas diagram={diagram} change={edit} locked={locked} selection={selection} setSelection={setSelection}
      valueStore={studio.equipmentValues} energization={studio.energization} iedNames={iedNames} modeBadges={badges} operate={operate} toolbar={toolbar} overlay={overlay} nodeMenu={openMenu}
      stoppedIeds={stoppedIeds} lockNote={lockNote} />
    {menu && menuNode && <TestMenu at={menu} node={menuNode} project={project} running={running} test={test}
      openSheet={() => setSelection([menuNode.id])} close={() => setMenu(null)} />}
    {signal && !selection.length && <SignalSheet store={studio.signalLayers} selection={signal} select={selectSignal} showFlow={openDevices} />}
    {selected && <LiveValues.Provider value={studio.equipmentValues}><EquipmentSheet node={selected} project={project}
      locked={locked} busy={studio.busy} perform={studio.perform} operate={operate}
      operation={operation?.nodeId === selected.id ? operation.message : undefined} close={() => setSelection([])}
      section={sectionSentence(energy.nodes[selected.id])} hints={energy.hints?.[selected.id]}
      update={data => edit({ ...diagram, nodes: diagram.nodes.map(item => item.id === selected.id ? { ...item, data } : item) })}
      applyColors={(active, inactive) => edit({ ...diagram, nodes: diagram.nodes.map(item =>
        item.data.kind === selected.data.kind ? { ...item, data: { ...item.data, active_color: active, inactive_color: inactive } } : item) })}
      remove={actions.remove} duplicate={actions.duplicate} showFlow={openDevices}
      pushToPanels={selected.data.kind === 'PUSH_BUTTON' ? { count: convertiblePushButtons(diagram),
        convert: () => { edit(convertPushButtons(diagram, iedNames).diagram); setSelection([]); } } : undefined}
      running={running} test={test} testMessage={tested?.nodeId === selected.id ? tested.message : undefined} /></LiveValues.Provider>}
    {panel && <LiveValues.Provider value={studio.equipmentValues}><PointPanelDialog key={panel.id} node={panel} project={project}
      locked={locked} running={running} close={() => setPanelId(null)}
      update={data => edit({ ...diagram, nodes: diagram.nodes.map(item => item.id === panel.id ? { ...item, data } : item) })} /></LiveValues.Provider>}
    {selection.length > 1 && <aside className="sheet"><div className="sheet-head"><div className="sheet-title">
      <h2>{selection.length} selected</h2><p>Drag to move them together</p></div></div>
      <div className="sheet-body"><div className="sheet-section"><div className="sheet-actions">
        <button disabled={locked} onClick={actions.duplicate}><Copy size={16} />Duplicate</button>
        <button className="danger-outline" disabled={locked} onClick={actions.remove}><Trash2 size={16} />Delete</button>
      </div><p className="small muted" style={{ margin: 0 }}>Shortcuts: Ctrl+C copy, Ctrl+V paste, Ctrl+D duplicate, arrows nudge.</p>
      </div></div></aside>}
  </TapBusy.Provider></TapCommand.Provider></OpenPointPanel.Provider></div>;
}
