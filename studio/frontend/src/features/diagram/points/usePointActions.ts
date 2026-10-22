import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../../../api/client';
import type { PanelPoint } from '../../../api/types';
import { HEALTHY_BODY, healthySentence, pointRoute, sendBody, setBody, type CommandValue, type HealthyAnswer, type PointBody }
  from '../../../equipment/pointRequests';
import { POINT_COPY } from '../../../copy/pointPanel';

/** What happened to one row after its last request. A failure stays until the next try; a success fades. */
export type Outcome = { state: 'busy' | 'done' | 'failed'; text: string };
const FADE_MS = 4000;

/** Requests of one point panel: set a value, send a command, set all healthy. Each row keeps its own
 *  answer so two quick changes never hide each other. */
export function usePointActions(nodeId: string) {
  const [outcomes, setOutcomes] = useState<Record<string, Outcome>>({});
  const [note, setNote] = useState('');
  const [working, setWorking] = useState(false);
  const timers = useRef(new Map<string, ReturnType<typeof setTimeout>>());
  useEffect(() => () => timers.current.forEach(timer => clearTimeout(timer)), []);
  const show = useCallback((id: string, outcome: Outcome | null) => {
    clearTimeout(timers.current.get(id));
    setOutcomes(current => {
      const { [id]: _, ...rest } = current;
      return outcome ? { ...rest, [id]: outcome } : rest;
    });
    if (outcome?.state === 'done') timers.current.set(id, setTimeout(() => show(id, null), FADE_MS));
  }, []);
  const run = useCallback(async (point: PanelPoint, body: PointBody, done: string) => {
    show(point.id, { state: 'busy', text: 'Working…' });
    try {
      await api(pointRoute(nodeId), 'POST', body);
      show(point.id, { state: 'done', text: POINT_COPY.sent(done) });
    } catch (error) {
      show(point.id, { state: 'failed', text: error instanceof Error ? error.message : String(error) });
    }
  }, [nodeId, show]);
  const set = useCallback((point: PanelPoint, value: boolean | number, done: string) =>
    run(point, setBody(point, value), done), [run]);
  const send = useCallback((point: PanelPoint, value: CommandValue, done: string) =>
    run(point, sendBody(point, value), done), [run]);
  const healthy = useCallback(async () => {
    setWorking(true); setNote('');
    try {
      const answer = await api<HealthyAnswer>(pointRoute(nodeId), 'POST', HEALTHY_BODY);
      for (const result of answer.results) {
        show(result.row, result.ok ? { state: 'done', text: POINT_COPY.sent('set to healthy') }
          : { state: 'failed', text: result.error || 'The IED did not confirm the new value.' });
      }
      setNote(healthySentence(answer));
    } catch (error) {
      setNote(error instanceof Error ? error.message : String(error));
    } finally { setWorking(false); }
  }, [nodeId, show]);
  return { outcomes, note, working, set, send, healthy, clearNote: () => setNote('') };
}
