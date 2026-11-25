import { useEffect, useReducer } from 'react';
import { api } from '../../api/client';
import type { CommandOptions } from '../../api/types';
import type { CommandValue } from '../../equipment/pointRequests';

/** One step of a real IED's command from Devices (ADR-0149). The Go engine checks it again, sends it
 *  with the device's own control model and answers in plain words; the browser only shows that. */
export type StepAction = 'select' | 'operate' | 'cancel';
export type StepAnswer = { ok: boolean; action: StepAction; summary: string; model: string | null; expires_ms: number;
  observed: string | null; changed: boolean };
type StepBody = { action: StepAction; ref: string; value?: CommandValue; options: CommandOptions; confirm?: boolean;
  selected?: boolean };

export const sendStep = (iedId: string, body: StepBody) =>
  api<StepAnswer>(`/live/${encodeURIComponent(iedId)}/command`, 'POST', body);

/** What Devices sends unless the person changes it: interlock check on and station control, exactly
 *  what 0.1.36 sent; Test follows the relay's Test mode until the person sets it (ADR-0149). */
export const DEVICE_OPTIONS: CommandOptions = { test: false, interlock_check: true, synchro_check: false, orcat: 2, orident: '' };

const chosen = new Map<string, CommandOptions>();
const listeners = new Set<() => void>();

/** The command options of one real IED for this session, shared by the pop-up and the Data points rows. */
export function useDeviceOptions(iedId: string, testMode: boolean): [CommandOptions, (next: CommandOptions) => void] {
  const [, repaint] = useReducer((count: number) => count + 1, 0);
  useEffect(() => { listeners.add(repaint); return () => { listeners.delete(repaint); }; }, []);
  const options = chosen.get(iedId) ?? { ...DEVICE_OPTIONS, test: testMode };
  return [options, next => { chosen.set(iedId, next); listeners.forEach(listener => listener()); }];
}

/** The answer in one sentence, with where the feedback went. */
export const answerText = (answer: StepAnswer): string =>
  answer.observed ? `${answer.summary} Feedback: ${answer.observed}.` : answer.summary;
