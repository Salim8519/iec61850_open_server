import { useCallback, useEffect, useSyncExternalStore } from 'react';
import { getPreferences, setPreferences } from '../../../api/preferences';
import { LINK_MAP, polls } from '../../../api/polls';
import { getMmsAccess, type GooseLinkMap } from '../../../api/signalLayers';
import type { Layers, SignalLayerStore } from '../../../equipment/signalLayerStore';

/** The two SLD switches, remembered by the backend (preferences sld_goose_layer / sld_readers_layer),
 * and the GOOSE link map, refreshed on project changes and every 5 s while running. */
export function useSignalLayers(store: SignalLayerStore, revision: number, running: boolean) {
  const state = useSyncExternalStore(store.subscribe, store.get, store.get);
  useEffect(() => {
    getPreferences().then(prefs => store.setLayers({ goose: !!prefs.sld_goose_layer, readers: !!prefs.sld_readers_layer }))
      .catch(() => {});
  }, [store]);
  const toggle = useCallback((layer: keyof Layers) => {
    const next = { ...store.get().layers, [layer]: !store.get().layers[layer] };
    store.setLayers(next);
    setPreferences({ sld_goose_layer: next.goose ? 1 : 0, sld_readers_layer: next.readers ? 1 : 0 }).catch(() => {});
  }, [store]);
  const active = state.layers.goose || state.layers.readers;
  useEffect(() => {
    if (!active) return;
    return polls.subscribe<GooseLinkMap>(LINK_MAP, running ? 5000 : null, map => store.setMap(map));
  }, [active, revision, running, store]);
  useEffect(() => {
    if (state.layers.readers && running) getMmsAccess().then(result => store.setClients(result.ieds)).catch(() => {});
  }, [state.layers.readers, running, store]);
  return { state, toggle };
}
