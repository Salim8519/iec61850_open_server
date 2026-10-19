import { useCallback, useEffect, useState } from 'react';
import { api } from './client';

/** The quiet daily update check (ADR-0071). The backend asks the server; the browser only shows it. */
export type UpdateStatus = {
  current: string; auto: boolean; checked_at: number | null; checking: boolean;
  error: 'offline' | 'no_server' | 'no_release' | 'server_error' | 'bad_answer' | null;
  available: boolean; important: boolean; dismissed: boolean;
  latest: { version: string; date: string; notes: string; page: string; min_version: string } | null;
};

/** How often the browser re-reads the backend's stored answer (no internet involved). */
const REFRESH_MS = 30 * 60 * 1000;

export function useUpdateStatus() {
  const [status, setStatus] = useState<UpdateStatus | null>(null);
  const load = useCallback(() => api<UpdateStatus>('/app/update').then(setStatus).catch(() => {}), []);
  useEffect(() => {
    load();
    const timer = window.setInterval(load, REFRESH_MS);
    return () => window.clearInterval(timer);
  }, [load]);
  const change = useCallback((changes: { auto?: boolean; dismissed?: string | null }) =>
    api<UpdateStatus>('/app/update', 'PUT', changes).then(setStatus), []);
  const checkNow = useCallback(() => api<UpdateStatus>('/app/update/check', 'POST').then(setStatus), []);
  return { status, change, checkNow };
}

/** Plain words for a check that did not get an answer; never an alarm. */
export function updateProblem(error: UpdateStatus['error']): string {
  if (error === 'offline') return 'Could not reach scadabench.com (no internet is fine; ScadaBench works offline).';
  if (error === 'bad_answer') return 'The answer from scadabench.com could not be checked, so it was ignored.';
  if (error === 'no_release' || error === 'server_error') return 'scadabench.com did not answer the version question. Try later.';
  if (error === 'no_server') return 'No license server address is set, so updates are not checked.';
  return '';
}
