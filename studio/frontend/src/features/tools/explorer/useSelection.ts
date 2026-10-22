import { useEffect, useState } from 'react';
import { secondsLeft, type Selection } from './selectionTime';

/** The device's selection of one control and its countdown. When the time runs out the selection is
 *  dropped here too and `expired` is called, because the device has already let it go. */
export function useSelection(expired: () => void) {
  const [selection, setSelection] = useState<Selection | null>(null);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!selection) return;
    const timer = window.setInterval(() => {
      const time = Date.now();
      setNow(time);
      if (time >= selection.until) { setSelection(null); expired(); }
    }, 250);
    return () => window.clearInterval(timer);
  }, [selection]);
  return {
    selection, left: selection ? secondsLeft(selection, now) : 0,
    start: (label: string, value: unknown, ms: number) => { setNow(Date.now()); setSelection({ label, value, until: Date.now() + ms }); },
    clear: () => setSelection(null),
  };
}
