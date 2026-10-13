import type { NodeDimensionChange } from '@xyflow/react';
import type { EquipmentNode } from '../api/types';

export const BUSBAR_MIN_WIDTH = 80;
export const BUSBAR_MAX_WIDTH = 2400;

export function isLiveResize(change: NodeDimensionChange): boolean {
  return change.resizing === true;
}

export function isResizeEnd(change: NodeDimensionChange): boolean {
  return change.resizing === false;
}

/** Persist resize-driven widths on busbar data; plain measure events never resize. */
export function applyNodeSizes(nodes: EquipmentNode[], changes: NodeDimensionChange[]): EquipmentNode[] {
  const widths = new Map(changes
    .filter((change): change is NodeDimensionChange & { dimensions: { width: number } } =>
      change.resizing !== undefined && change.dimensions?.width !== undefined)
    .map(change => [change.id, change.dimensions.width]));
  if (!widths.size) return nodes;
  return nodes.map(node => {
    const width = widths.get(node.id);
    if (node.data.kind !== 'BUSBAR' || width === undefined) return node;
    const bounded = Math.min(BUSBAR_MAX_WIDTH, Math.max(BUSBAR_MIN_WIDTH, Math.round(width)));
    if (node.data.width === bounded) return node;
    return { ...node, data: { ...node.data, width: bounded } };
  });
}
