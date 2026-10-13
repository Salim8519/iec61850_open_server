import type { NodeChange } from '@xyflow/react';
import type { EquipmentNode } from '../api/types';

/** View-only React Flow measurements per node; never persisted or recorded in history. */
export type MeasuredStore = Map<string, { width: number; height: number }>;

export function rememberMeasured(store: MeasuredStore, changes: NodeChange<EquipmentNode>[]): void {
  for (const change of changes) {
    if (change.type === 'remove') store.delete(change.id);
    else if (change.type === 'dimensions' && change.dimensions) {
      const { width, height } = change.dimensions;
      const known = store.get(change.id);
      if (!known || known.width !== width || known.height !== height) store.set(change.id, { width, height });
    }
  }
}
