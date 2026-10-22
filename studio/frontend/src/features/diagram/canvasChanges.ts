import { applyEdgeChanges, applyNodeChanges, type EdgeChange, type NodeChange } from '@xyflow/react';
import type { Diagram, EquipmentNode } from '../../api/types';

/** A React Flow change list written as a step on a diagram, so one step can build on another. */
export type DiagramStep = (from: Diagram) => Diagram;

export const edgeStep = (changes: EdgeChange[]): DiagramStep =>
  from => ({ ...from, edges: applyEdgeChanges(changes, from.edges) });

/** Node changes; while a drag is on they apply to its draft nodes. Wires whose item is gone go with it. */
export const nodeStep = (changes: NodeChange<EquipmentNode>[], draft: EquipmentNode[] | null): DiagramStep => from => {
  const nodes = applyNodeChanges(changes, draft ?? from.nodes);
  const ids = new Set(nodes.map(node => node.id));
  return { ...from, nodes, edges: from.edges.filter(edge => ids.has(edge.source) && ids.has(edge.target)) };
};

/** True when the change list deletes something (Delete or Backspace on the selection). */
export const hasRemoval = (changes: readonly { type: string }[]): boolean => changes.some(item => item.type === 'remove');
