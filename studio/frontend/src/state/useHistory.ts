import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import { liveFeed } from '../api/polls';
import { useAskAgain } from '../api/useAskAgain';
import type { Project } from '../api/types';

/** The engine's answer to GET /project/history (ADR-0135). */
export type HistoryStatus = {
  can_undo: boolean; can_redo: boolean; undo: string; redo: string; running: boolean; limit: number;
};

export const HISTORY = '/project/history';

/** Undo and Redo for the whole project. The engine keeps the steps; this hook only shows its
 *  answer and sends the click. The Go engine pushes the answer whenever it changes (topic
 *  project.history, change 0224); a polled engine is asked again when the project revision or
 *  run state changes. A pending diagram edit is saved first, so Undo always takes back what is on screen. */
export function useHistory(project: Project | null, perform: (task: () => Promise<unknown>, label?: string) => Promise<void>,
  prepare: () => Promise<void>) {
  const [status, setStatus] = useState<HistoryStatus | null>(null);
  const [available, setAvailable] = useState(true);
  const loaded = !!project;
  useEffect(() => {
    if (!loaded) return undefined;
    return liveFeed.subscribe<HistoryStatus>(HISTORY, null, next => { setStatus(next); setAvailable(true); },
      () => setAvailable(false));  // an engine without Undo hides the buttons
  }, [loaded]);
  useAskAgain(loaded ? HISTORY : null, `${project?.revision}|${project?.runtime}|${project?.session_id}`);
  const step = useCallback((direction: 'undo' | 'redo') => perform(async () => {
    await prepare();
    setStatus(await api<HistoryStatus>(`/project/${direction}`, 'POST'));
  }, direction === 'undo' ? 'Undoing…' : 'Redoing…'), [perform, prepare]);
  const undo = useCallback(() => step('undo'), [step]);
  const redo = useCallback(() => step('redo'), [step]);
  return { status, available, canUndo: !!status?.can_undo, canRedo: !!status?.can_redo, undo, redo };
}

export type ProjectHistory = ReturnType<typeof useHistory>;
