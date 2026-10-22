import { useSyncExternalStore } from 'react';
import { getPreferences, setPreferences } from '../../../api/preferences';
import { createSavedSwitch } from '../../../equipment/savedSwitch';

/** Select before operate on the SLD and point panels (change 0173): one click by default, like a
 *  station HMI (ScadaBench still sends Select, then Operate, on the wire); "step by step" shows the
 *  selection and its countdown for testing. One switch for the app, kept in the data folder; a
 *  failed save moves it back and says why (change 0176). */
const steps = createSavedSwitch(next => setPreferences({ sld_sbo_steps: next ? 1 : 0 }));
let loaded = false;

function subscribe(listener: () => void) {
  const leave = steps.subscribe(listener);
  if (!loaded) {
    loaded = true;
    getPreferences().then(prefs => steps.load(!!prefs.sld_sbo_steps)).catch(() => { loaded = false; });
  }
  return leave;
}

export function setSboSteps(next: boolean) { void steps.set(next); }

/** True when select before operate is shown step by step. */
export function useSboSteps() {
  return useSyncExternalStore(subscribe, steps.get, steps.get);
}

/** Why the last change of the switch was not saved ('' when it was). */
export function useSboStepsError() {
  return useSyncExternalStore(subscribe, steps.error, steps.error);
}
