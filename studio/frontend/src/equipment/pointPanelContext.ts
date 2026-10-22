import { createContext, useCallback, useContext, useMemo, useSyncExternalStore } from 'react';
import { LiveValues } from './liveContext';

/** Opens the popup of a point panel (ADR-0102); the diagram page provides it. */
export const OpenPointPanel = createContext<(nodeId: string) => void>(() => {});
/** True while a run locks the drawing; the canvas provides it so a card knows when values can exist. */
export const CanvasLocked = createContext(false);

/** The live values of many keys at once ("node:point" keys of the shared equipment store). The answer
 *  changes only when one of the values does; a card with 300 points still re-renders once per change. */
export function useLiveValues(keys: readonly string[]): unknown[] {
  const store = useContext(LiveValues);
  const signature = keys.join('\n');
  const subscribe = useCallback((listener: () => void) => {
    const stops = signature ? signature.split('\n').map(key => store.subscribe(key, listener)) : [];
    return () => stops.forEach(stop => stop());
  }, [signature, store]);
  const snapshot = useCallback(() => JSON.stringify(keys.map(key => store.get(key) ?? null)),
    [signature, store]);
  const text = useSyncExternalStore(subscribe, snapshot, snapshot);
  return useMemo(() => JSON.parse(text) as unknown[], [text]);
}
