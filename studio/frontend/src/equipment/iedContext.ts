import { createContext, useCallback, useContext, useMemo, useSyncExternalStore } from 'react';
import type { Ied, IedRole } from '../api/types';
import { createKeyedStore } from './keyedStore';
import type { IedLive } from './iedStatus.ts';
import type { LeafIndex } from './gooseLeaves.ts';

/** What an IED component card shows without asking the backend (ADR-0112): the stored IED's name,
 * role and how many GOOSE messages its files send and receive. */
export type IedRecord = { name: string; role: IedRole; sends: number; gets: number };
export const IedRecords = createContext<Record<string, IedRecord>>({});
/** Persistent live words per IED id (GOOSE problems, readers), read key by key: GOOSE traffic never
 * re-renders a card, and a card re-renders only when its own words change. */
export const IedStatus = createContext(createKeyedStore<string>());
/** Exact leaves of every GOOSE message (Go's GET /goose/members); changes only with the project, so
 * point panel rows can show their "GOOSE" tag without traffic re-rendering them. */
export const GooseLeaves = createContext<LeafIndex>(new Map());

export function iedRecords(ieds: Ied[]): Record<string, IedRecord> {
  return Object.fromEntries(ieds.map(ied => [ied.id, { name: ied.name, role: ied.role,
    sends: ied.goose?.publishes?.length ?? 0, gets: ied.goose?.subscribes?.length ?? 0 }]));
}

/** The card's live words, or undefined when there is nothing to say. */
export function useIedLive(iedId: string | null | undefined): IedLive | undefined {
  const store = useContext(IedStatus);
  const key = iedId ?? '';
  const subscribe = useCallback((listener: () => void) => store.subscribe(key, listener), [key, store]);
  const snapshot = useCallback(() => store.get(key), [key, store]);
  const text = useSyncExternalStore(subscribe, snapshot, snapshot);
  return useMemo(() => text ? JSON.parse(text) as IedLive : undefined, [text]);
}
