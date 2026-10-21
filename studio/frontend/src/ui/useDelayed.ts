import { useEffect, useState } from 'react';

/** The value once it has stayed set for `ms`; empty again at once when it clears.
 *  Used so "Working…" words appear only for waits a person would notice. */
export function useDelayed(value: string, ms: number): string {
  const [shown, setShown] = useState('');
  useEffect(() => {
    if (!value) { setShown(''); return; }
    const timer = window.setTimeout(() => setShown(value), ms);
    return () => window.clearTimeout(timer);
  }, [value, ms]);
  return value ? shown : '';
}
