import { memo, useCallback, useMemo, useRef, useState } from 'react';
import { ReactFlow, Background, Controls, MiniMap, BackgroundVariant, ReactFlowProvider,
  useReactFlow, applyNodeChanges, applyEdgeChanges, addEdge, ConnectionMode, SelectionMode,
  type NodeChange, type EdgeChange, type Connection, type NodeTypes, type NodeDimensionChange } from '@xyflow/react';
import type { Diagram, EquipmentKind, EquipmentNode as NodeType } from '../../api/types';
import { EquipmentNode } from '../../equipment/EquipmentNode';
import { IedNames, LiveValues } from '../../equipment/liveContext';
import { stableFlowNodes, type FlowNodeCache } from '../../equipment/flowNodes';
import { rememberMeasured, type MeasuredStore } from '../../equipment/measured';
import { LEGACY_KINDS } from '../../equipment/registry';
import { applyNodeSizes, isLiveResize, isResizeEnd } from '../../equipment/resize';
import type { EquipmentValueStore } from '../../equipment/liveValues';

const nodeTypes: NodeTypes = { equipment: EquipmentNode };
const snapGrid: [number, number] = [20, 20];
const defaultEdgeOptions = { type: 'smoothstep', interactionWidth: 24, style: { stroke: '#526781', strokeWidth: 2 } };
const connectionLineStyle = { stroke: '#1d4ed8', strokeWidth: 2, strokeDasharray: '6 4' };
const panButtons = [1, 2];
const multiKeys = ['Shift', 'Control', 'Meta'];
const deleteKeys = ['Delete', 'Backspace'];
const noMenu = (event: React.MouseEvent | MouseEvent) => event.preventDefault();
type Props = {
  diagram: Diagram; locked: boolean; valueStore: EquipmentValueStore; iedNames: Record<string, string>;
  selection: string[]; setSelection: (ids: string[]) => void; operate: (nodeId: string, action: string) => void;
  change: (diagram: Diagram, group?: string) => void;
};

function FlowCanvas({ diagram, locked, valueStore, iedNames, selection, setSelection, operate, change }: Props) {
  const flow = useReactFlow<NodeType>();
  const selectionRef = useRef(selection);
  selectionRef.current = selection;
  // Drag frames move a local draft; the diagram (history, save, inspector) updates once on drop.
  const draftRef = useRef<NodeType[] | null>(null);
  const [draft, setDraft] = useState<NodeType[] | null>(null);
  const resizeGroup = useRef<string | undefined>(undefined);
  const nodeCache = useRef<FlowNodeCache>(new Map());
  const measured = useRef<MeasuredStore>(new Map());
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
    const sizing = structural.filter((change): change is NodeDimensionChange =>
      change.type === 'dimensions' && change.resizing !== undefined);
    const moving = structural.filter(change => change.type !== 'dimensions');
    const base = draftRef.current ?? diagram.nodes;
    if (moving.length && moving.length === structural.length &&
        moving.every(change => change.type === 'position' && change.dragging)) {
      draftRef.current = applyNodeChanges(moving, base); setDraft(draftRef.current); return;
    }
    if (draftRef.current) { draftRef.current = null; setDraft(null); }
    if (!sizing.length) resizeGroup.current = undefined;
    let nodes = moving.length ? applyNodeChanges(moving, base) : base;
    if (sizing.length) {
      if (!resizeGroup.current && sizing.some(isLiveResize)) resizeGroup.current = crypto.randomUUID();
      nodes = applyNodeSizes(nodes, sizing);
    }
    const group = resizeGroup.current;
    if (sizing.length && sizing.every(isResizeEnd)) resizeGroup.current = undefined;
    const ids = new Set(nodes.map(node => node.id));
    change({ ...diagram, nodes, edges: diagram.edges.filter(edge => ids.has(edge.source) && ids.has(edge.target)) }, group);
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
    if (!kind || locked || (LEGACY_KINDS as string[]).includes(kind)) return;
    const position = flow.screenToFlowPosition({ x: event.clientX, y: event.clientY });
    const node: NodeType = { id: crypto.randomUUID(), type: 'equipment', position,
      data: { kind, label: `${kind}-${diagram.nodes.filter(n => n.data.kind === kind).length + 1}`, rotation: 0,
        ...(kind === 'PUSH_BUTTON' ? { push_mode: 'latching_status' as const } : {}) } };
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
  return <LiveValues.Provider value={valueStore}><IedNames.Provider value={iedNames}>
    <div className="canvas" onDragOver={dragOver} onDrop={drop}>
      <div className="canvas-hint">{locked ? 'Live view · structure locked while running' : 'Drag empty space to select · Space or right-drag to pan · Ctrl+C / V / D to copy, paste, duplicate'}</div>
      <ReactFlow nodes={projection} edges={diagram.edges} nodeTypes={nodeTypes}
        onNodesChange={nodesChanged} onEdgesChange={edgesChanged} onConnect={connect}
        onNodeDragStop={dragStop} onSelectionDragStop={dragStop}
        onNodeClick={nodeClick} onPaneClick={paneClick} onPaneContextMenu={noMenu}
        connectionMode={ConnectionMode.Loose} connectionRadius={36} connectionLineStyle={connectionLineStyle}
        selectionOnDrag panOnDrag={panButtons} selectionMode={SelectionMode.Partial}
        multiSelectionKeyCode={multiKeys} nodeDragThreshold={3}
        nodesDraggable={!locked} nodesConnectable={!locked} deleteKeyCode={locked ? null : deleteKeys}
        snapToGrid snapGrid={snapGrid} minZoom={0.2} maxZoom={2} fitView
        defaultEdgeOptions={defaultEdgeOptions}>
        <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="#c8d3df" />
        <Controls showInteractive={false} />
        <MiniMap pannable zoomable nodeColor="#91a8c9" maskColor="rgba(245,247,250,.6)" />
      </ReactFlow>
      {!diagram.nodes.length && <div className="canvas-empty"><h2>Build your first bay</h2>
        <p>Start with a circuit breaker, then add switches and signals.</p></div>}
    </div>
  </IedNames.Provider></LiveValues.Provider>;
}

export const Canvas = memo(function Canvas(props: Props) {
  return <ReactFlowProvider><FlowCanvas {...props} /></ReactFlowProvider>;
});
