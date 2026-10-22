import { useCallback, useEffect, useReducer, useRef, useState } from 'react';
import { SBO_IDLE, sboReducer, secondsLeft, type SboEvent, type SboState } from '../../../equipment/selectOperate';

/** The three requests of one select-before-operate command. `choice` is what the person picked
 *  (Open, Close, a number, a label, Raise …); the Go engine holds the selection and its time. */
export type SelectOperateSteps<C> = {
  select: (choice: C) => Promise<{ expires_ms: number }>;
  /** Sends the selected command; resolves with words for the outcome ('' when the caller shows it). */
  operate: (choice: C) => Promise<string | void>;
  cancel: (choice: C) => Promise<unknown>;
};

const TICK_MS = 250;
const messageOf = (error: unknown) => error instanceof Error ? error.message : String(error);

/** Select → countdown → Operate or Cancel, shared by the SLD switches, the point panel and the tap
 *  changer (exported for ADR-0108 users). A selection that is still held when the component goes
 *  away is given back. */
export function useSelectOperate<C>(steps: SelectOperateSteps<C>) {
  const [state, dispatch] = useReducer(
    (current: SboState<C>, event: SboEvent<C>) => sboReducer(current, event), SBO_IDLE as SboState<C>);
  const [now, setNow] = useState(() => Date.now());
  const stepsRef = useRef(steps);
  stepsRef.current = steps;
  const stateRef = useRef(state);
  stateRef.current = state;
  useEffect(() => {
    if (state.phase !== 'selected') return undefined;
    const timer = setInterval(() => { const at = Date.now(); setNow(at); dispatch({ type: 'tick', now: at }); }, TICK_MS);
    return () => clearInterval(timer);
  }, [state.phase]);
  useEffect(() => () => {
    const held = stateRef.current;
    if (held.phase === 'selected' && held.choice !== null) stepsRef.current.cancel(held.choice).catch(() => undefined);
  }, []);
  const select = useCallback(async (choice: C) => {
    if (stateRef.current.phase !== 'idle') return;
    dispatch({ type: 'select', choice });
    try {
      const answer = await stepsRef.current.select(choice);
      const at = Date.now();
      setNow(at);
      dispatch({ type: 'selected', windowMs: answer.expires_ms, now: at });
    } catch (error) { dispatch({ type: 'failed', message: messageOf(error) }); }
  }, []);
  const operate = useCallback(async () => {
    const held = stateRef.current;
    if (held.phase !== 'selected' || held.choice === null) return;
    dispatch({ type: 'operate' });
    try { dispatch({ type: 'operated', message: (await stepsRef.current.operate(held.choice)) || '' }); }
    catch (error) { dispatch({ type: 'failed', message: messageOf(error) }); }
  }, []);
  const cancel = useCallback(async () => {
    const held = stateRef.current;
    if (held.phase !== 'selected' || held.choice === null) return;
    dispatch({ type: 'cancel' });
    try { await stepsRef.current.cancel(held.choice); dispatch({ type: 'cancelled', message: '' }); }
    catch (error) { dispatch({ type: 'cancelRefused', message: messageOf(error) }); } // still selected (change 0176)
  }, []);
  const clear = useCallback(() => dispatch({ type: 'clear' }), []);
  return { state, left: secondsLeft(state, now), select, operate, cancel, clear };
}

export type SelectOperate<C> = ReturnType<typeof useSelectOperate<C>>;
