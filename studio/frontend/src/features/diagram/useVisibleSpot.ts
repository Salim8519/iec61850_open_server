import { useCallback } from 'react';
import { useReactFlow, useStoreApi } from '@xyflow/react';
import type { EquipmentNode } from '../../api/types';
import { freeSpot, NEW_SIZE, type Point } from '../../equipment/placeNew';

/** A free spot in the middle of what the canvas shows now, for a symbol added by a click (change 0176).
 *  Used inside the canvas (the toolbar), where React Flow's view and measured sizes are known. */
export function useVisibleSpot(): () => Point {
  const flow = useReactFlow<EquipmentNode>();
  const store = useStoreApi();
  return useCallback(() => {
    const { width, height } = store.getState();
    const view = flow.getViewport();
    const center = { x: (width / 2 - view.x) / view.zoom, y: (height / 2 - view.y) / view.zoom };
    const boxes = flow.getNodes().map(node => ({ x: node.position.x, y: node.position.y,
      width: node.measured?.width ?? NEW_SIZE.width, height: node.measured?.height ?? NEW_SIZE.height }));
    return freeSpot(boxes, center);
  }, [flow, store]);
}
