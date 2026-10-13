import type { EquipmentKind, EquipmentNode } from '../api/types';

export const INITIAL_NODE_WIDTH = 116;
export const INITIAL_NODE_HEIGHT = 112;
const DRAW_SIZES: Partial<Record<EquipmentKind, number>> = { JUNCTION: 40 };

type CacheEntry = {
  source: EquipmentNode;
  selected: boolean;
  projected: EquipmentNode;
};

export type FlowNodeCache = Map<string, CacheEntry>;
export type MeasuredSizes = ReadonlyMap<string, { width: number; height: number }>;

function initialSize(source: EquipmentNode): [number, number] {
  const draw = DRAW_SIZES[source.data.kind];
  const width = source.data.kind === 'BUSBAR' && source.data.width ? source.data.width :
    source.initialWidth || draw || INITIAL_NODE_WIDTH;
  return [width, source.initialHeight || draw || INITIAL_NODE_HEIGHT];
}

/**
 * Retain unaffected node objects while supplying dimensions before first measure.
 * A replaced node must carry React Flow's last `measured` size: without it React Flow
 * discards the node's terminal bounds, and a re-measure racing another update can
 * leave them empty so attached wires stop rendering until the canvas remounts.
 */
export function stableFlowNodes(
  sourceNodes: EquipmentNode[], selectedIds: ReadonlySet<string>, previous: FlowNodeCache,
  measuredSizes: MeasuredSizes = new Map(),
) {
  const cache: FlowNodeCache = new Map();
  const nodes = sourceNodes.map(source => {
    const selected = selectedIds.has(source.id);
    const existing = previous.get(source.id);
    if (existing?.source === source && existing.selected === selected) {
      cache.set(source.id, existing);
      return existing.projected;
    }
    const [initialWidth, initialHeight] = initialSize(source);
    const measured = measuredSizes.get(source.id);
    const projected: EquipmentNode = {
      ...source,
      selected,
      initialWidth,
      initialHeight,
      ...(measured ? { measured } : {}),
    };
    cache.set(source.id, { source, selected, projected });
    return projected;
  });
  return { nodes, cache };
}
