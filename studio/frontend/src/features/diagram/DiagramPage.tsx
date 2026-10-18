import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import { Copy, Cpu, Redo2, Trash2, Undo2 } from 'lucide-react';
import { api } from '../../api/client';
import type { Diagram, EquipmentKind, EquipmentNode, Project } from '../../api/types';
import { LiveValues } from '../../equipment/liveContext';
import { sectionSentence } from '../../equipment/energization';
import type { useStudio } from '../../state/useStudio';
import { AddMenu } from './AddMenu';
import { addBay as placeBay, type BayKind } from '../../equipment/bayTemplates';
import { Canvas } from './Canvas';
import { EquipmentSheet } from './EquipmentSheet';
import { SldImport } from './SldImport';
import { TidyButton } from './TidyButton';
import { useCanvasShortcuts } from './useCanvasShortcuts';
import { useModes } from '../devices/useModes';
import { SignalLayerSwitches } from './signals/SignalLayerSwitches';
import { SignalOverlay, type SignalSelection } from './signals/SignalOverlay';
import { SignalSheet } from './signals/SignalSheet';
import { useSignalLayers } from './signals/useSignalLayers';
import { OutsideSenders } from './signals/OutsideSenders';
import { TestMenu, type MenuAt } from './TestMenu';
import './diagram.css';

type Studio = ReturnType<typeof useStudio>;

