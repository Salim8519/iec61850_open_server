import { useEffect, useState } from 'react';
import { LINK_MAP, liveFeed } from '../../api/polls';
import type { GooseLinkMap, MmsClient } from '../../api/signalLayers';
import { EMPTY_MAP } from '../../equipment/signalLayerStore';

export const MMS_ACCESS = '/mms/access';
const NO_CLIENTS: MmsClient[] = [];

/** The project's GOOSE link map for the Devices place; refreshed every 5 s while running. */
export function useGooseLinks(revision: number, running: boolean): GooseLinkMap {
  const [map, setMap] = useState<GooseLinkMap>(EMPTY_MAP);
  useEffect(() => liveFeed.subscribe<GooseLinkMap>(LINK_MAP, running ? 5000 : null, setMap), [revision, running]);
  return map;
}

/** MMS clients of one simulated IED, refreshed every 3 s while running (shared live feed). */
export function useMmsClients(iedId: string, running: boolean): MmsClient[] {
  const [clients, setClients] = useState<MmsClient[]>(NO_CLIENTS);
  useEffect(() => {
    if (!running) { setClients(NO_CLIENTS); return undefined; }
    return liveFeed.subscribe<{ ieds: Record<string, MmsClient[]> }>(MMS_ACCESS, 3000,
      result => setClients(result.ieds[iedId] || NO_CLIENTS));
  }, [iedId, running]);
  return clients;
}
