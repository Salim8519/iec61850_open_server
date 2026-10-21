import { useCallback, useEffect, useState } from 'react';
import { getPreferences, saveListFilter } from '../api/preferences';

// Full view (ADR-0095): one big list fills the window. Whether a list was in full view is
// remembered per list like its filters, in the backend preferences under the list name
// "full-view" ({listKey: true/false}), so it survives a restart and a changed port.
const STORE = 'full-view';
type Stored = Record<string, string | boolean>;
let remembered: Promise<Stored> | null = null;
const load = () => remembered ??= getPreferences()
  .then((prefs): Stored => ({ ...(prefs.list_filters?.[STORE] ?? {}) })).catch((): Stored => ({}));

/** On/off of one list's full view; `key` is a short camelCase list name such as "gatewayPoints". */
export function useFullView(key: string): [boolean, (on: boolean) => void] {
  const [full, setFull] = useState(false);
  useEffect(() => {
    let alive = true;
    void load().then(all => { if (alive) setFull(all[key] === true); });
    return () => { alive = false; };
  }, [key]);
  const change = useCallback((on: boolean) => {
    setFull(on);
    void load().then(all => {
      if (all[key] === on) return;
      all[key] = on;
      saveListFilter(STORE, all).catch(() => {});  // display-only: a failed save changes nothing else
    });
  }, [key]);
  return [full, change];
}
