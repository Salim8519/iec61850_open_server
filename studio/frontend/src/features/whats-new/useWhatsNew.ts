import { useCallback, useEffect, useState } from 'react';
import { fetchCapabilities } from '../../api/capabilities';
import { getPreferences, setPreferences } from '../../api/preferences';
import { versionCode } from '../../copy/releaseNotes';
import { whatsNewChoice } from './whatsNewChoice';

/** Only the Go engine stores whats_new_seen; it is also the only one that answers capabilities. */
const engineRemembers = () => fetchCapabilities((input, init) => fetch(input, init), window.location.origin)
  .then(capabilities => capabilities !== null);

/** Opens What's new once after an update; `open` shows it again from Help (ADR-0148). Under the
 *  frozen Python engine, which cannot remember it, it opens only from Help. */
export function useWhatsNew(current: string | undefined) {
  const [shown, setShown] = useState(false);
  const remember = useCallback((version: string) => {
    const code = versionCode(version);
    if (code !== null) setPreferences({ whats_new_seen: code }).catch(() => {});
  }, []);
  useEffect(() => {
    if (!current) return;
    let live = true;
    Promise.all([getPreferences(), engineRemembers()]).then(([prefs, remembers]) => {
      if (!live || !remembers) return;
      const choice = whatsNewChoice(prefs, current);
      if (choice === 'show') setShown(true);
      else if (choice === 'mark') remember(current);
    }).catch(() => {});
    return () => { live = false; };
  }, [current, remember]);
  const close = useCallback(() => { setShown(false); if (current) remember(current); }, [current, remember]);
  return { shown, open: useCallback(() => setShown(true), []), close };
}
