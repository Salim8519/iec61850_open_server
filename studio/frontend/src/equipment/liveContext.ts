import { createContext, useCallback, useContext, useSyncExternalStore } from 'react';
import { EquipmentValueStore } from './liveValues';
import { createKeyedStore } from './keyedStore';

/** Shared live equipment values; canvas and inspector read the same store. */
export const LiveValues = createContext(new EquipmentValueStore());
/** IED id -> name, for display-only shortening of references. */
export const IedNames = createContext<Record<string, string>>({});
/** IED ids that are stopped while other IEDs run (change 0106); empty when nothing or everything runs. */
export const StoppedIeds = createContext<ReadonlySet<string>>(new Set());
/** Equipment id -> mode name when its IED logical device is not in normal service, read key by key. */
export const ModeBadges = createContext(createKeyedStore<string>());

/** Subscribe to one key of the live equipment store. */
export function useLiveValue(key: string): unknown {
  const store = useContext(LiveValues);
  const subscribe = useCallback((listener: () => void) => store.subscribe(key, listener), [key, store]);
  const snapshot = useCallback(() => store.get(key), [key, store]);
  return useSyncExternalStore(subscribe, snapshot, snapshot);
}

/** The mode badge of one equipment; only this equipment re-renders when it changes. */
export function useModeBadge(id: string): string | undefined {
  const store = useContext(ModeBadges);
  const subscribe = useCallback((listener: () => void) => store.subscribe(id, listener), [id, store]);
  const snapshot = useCallback(() => store.get(id), [id, store]);
  return useSyncExternalStore(subscribe, snapshot, snapshot);
}
