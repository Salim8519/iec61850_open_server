import { memo, useCallback, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from 'react';
import { ReactFlow, Background, Controls, BackgroundVariant, ReactFlowProvider,
  useReactFlow, applyNodeChanges, applyEdgeChanges, addEdge, ConnectionMode, SelectionMode,
  type NodeChange, type EdgeChange, type Connection, type NodeTypes } from '@xyflow/react';
import type { Diagram, EquipmentKind, EquipmentNode as NodeType } from '../../api/types';
import { EquipmentNode } from '../../equipment/EquipmentNode';
import { IedNames, LiveValues, ModeBadges } from '../../equipment/liveContext';
import { stableFlowNodes, type FlowNodeCache } from '../../equipment/flowNodes';
import { rememberMeasured, type MeasuredStore } from '../../equipment/measured';
import type { EquipmentValueStore } from '../../equipment/liveValues';
import { energizedEdges, type EdgeCache, type EnergizationStore } from '../../equipment/energization';
import { EnergizationLegend } from './EnergizationLegend';
import { useZoomTier } from './useZoomTier';
import { addBay, BAYS, type BayKind } from '../../equipment/bayTemplates';
import './energization.css';

const nodeTypes: NodeTypes = { equipment: EquipmentNode };
const snapGrid: [number, number] = [20, 20];
const defaultEdgeOptions = { type: 'smoothstep', interactionWidth: 24, style: { stroke: '#1f2937', strokeWidth: 2.5 } };
const connectionLineStyle = { stroke: '#2563eb', strokeWidth: 2, strokeDasharray: '6 4' };
const panButtons = [1, 2];
const fitOptions = { maxZoom: 1, padding: 0.3 };
const multiKeys = ['Shift', 'Control', 'Meta'];
const deleteKeys = ['Delete', 'Backspace'];
const noMenu = (event: React.MouseEvent | MouseEvent) => event.preventDefault();
type Props = {
  diagram: Diagram; locked: boolean; valueStore: EquipmentValueStore; iedNames: Record<string, string>;
  energization: EnergizationStore;
  modeBadges?: Record<string, string>;
  selection: string[]; setSelection: (ids: string[]) => void; operate: (nodeId: string, action: string) => void;
  change: (diagram: Diagram, group?: string) => void; toolbar?: ReactNode;
  /** View-only layers drawn in flow coordinates (spec signal-layers); never part of the diagram. */
  overlay?: ReactNode;
  /** Right-click on an equipment while running opens its test menu (ADR-0067). */
  nodeMenu?: (nodeId: string, x: number, y: number) => void;
};

function FlowCanvas({ diagram, locked, valueStore, iedNames, energization, modeBadges = {}, selection, setSelection, operate, change, toolbar, overlay, nodeMenu }: Props) {
  const flow = useReactFlow<NodeType>();
  const zoomTier = useZoomTier();
  const selectionRef = useRef(selection);
  selectionRef.current = selection;
  // Drag frames move a local draft; the diagram (history, save, inspector) updates once on drop.
  const draftRef = useRef<NodeType[] | null>(null);
  const [draft, setDraft] = useState<NodeType[] | null>(null);
  const nodeCache = useRef<FlowNodeCache>(new Map());
  const measured = useRef<MeasuredStore>(new Map());
  const edgeCache = useRef<EdgeCache>(new Map());
  const energy = useSyncExternalStore(energization.subscribe, energization.get, energization.get);
  const edges = useMemo(() => {
    const result = energizedEdges(diagram.edges, energy.edges, edgeCache.current);
    edgeCache.current = result.cache;
    return result.edges;
  }, [diagram.edges, energy]);
  const nodesChanged = useCallback((changes: NodeChange<NodeType>[]) => {
    rememberMeasured(measured.current, changes);
    const selects = changes.filter(change => change.type === 'select');
    if (selects.length) {
      let next = selectionRef.current;
      for (const item of selects) next = item.selected ? next.includes(item.id) ? next : [...next, item.id] :
        next.filter(id => id !== item.id);
      if (next !== selectionRef.current) { selectionRef.current = next; setSelection(next); }
    }
    const structural = changes.filter(change => change.type !== 'select');
    if (locked || !structural.length) return;
    const moving = structural.filter(change => change.type !== 'dimensions');
    if (!moving.length) return;
    const base = draftRef.current ?? diagram.nodes;
    if (moving.length && moving.length === structural.length &&
        moving.every(change => change.type === 'position' && change.dragging)) {
      draftRef.current = applyNodeChanges(moving, base); setDraft(draftRef.current); return;
    }
    if (draftRef.current) { draftRef.current = null; setDraft(null); }
    const nodes = applyNodeChanges(moving, base);
    const ids = new Set(nodes.map(node => node.id));
    change({ ...diagram, nodes, edges: diagram.edges.filter(edge => ids.has(edge.source) && ids.has(edge.target)) });
  }, [diagram, locked, change, setSelection]);
  const edgesChanged = useCallback((changes: EdgeChange[]) => {
    if (!locked) change({ ...diagram, edges: applyEdgeChanges(changes, diagram.edges) });
  }, [diagram, locked, change]);
  const connect = useCallback((connection: Connection) => {
    if (!locked) change({ ...diagram, edges: addEdge({ ...connection, type: 'smoothstep' }, diagram.edges) });
  }, [change, diagram, locked]);
  const projection = useMemo(() => {
    const result = stableFlowNodes(draft ?? diagram.nodes, new Set(selection), nodeCache.current, measured.current);
    nodeCache.current = result.cache;
    return result.nodes;
  }, [diagram.nodes, draft, selection]);
  const dragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault(); event.dataTransfer.dropEffect = 'move';
  }, []);
  const drop = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    const kind = event.dataTransfer.getData('application/studio-equipment') as EquipmentKind;
    const bay = event.dataTransfer.getData('application/studio-bay') as BayKind;
    if ((!kind && !bay) || locked) return;
    const position = flow.screenToFlowPosition({ x: event.clientX, y: event.clientY });
    if (bay in BAYS) {
      const result = addBay(diagram, bay, { at: position });
      change(result.diagram); setSelection(result.ids); return;
    }
    const node: NodeType = { id: crypto.randomUUID(), type: 'equipment', position,
      data: { kind, label: `${kind}-${diagram.nodes.filter(n => n.data.kind === kind).length + 1}`, rotation: 0,
        ...(kind === 'PUSH_BUTTON' ? { push_mode: 'latching_status' as const } : {}),
        ...(kind === 'INFEED' ? { infeed_type: 'line' as const, infeed_state: 'live' as const } : {}) } };
    change({ ...diagram, nodes: [...diagram.nodes, node] }); setSelection([node.id]);
  }, [change, diagram, flow, locked, setSelection]);
  const dragStop = useCallback(() => {
    const moved = draftRef.current;
    draftRef.current = null; setDraft(null);
    if (moved && !locked) change({ ...diagram, nodes: moved });
  }, [change, diagram, locked]);
  const nodeClick = useCallback((_: React.MouseEvent, node: NodeType) => {
    if (locked && node.data.kind === 'PUSH_BUTTON') operate(node.id, 'press');
  }, [locked, operate]);
  const paneClick = useCallback(() => setSelection([]), [setSelection]);
  const nodeContext = useCallback((event: React.MouseEvent, node: NodeType) => {
    event.preventDefault();
    if (locked && nodeMenu) nodeMenu(node.id, event.clientX, event.clientY);
  }, [locked, nodeMenu]);
  return <LiveValues.Provider value={valueStore}><IedNames.Provider value={iedNames}><ModeBadges.Provider value={modeBadges}>
    <div className={`canvas zoom-${zoomTier}`} onDragOver={dragOver} onDrop={drop}>
      {toolbar}
      {locked && <div className="canvas-lock" role="status">Running · drawing is locked</div>}
      {locked && <EnergizationLegend energy={energy} />}
      <ReactFlow nodes={projection} edges={edges} nodeTypes={nodeTypes}
        onNodesChange={nodesChanged} onEdgesChange={edgesChanged} onConnect={connect}
        onNodeDragStop={dragStop} onSelectionDragStop={dragStop}
        onNodeClick={nodeClick} onPaneClick={paneClick} onPaneContextMenu={noMenu} onNodeContextMenu={nodeContext}
        connectionMode={ConnectionMode.Loose} connectionRadius={36} connectionLineStyle={connectionLineStyle}
        selectionOnDrag panOnDrag={panButtons} selectionMode={SelectionMode.Partial}
        multiSelectionKeyCode={multiKeys} nodeDragThreshold={3}
        nodesDraggable={!locked} nodesConnectable={!locked} deleteKeyCode={locked ? null : deleteKeys}
        snapToGrid snapGrid={snapGrid} minZoom={0.2} maxZoom={2} fitView fitViewOptions={fitOptions}
        defaultEdgeOptions={defaultEdgeOptions}>
        <Background variant={BackgroundVariant.Dots} gap={20} size={1.6} color="#b4bcc8" />
        <Controls showInteractive={false} position="bottom-left" />
        {overlay}
      </ReactFlow>
      {!diagram.nodes.length && <div className="canvas-empty"><h2>Start your single-line diagram</h2>
        <p>Press <strong>Add</strong> and pick a circuit breaker. Then connect it to an IED signal on the right.</p></div>}
    </div>
  </ModeBadges.Provider></IedNames.Provider></LiveValues.Provider>;
}

export const Canvas = memo(function Canvas(props: Props) {
  return <ReactFlowProvider><FlowCanvas {...props} /></ReactFlowProvider>;
});
