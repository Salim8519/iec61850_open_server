import { useCallback, useEffect, useRef, useState } from 'react';
import { explorerApi, type Attribute, type ReadValue } from '../../../api/tools';
import { bulkReadable } from './readSafety';

/** How often the open data objects are read again while the logical node is shown. */
export const REFRESH_MS = 2000;
const BATCH = 200;  // the engine reads at most 200 attributes in one request

/** Values of one logical node's open data objects: read once when opened, read again every 2 s
 *  while shown (never a command field: SBO would select, change 0177), and all at once on demand.
 *  A failed automatic read pauses the refresh until the next manual read, so a lost device is
 *  reported once instead of every 2 s. */
export function useNodeReads(reference: string, fail: (message: string) => void) {
  const [values, setValues] = useState<Record<string, ReadValue>>({});
  const [reading, setReading] = useState('');  // the object being read by hand, or '*' for all
  const [paused, setPaused] = useState(false);
  const inFlight = useRef(false);
  const generation = useRef(0);
  useEffect(() => { generation.current += 1; setValues({}); setPaused(false); }, [reference]);

  const read = useCallback((label: string, groups: Attribute[][], automatic = false) => {
    const attributes = groups.flatMap(group => bulkReadable(group));
    if (!attributes.length || (automatic && inFlight.current)) return;
    const own = generation.current;
    inFlight.current = true;
    if (!automatic) { setReading(label); setPaused(false); }
    (async () => {
      for (let start = 0; start < attributes.length; start += BATCH) {
        const answer = await explorerApi.read(attributes.slice(start, start + BATCH));
        if (own !== generation.current) return;
        setValues(current => ({ ...current, ...answer.values }));
      }
    })().catch(error => {
      if (own !== generation.current) return;
      if (automatic) setPaused(true);
      fail(error.message);
    }).finally(() => { inFlight.current = false; if (!automatic) setReading(''); });
  }, [fail]);
  return { values, reading, paused, read };
}

/** Reads the open objects again every REFRESH_MS while the window is visible and not paused. */
export function useAutoRefresh(refresh: () => void, active: boolean) {
  const latest = useRef(refresh);
  latest.current = refresh;
  useEffect(() => {
    if (!active) return;
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') latest.current();
    }, REFRESH_MS);
    return () => window.clearInterval(timer);
  }, [active]);
}
