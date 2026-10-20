import { useCallback, useSyncExternalStore } from 'react';
import { liveData } from './liveData';

const noop = () => () => {};

/** One data point's live value ("iedId:reference"); undefined until the IED reports it.
 * Only this component re-renders when that value changes, at most once per frame. */
export function useSignalValue(key: string | null | undefined): unknown {
  const store = liveData().values;
  const subscribe = useCallback((listener: () => void) => key ? store.subscribe(key, listener) : noop(), [key, store]);
  const snapshot = useCallback(() => key ? store.get(key) : undefined, [key, store]);
  return useSyncExternalStore(subscribe, snapshot, snapshot);
}

/** How many Activity events are kept (at most 500), or how many of one kind arrived. */
export function useEventCount(kind?: string): number {
  const log = liveData().log;
  const snapshot = useCallback(() => kind ? log.count(kind) : log.size(), [kind, log]);
  return useSyncExternalStore(log.subscribe, snapshot, snapshot);
}

/** Changes whenever the Activity log does (once per frame at most). */
export function useEventLogVersion(): number {
  const log = liveData().log;
  return useSyncExternalStore(log.subscribe, log.version, log.version);
}
