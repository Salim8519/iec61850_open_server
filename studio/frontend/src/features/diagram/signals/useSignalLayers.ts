import { useCallback, useEffect, useSyncExternalStore } from 'react';
import { getPreferences, setPreferences } from '../../../api/preferences';
import { LINK_MAP, liveFeed } from '../../../api/polls';
import type { GooseLinkMap, MmsClient } from '../../../api/signalLayers';
import type { Layers, SignalLayerStore } from '../../../equipment/signalLayerStore';
import { MMS_ACCESS } from '../../devices/useGooseLinks';

/** The two SLD switches, remembered by the backend (preferences sld_goose_layer / sld_readers_layer),
 * and the GOOSE link map, refreshed on project changes and every 5 s while running.
 * The page reads only the switches: live layer data re-renders the overlay, not the diagram. */
export function useSignalLayers(store: SignalLayerStore, revision: number, running: boolean) {
  const layers = useSyncExternalStore(store.subscribe, store.layers, store.layers);
  useEffect(() => {
    getPreferences().then(prefs => store.setLayers({ goose: !!prefs.sld_goose_layer, readers: !!prefs.sld_readers_layer }))
      .catch(() => {});
  }, [store]);
  const toggle = useCallback((layer: keyof Layers) => {
    const next = { ...store.layers(), [layer]: !store.layers()[layer] };
    store.setLayers(next);
    setPreferences({ sld_goose_layer: next.goose ? 1 : 0, sld_readers_layer: next.readers ? 1 : 0 }).catch(() => {});
  }, [store]);
  const active = layers.goose || layers.readers;
  useEffect(() => {
    if (!active) return undefined;
    return liveFeed.subscribe<GooseLinkMap>(LINK_MAP, running ? 5000 : null, map => store.setMap(map));
  }, [active, revision, running, store]);
  useEffect(() => {
    if (!layers.readers || !running) return undefined;
    return liveFeed.subscribe<{ ieds: Record<string, MmsClient[]> }>(MMS_ACCESS, null, result => store.setClients(result.ieds));
  }, [layers.readers, running, store]);
  return { layers, toggle };
}
