import { useEffect, useRef, useState } from 'react';
import { gatewayApi, gatewayPointsPath, problemsOf, type GatewayPoint, type Protocol } from '../../../api/gateway';
import { liveFeed } from '../../../api/polls';
import { SaveLater } from '../../../state/autoSave';
import { useAutoSave } from '../../../state/useAutoSave';
import { mergeValues } from './pointRows';
import { clearDraft, isLockedAnswer, keepDraft, keptDraft, saveProblems, type PointSave } from './pointSave';

/** The SCADA point list of one protocol, shown from the live feed and saved as it changes ("save always").
 *
 *  Every change goes through `edit`: it is shown at once and saved at once. One request runs at a time and
 *  the latest list wins. While the gateway runs the engine refuses a change; the list then waits and is
 *  saved when the gateway stops (also when the Points tab was left meanwhile). A list with problems stays
 *  on the page with its problems listed until it is fixed or discarded. */
export function usePointList(protocol: Protocol, locked: boolean, fail: (message: string) => void) {
  const [rows, setRows] = useState<GatewayPoint[] | null>(null);
  const [dirty, setDirty] = useState(false);
  const [problems, setProblems] = useState<string[]>([]);
  const [undo, setUndo] = useState<GatewayPoint[] | null>(null);
  const dirtyRef = useRef(false);
  dirtyRef.current = dirty;
  const rowsRef = useRef<GatewayPoint[] | null>(null);
  rowsRef.current = rows;
  const lockedRef = useRef(locked);
  lockedRef.current = locked;
  const protocolRef = useRef(protocol);
  protocolRef.current = protocol;
  // The number of the latest edit: a save's answer only replaces the list when no newer edit exists.
  const edits = useRef(0);
  const autosave = useAutoSave<PointSave>(sent => gatewayApi.savePoints(sent.protocol, sent.rows).then(value => {
    if (sent.edit !== edits.current) return; // a newer edit waits behind this save and brings the final list
    clearDraft(sent.protocol);
    setRows(value.points); setDirty(false); setProblems([]);
  }).catch(error => {
    if (isLockedAnswer(error)) throw new SaveLater(); // the gateway started: keep the list, save it when it stops
    if (sent.edit === edits.current) setProblems(saveProblems(error, problemsOf(error)));
    throw error;
  }), locked);
  const path = gatewayPointsPath(protocol);
  const load = () => liveFeed.refresh(path);
  // Shared live feed every second: an unchanged list re-renders nothing (speed lane F).
  useEffect(() => {
    edits.current += 1;
    const waiting = keptDraft(protocol); // a list kept while the gateway ran comes back and is saved
    dirtyRef.current = !!waiting;
    setRows(waiting ?? null); setDirty(!!waiting); setProblems([]); setUndo(null);
    if (waiting) autosave.push({ protocol, rows: waiting, edit: edits.current });
    return liveFeed.subscribe<{ points: GatewayPoint[] }>(path, 1000, value => {
      setRows(current => dirtyRef.current && current ? mergeValues(current, value.points) : value.points);
    }, error => fail(error.message));
  }, [path]);
  useEffect(() => () => { // leaving the tab while the gateway runs keeps an unsaved list for the next visit
    if (dirtyRef.current && lockedRef.current && rowsRef.current) keepDraft(protocolRef.current, rowsRef.current);
  }, []);
  const wasLocked = useRef(locked);
  useEffect(() => { // the gateway stopped: an unsaved list (refused, or with problems) is tried again
    if (wasLocked.current && !locked && dirtyRef.current && rowsRef.current && !autosave.pending()) {
      autosave.push({ protocol, rows: rowsRef.current, edit: edits.current });
    }
    wasLocked.current = locked;
  }, [locked]);
  // The engine says the gateway runs but this page is not locked yet (it is a moment behind): ask again soon.
  useEffect(() => {
    if (autosave.state.phase !== 'waiting' || locked) return;
    const timer = setTimeout(autosave.retry, 2000);
    return () => clearTimeout(timer);
  }, [autosave.state, locked]);
  /** Show and save a new list. `back` is the list before a whole-list action (Undo puts it back). */
  const edit = (next: GatewayPoint[], back: GatewayPoint[] | null = null) => {
    edits.current += 1;
    rowsRef.current = next;
    setRows(next); setDirty(true); setProblems([]); setUndo(back?.length ? back : null);
    autosave.push({ protocol, rows: next, edit: edits.current });
  };
  const update = (index: number, change: Partial<GatewayPoint>) =>
    rows && edit(rows.map((row, at) => at === index ? { ...row, ...change } : row));
  /** Give up a list that cannot be saved (or waits for the gateway) and show the saved one again. */
  const discard = () => {
    autosave.drop(); clearDraft(protocol); edits.current += 1; dirtyRef.current = false;
    setDirty(false); setProblems([]); setUndo(null);
    const ticket = edits.current; // the feed hands out unchanged lists only once, so ask for the saved one directly
    gatewayApi.points(protocol).then(value => { if (ticket === edits.current) setRows(value.points); })
      .catch(error => fail(error.message));
  };
  return { rows, setRows, dirty, problems, setProblems, undo, edit, update, discard, load, state: autosave.state };
}
