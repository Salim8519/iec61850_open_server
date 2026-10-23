import { useEffect, useMemo, type ReactNode } from 'react';
import type { Ied } from '../../../api/types';
import { GooseLeaves, IedRecords, IedStatus, iedRecords } from '../../../equipment/iedContext';
import { watchIedLive } from '../../../equipment/iedStatus';
import { createKeyedStore } from '../../../equipment/keyedStore';
import type { SignalLayerStore } from '../../../equipment/signalLayerStore';
import { useGooseMembers } from '../signals/useGooseMembers';

/** What IED components and GOOSE-aware rows read (ADR-0112): the stored IEDs' names and GOOSE
 * counts, their live words (a keyed store fed from the signal layers, so traffic re-renders only a
 * card whose words changed) and the exact leaves of every GOOSE message. */
export function IedProviders({ store, ieds, revision, children }: {
  store: SignalLayerStore; ieds: Ied[]; revision: number; children: ReactNode;
}) {
  const key = JSON.stringify(ieds.map(ied => [ied.id, ied.name, ied.role, ied.goose?.publishes?.length ?? 0,
    ied.goose?.subscribes?.length ?? 0]));
  const records = useMemo(() => iedRecords(ieds), [key]);
  const status = useMemo(() => createKeyedStore<string>(), []);
  useEffect(() => watchIedLive(store, status), [store, status]);
  const leaves = useGooseMembers(store, revision);
  return <IedRecords.Provider value={records}><IedStatus.Provider value={status}>
    <GooseLeaves.Provider value={leaves}>{children}</GooseLeaves.Provider></IedStatus.Provider></IedRecords.Provider>;
}
