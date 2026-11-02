import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Project } from '../api/types';

/** The engine's answer to GET /project/history (ADR-0135). */
export type HistoryStatus = {
  can_undo: boolean; can_redo: boolean; undo: string; redo: string; running: boolean; limit: number;
};

/** Undo and Redo for the whole project. The engine keeps the steps; this hook only shows its
 *  answer (read again whenever the project revision or run state changes) and sends the click.
 *  A pending diagram edit is saved first, so Undo always takes back what is on screen. */
export function useHistory(project: Project | null, perform: (task: () => Promise<unknown>, label?: string) => Promise<void>,
  prepare: () => Promise<void>) {
  const [status, setStatus] = useState<HistoryStatus | null>(null);
  const [available, setAvailable] = useState(true);
  const revision = project?.revision;
  const runtime = project?.runtime;
  const session = project?.session_id;
  useEffect(() => {
    if (revision === undefined) return;
    let current = true;
    api<HistoryStatus>('/project/history').then(next => { if (current) { setStatus(next); setAvailable(true); } })
      .catch(() => { if (current) setAvailable(false); });  // an engine without Undo hides the buttons
    return () => { current = false; };
  }, [revision, runtime, session]);
  const step = useCallback((direction: 'undo' | 'redo') => perform(async () => {
    await prepare();
    setStatus(await api<HistoryStatus>(`/project/${direction}`, 'POST'));
  }, direction === 'undo' ? 'Undoing…' : 'Redoing…'), [perform, prepare]);
  const undo = useCallback(() => step('undo'), [step]);
  const redo = useCallback(() => step('redo'), [step]);
  return { status, available, canUndo: !!status?.can_undo, canRedo: !!status?.can_redo, undo, redo };
}

export type ProjectHistory = ReturnType<typeof useHistory>;
