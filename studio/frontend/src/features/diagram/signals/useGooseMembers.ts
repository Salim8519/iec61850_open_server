import { useEffect, useState } from 'react';
import { GOOSE_MEMBERS, type GooseMembers } from '../../../api/gooseMessages';
import { liveFeed } from '../../../api/polls';
import type { LeafIndex } from '../../../equipment/gooseLeaves';
import type { SignalLayerStore } from '../../../equipment/signalLayerStore';

/** The exact leaves of every GOOSE message (ADR-0112, Go only), asked again after each project
 * change. The signal layers use them for expected changes, point panel rows for their "GOOSE" tag.
 * The index changes only with the project, so nodes that read it are not re-rendered by traffic. */
export function useGooseMembers(store: SignalLayerStore, revision: number): LeafIndex {
  const [index, setIndex] = useState<LeafIndex>(() => new Map());
  useEffect(() => liveFeed.subscribe<GooseMembers>(GOOSE_MEMBERS, null, members => {
    store.setMembers(members);
    setIndex(store.get().members);
  }, () => {}), [revision, store]);
  return index;
}
