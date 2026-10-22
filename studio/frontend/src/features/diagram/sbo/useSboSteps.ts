import { useSyncExternalStore } from 'react';
import { getPreferences, setPreferences } from '../../../api/preferences';

/** Select before operate on the SLD and point panels (change 0173): one click by default, like a
 *  station HMI (ScadaBench still sends Select, then Operate, on the wire); "step by step" shows the
 *  selection and its countdown for testing. One switch for the app, kept in the data folder. */
let steps = false;
let loaded = false;
const listeners = new Set<() => void>();
const notify = () => listeners.forEach(listener => listener());

function subscribe(listener: () => void) {
  listeners.add(listener);
  if (!loaded) {
    loaded = true;
    getPreferences().then(prefs => { steps = !!prefs.sld_sbo_steps; notify(); }).catch(() => { loaded = false; });
  }
  return () => { listeners.delete(listener); };
}

export function setSboSteps(next: boolean) {
  steps = next;
  notify();
  setPreferences({ sld_sbo_steps: next ? 1 : 0 }).catch(() => {});
}

/** True when select before operate is shown step by step. */
export function useSboSteps() {
  return useSyncExternalStore(subscribe, () => steps, () => steps);
}
