import { useEffect, useRef, useState } from 'react';
import { MASTER_COMPARE, masterListApi, type Comparison, type ListPoint, type MasterList } from '../../../../../api/masterList';
import { liveFeed } from '../../../../../api/polls';
import { useAutoSave } from '../../../../../state/useAutoSave';

/** The master point list and the engine's comparison with what the device sent (ADR-0110).
 *  Every change is shown at once and saved at once (save always, ADR-0101): one request at a time, the
 *  latest list wins. Whole-list actions (add, copy, import, remove) keep the list before them for Undo.
 *  A list the engine refuses stays on the page with the engine's reason until it is fixed or undone. */
export function useMasterList(protocol: string, fail: (message: string) => void) {
  const [list, setList] = useState<MasterList | null>(null);
  const [comparison, setComparison] = useState<Comparison | null>(null);
  const [undo, setUndo] = useState<ListPoint[] | null>(null);
  const [problem, setProblem] = useState('');
  const edits = useRef(0);
  const autosave = useAutoSave<{ points: ListPoint[]; edit: number }>(sent => masterListApi.save(sent.points).then(saved => {
    if (sent.edit !== edits.current) return; // a newer edit waits behind this save
    setList(saved); setProblem('');
    liveFeed.refresh(MASTER_COMPARE);
  }).catch(error => {
    if (sent.edit === edits.current) setProblem(error.message);
    throw error;
  }));
  useEffect(() => {
    edits.current += 1;
    setList(null); setUndo(null); setProblem('');
    masterListApi.list().then(setList).catch(error => fail(error.message));
    return liveFeed.subscribe<Comparison>(MASTER_COMPARE, 1000, setComparison, error => fail(error.message));
  }, [protocol]);
  /** Show and save a new list; `back` is the list before a whole-list action (Undo puts it back). */
  const edit = (points: ListPoint[], back: ListPoint[] | null = null) => {
    if (!list) return;
    edits.current += 1;
    setList({ ...list, points }); setUndo(back); setProblem('');
    autosave.push({ points, edit: edits.current });
  };
  const restore = () => { if (undo) { const back = undo; edit(back, null); } };
  const setCheck = (on: boolean) => {
    if (list) setList({ ...list, check_ied: on });
    masterListApi.checkIed(on).then(() => liveFeed.refresh(MASTER_COMPARE)).catch(error => fail(error.message));
  };
  return { list, comparison, edit, undo, restore, problem, state: autosave.state, setCheck };
}
