import { useCallback, useEffect, useMemo, useState } from 'react';
import { Copy, Cpu, Redo2, Trash2, Undo2 } from 'lucide-react';
import { api } from '../../api/client';
import type { Diagram, EquipmentKind, EquipmentNode, Project } from '../../api/types';
import { LiveValues } from '../../equipment/liveContext';
import type { useStudio } from '../../state/useStudio';
import { AddMenu } from './AddMenu';
import { Canvas } from './Canvas';
import { EquipmentSheet } from './EquipmentSheet';
import { SldImport } from './SldImport';
import { useCanvasShortcuts } from './useCanvasShortcuts';
import { useModes } from '../devices/useModes';
import './diagram.css';

type Studio = ReturnType<typeof useStudio>;

export function DiagramPage({ studio, project, locked, openDevices }: {
  studio: Studio; project: Project; locked: boolean; openDevices: (flowIedId?: string) => void;
}) {
  const [selection, setSelection] = useState<string[]>([]);
  const [operation, setOperation] = useState<{ nodeId: string; message: string } | null>(null);
  const diagram = project.diagram;
  const iedKey = JSON.stringify(project.ieds.map(ied => [ied.id, ied.name]));
  const iedNames = useMemo(() => Object.fromEntries(project.ieds.map(ied => [ied.id, ied.name])), [iedKey]);
  const { modes } = useModes(project.runtime !== 'stopped', project.revision);
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
      data: kind === 'JUNCTION' || kind === 'RELAY' ? base : { ...base, trip_command_polarity: 'operate_false',
        trip_state_polarity: 'operated_true', push_mode: 'latching_status', push_value: true, initial_position: 'open' } };
    edit({ ...diagram, nodes: [...diagram.nodes, item] }); setSelection([item.id]);
  };
  const operate = useCallback((nodeId: string, action: string) => {
    setOperation(null);
    studio.perform(async () => {
      const result = await api<{ action: string; wire_value: boolean | null }>(
        `/equipment/${encodeURIComponent(nodeId)}/operate`, 'POST', { action });
      const sent = result.wire_value === null ? '' : ` · sent ${result.wire_value ? 1 : 0}`;
      setOperation({ nodeId, message: `${result.action[0].toUpperCase() + result.action.slice(1)} done${sent}` });
    });
  }, [studio.perform]);
  const toolbar = <div className="canvas-toolbar" role="toolbar" aria-label="Diagram tools">
    <AddMenu disabled={locked} add={add} />
    <SldImport disabled={locked} revision={project.revision} apply={(next: Diagram) => edit(next)} prepare={studio.prepare} />
    <span className="divider" />
    <button className="icon ghost" title="Undo" aria-label="Undo" disabled={locked || !studio.canUndo} onClick={studio.undo}><Undo2 size={18} /></button>
    <button className="icon ghost" title="Redo" aria-label="Redo" disabled={locked || !studio.canRedo} onClick={studio.redo}><Redo2 size={18} /></button>
    {!project.ieds.length && <><span className="divider" />
      <button className="ghost" onClick={() => openDevices()}><Cpu size={17} />Import an IED first</button></>}
  </div>;
  return <div className="diagram">
    <Canvas diagram={diagram} change={edit} locked={locked} selection={selection} setSelection={setSelection}
      valueStore={studio.equipmentValues} iedNames={iedNames} modeBadges={modes.equipment} operate={operate} toolbar={toolbar} />
    {selected && <LiveValues.Provider value={studio.equipmentValues}><EquipmentSheet node={selected} project={project}
      locked={locked} busy={studio.busy} values={studio.values} perform={studio.perform} operate={operate}
      operation={operation?.nodeId === selected.id ? operation.message : undefined} close={() => setSelection([])}
      update={data => edit({ ...diagram, nodes: diagram.nodes.map(item => item.id === selected.id ? { ...item, data } : item) })}
      applyColors={(active, inactive) => edit({ ...diagram, nodes: diagram.nodes.map(item =>
        item.data.kind === selected.data.kind ? { ...item, data: { ...item.data, active_color: active, inactive_color: inactive } } : item) })}
      remove={actions.remove} duplicate={actions.duplicate} showFlow={openDevices} /></LiveValues.Provider>}
    {selection.length > 1 && <aside className="sheet"><div className="sheet-head"><div className="sheet-title">
      <h2>{selection.length} selected</h2><p>Drag to move them together</p></div></div>
      <div className="sheet-body"><div className="sheet-section"><div className="sheet-actions">
        <button disabled={locked} onClick={actions.duplicate}><Copy size={16} />Duplicate</button>
        <button className="danger-outline" disabled={locked} onClick={actions.remove}><Trash2 size={16} />Delete</button>
      </div><p className="small muted" style={{ margin: 0 }}>Shortcuts: Ctrl+C copy, Ctrl+V paste, Ctrl+D duplicate, arrows nudge.</p>
      </div></div></aside>}
  </div>;
}