export function DiagramPage({ studio, project, locked, openDevices, openFile, openNetwork }: {
  studio: Studio; project: Project; locked: boolean; openDevices: (flowIedId?: string) => void;
  openFile: () => void; openNetwork: () => void;
}) {
  const [selection, setSelection] = useState<string[]>([]);
  const [operation, setOperation] = useState<{ nodeId: string; message: string } | null>(null);
  const [signal, setSignal] = useState<SignalSelection>(null);
  const [menu, setMenu] = useState<MenuAt | null>(null);
  const [tested, setTested] = useState<{ nodeId: string; message: string } | null>(null);
  const runningKey = (project.running_ieds || []).join(',');
  const running = useMemo(() => new Set(runningKey ? runningKey.split(',') : []), [runningKey]);
  const diagram = project.diagram;
  const iedKey = JSON.stringify(project.ieds.map(ied => [ied.id, ied.name]));
  const iedNames = useMemo(() => Object.fromEntries(project.ieds.map(ied => [ied.id, ied.name])), [iedKey]);
  const { modes } = useModes(project.runtime !== 'stopped', project.revision);
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
  const add = (kind: EquipmentKind) => {
    const count = diagram.nodes.filter(node => node.data.kind === kind).length + 1;
    const base = { kind, label: `${kind}-${count}`, rotation: 0 };
    const item: EquipmentNode = { id: crypto.randomUUID(), type: 'equipment',
      position: { x: 200 + (diagram.nodes.length % 4) * 140, y: 100 + Math.floor(diagram.nodes.length / 4) * 160 },
      data: kind === 'INFEED' ? { ...base, infeed_type: 'line', infeed_state: 'live' } :
        kind === 'JUNCTION' || kind === 'RELAY' ? base : { ...base, trip_command_polarity: 'operate_false',
        trip_state_polarity: 'operated_true', push_mode: 'latching_status', push_value: true, initial_position: 'open' } };
    edit({ ...diagram, nodes: [...diagram.nodes, item] }); setSelection([item.id]);
  };
  const junctions = selection.filter(id => diagram.nodes.find(node => node.id === id)?.data.kind === 'JUNCTION');
  const addBay = (kind: BayKind) => {
    const result = placeBay(diagram, kind, { attach: junctions });
    edit(result.diagram); setSelection(result.ids);
  };
  const attachHint = junctions.length ? `New bays connect to the ${junctions.length} selected busbar point${junctions.length > 1 ? 's' : ''}.`
    : 'Tip: select a busbar junction first and the bay connects to it.';
  const operate = useCallback((nodeId: string, action: string) => {
    setOperation(null);
    studio.perform(async () => {
      const result = await api<{ action: string; wire_value: boolean | null }>(
        `/equipment/${encodeURIComponent(nodeId)}/operate`, 'POST', { action });
      const sent = result.wire_value === null ? '' : ` · sent ${result.wire_value ? 1 : 0}`;
      setOperation({ nodeId, message: `${result.action[0].toUpperCase() + result.action.slice(1)} done${sent}` });
    });
  }, [studio.perform]);
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
  const toolbar = <div className="canvas-toolbar" role="toolbar" aria-label="Diagram tools">
    <AddMenu disabled={locked} add={add} addBay={addBay} attachHint={attachHint} />
    <SldImport disabled={locked} revision={project.revision} apply={(next: Diagram) => edit(next)} prepare={studio.prepare} />
    <TidyButton diagram={diagram} selection={selection} locked={locked} apply={(next: Diagram) => edit(next)} />
    <span className="divider" />
    <button className="icon ghost" title="Undo" aria-label="Undo" disabled={locked || !studio.canUndo} onClick={studio.undo}><Undo2 size={18} /></button>
    <button className="icon ghost" title="Redo" aria-label="Redo" disabled={locked || !studio.canRedo} onClick={studio.redo}><Redo2 size={18} /></button>
    <span className="divider" />
    <SignalLayerSwitches layers={layers.state.layers} toggle={layers.toggle} />
    <OutsideSenders state={layers.state} importFile={openFile} />
    {!project.ieds.length && <><span className="divider" />
      <button className="ghost" onClick={() => openDevices()}><Cpu size={17} />Import an IED first</button></>}
  </div>;
  return <div className="diagram">
    <Canvas diagram={diagram} change={edit} locked={locked} selection={selection} setSelection={setSelection}
      valueStore={studio.equipmentValues} energization={studio.energization} iedNames={iedNames} modeBadges={modes.equipment} operate={operate} toolbar={toolbar} overlay={overlay} nodeMenu={openMenu} />
    {menu && menuNode && <TestMenu at={menu} node={menuNode} project={project} running={running} test={test}
      openSheet={() => setSelection([menuNode.id])} close={() => setMenu(null)} />}
    {signal && !selection.length && <SignalSheet state={layers.state} selection={signal} select={selectSignal} showFlow={openDevices}
      lab={project.network.mode === 'lab'} openNetwork={openNetwork} />}
    {selected && <LiveValues.Provider value={studio.equipmentValues}><EquipmentSheet node={selected} project={project}
      locked={locked} busy={studio.busy} values={studio.values} perform={studio.perform} operate={operate}
      operation={operation?.nodeId === selected.id ? operation.message : undefined} close={() => setSelection([])}
      section={sectionSentence(energy.nodes[selected.id])} hints={energy.hints?.[selected.id]}
      update={data => edit({ ...diagram, nodes: diagram.nodes.map(item => item.id === selected.id ? { ...item, data } : item) })}
      applyColors={(active, inactive) => edit({ ...diagram, nodes: diagram.nodes.map(item =>
        item.data.kind === selected.data.kind ? { ...item, data: { ...item.data, active_color: active, inactive_color: inactive } } : item) })}
      remove={actions.remove} duplicate={actions.duplicate} showFlow={openDevices}
      running={running} test={test} testMessage={tested?.nodeId === selected.id ? tested.message : undefined} /></LiveValues.Provider>}
    {selection.length > 1 && <aside className="sheet"><div className="sheet-head"><div className="sheet-title">
      <h2>{selection.length} selected</h2><p>Drag to move them together</p></div></div>
      <div className="sheet-body"><div className="sheet-section"><div className="sheet-actions">
        <button disabled={locked} onClick={actions.duplicate}><Copy size={16} />Duplicate</button>
        <button className="danger-outline" disabled={locked} onClick={actions.remove}><Trash2 size={16} />Delete</button>
      </div><p className="small muted" style={{ margin: 0 }}>Shortcuts: Ctrl+C copy, Ctrl+V paste, Ctrl+D duplicate, arrows nudge.</p>
      </div></div></aside>}
  </div>;
}
