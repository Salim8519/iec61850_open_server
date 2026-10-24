import { useCallback } from 'react';
import { useReactFlow, useStoreApi } from '@xyflow/react';
import type { EquipmentNode } from '../../api/types';
import { freeSpot, NEW_SIZE, type Point } from '../../equipment/placeNew';
import { toolbarSpace } from './fitPadding';

/** A free spot in the middle of what the canvas shows now, for a symbol added by a click (change 0176).
 *  Used inside the canvas (the toolbar), where React Flow's view and measured sizes are known. The floating
 *  toolbar's strip counts as taken, so a new card never lands hidden under it (change 0187). */
export function useVisibleSpot(): () => Point {
  const flow = useReactFlow<EquipmentNode>();
  const store = useStoreApi();
  return useCallback(() => {
    const { width, height } = store.getState();
    const view = flow.getViewport();
    const bar = toolbarSpace(document.querySelector('.canvas'));
    const center = { x: (width / 2 - view.x) / view.zoom, y: ((bar + height) / 2 - view.y) / view.zoom };
    const boxes = flow.getNodes().map(node => ({ x: node.position.x, y: node.position.y,
      width: node.measured?.width ?? NEW_SIZE.width, height: node.measured?.height ?? NEW_SIZE.height }));
    if (bar) boxes.push(underToolbar(view, width, bar));
    return freeSpot(boxes, center);
  }, [flow, store]);
}

/** The canvas strip from the top down to the toolbar's lower edge, in flow coordinates. */
function underToolbar(view: { x: number; y: number; zoom: number }, width: number, bar: number) {
  return { x: -view.x / view.zoom, y: -view.y / view.zoom, width: width / view.zoom, height: bar / view.zoom };
}
