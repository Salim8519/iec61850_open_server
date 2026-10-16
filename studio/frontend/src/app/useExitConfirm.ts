// The packaged app window stops ScadaBench and all its IEDs when it closes, so closing
// it asks first, like any desktop app. Browser tabs (development, or no Edge/Chrome)
// do not stop anything when closed and therefore never ask.
import { useEffect } from 'react';

const KEY = 'scadabench-app-window';

/** Call before the session fragment is cleared: the launcher marks its own window with window=app. */
export function rememberAppWindow(fragment: URLSearchParams) {
  if (fragment.get('window') !== 'app') return;
  try { sessionStorage.setItem(KEY, '1'); } catch { /* storage blocked: no prompt, app still works */ }
}

function isAppWindow() {
  try { return sessionStorage.getItem(KEY) === '1'; } catch { return false; }
}

export function useExitConfirm() {
  useEffect(() => {
    if (!isAppWindow()) return;
    const ask = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = ''; };
    window.addEventListener('beforeunload', ask);
    return () => window.removeEventListener('beforeunload', ask);
  }, []);
}
