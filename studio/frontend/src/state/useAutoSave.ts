import { useEffect, useMemo, useRef, useState } from 'react';
import { createAutoSaver, IDLE, type SaveState } from './autoSave';

/** The React side of "save always" (state/autoSave.ts). `run` is the latest save function; `held` stops
 *  sending while true (the gateway runs). The saver outlives the component on purpose: a request or a
 *  waiting value still finishes after the page is left, so leaving a tab never loses a valid edit. */
export function useAutoSave<T>(run: (value: T) => Promise<unknown>, held = false) {
  const [state, setState] = useState<SaveState>(IDLE);
  const latest = useRef(run);
  latest.current = run;
  const alive = useRef(true);
  const saver = useMemo(() => createAutoSaver<T>(value => latest.current(value), next => { if (alive.current) setState(next); }), []);
  useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);
  useEffect(() => saver.hold(held), [held, saver]);
  return { state, push: saver.push, drop: saver.drop, retry: saver.retry, pending: saver.pending, settled: saver.settled };
}
