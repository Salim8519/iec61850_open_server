import { useMemo } from 'react';
import { useStore, type ReactFlowState } from '@xyflow/react';
import type { EquipmentNode } from '../../../api/types';
import type { Box } from '../../../equipment/signalLayout';

/** Every node's measured size as one text, so the overlay re-plans only when a size changes. The
 * drawing's own nodes carry no size (the diagram is controlled); React Flow measures them in its
 * internal lookup. Without this, an IED component (200 px wide) was routed as an 80 px box. */
function sizeText(state: ReactFlowState): string {
  const parts: string[] = [];
  state.nodeLookup.forEach((node, id) => parts.push(`${id}\t${node.measured?.width ?? 0}\t${node.measured?.height ?? 0}`));
  return parts.join('\n');
}

/** Boxes of the drawing in flow coordinates, with measured sizes (80 x 60 until measured). */
export function useMeasuredBoxes(nodes: EquipmentNode[]): Box[] {
  const text = useStore(sizeText);
  const sizes = useMemo(() => new Map(text.split('\n').filter(Boolean).map(line => {
    const [id, width, height] = line.split('\t');
    return [id, { width: Number(width), height: Number(height) }] as const;
  })), [text]);
  return useMemo(() => nodes.map(node => {
    const size = sizes.get(node.id);
    return { id: node.id, x: node.position.x, y: node.position.y,
      width: size?.width || node.measured?.width || 80, height: size?.height || node.measured?.height || 60 };
  }), [nodes, sizes]);
}
