import { useStore } from '@xyflow/react';
import { zoomTier, type ZoomTier } from '../../equipment/readability';

/** Current canvas zoom band; re-renders only when the band changes. */
export function useZoomTier(): ZoomTier {
  return useStore(state => zoomTier(state.transform[2]));
}
