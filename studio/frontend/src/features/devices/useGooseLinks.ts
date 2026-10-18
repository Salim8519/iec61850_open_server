import { useEffect, useState } from 'react';
import { getLinkMap, getMmsAccess, type GooseLinkMap, type MmsClient } from '../../api/signalLayers';
import { EMPTY_MAP } from '../../equipment/signalLayerStore';

/** The project's GOOSE link map for the Devices place; refreshed every 5 s while running. */
export function useGooseLinks(revision: number, running: boolean): GooseLinkMap {
  const [map, setMap] = useState<GooseLinkMap>(EMPTY_MAP);
  useEffect(() => {
    let disposed = false;
    const load = () => getLinkMap().then(next => { if (!disposed) setMap(next); }).catch(() => {});
    load();
    const timer = running ? setInterval(load, 5000) : undefined;
    return () => { disposed = true; if (timer) clearInterval(timer); };
  }, [revision, running]);
  return map;
}

/** MMS clients of one simulated IED, refreshed every 3 s while running. */
export function useMmsClients(iedId: string, running: boolean): MmsClient[] {
  const [clients, setClients] = useState<MmsClient[]>([]);
  useEffect(() => {
    if (!running) { setClients([]); return; }
    let disposed = false;
    const load = () => getMmsAccess().then(result => { if (!disposed) setClients(result.ieds[iedId] || []); }).catch(() => {});
    load();
    const timer = setInterval(load, 3000);
    return () => { disposed = true; clearInterval(timer); };
  }, [iedId, running]);
  return clients;
}
