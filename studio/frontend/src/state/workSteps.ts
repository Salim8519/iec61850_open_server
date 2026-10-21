import { useCallback, useEffect, useRef, useState } from 'react';
import type { Step } from '../ui/StepList';

/** A longer task the user waits for (open a project, read or import an SCL file): its title, the
 *  steps in order, which one runs now and how long the finished ones took (change 0150). */
export type Work = { title: string; labels: string[]; at: number; times: number[] };

/** The steps for StepList: finished ones ticked with their time, the current one turning. */
export function workSteps(work: Work): Step[] {
  return work.labels.map((label, index) => ({ key: String(index), label, ms: work.times[index],
    state: index < work.at ? 'done' : index === work.at ? 'running' : 'waiting' }));
}

/** Tracks one task at a time: begin with its steps, next() as each step finishes, end() at the end. */
export function useWork() {
  const [work, setWork] = useState<Work | null>(null);
  const stepStarted = useRef(0);
  const begin = useCallback((title: string, labels: string[]) => {
    stepStarted.current = performance.now();
    setWork({ title, labels, at: 0, times: [] });
  }, []);
  const next = useCallback(() => {
    const now = performance.now();
    const took = Math.round(now - stepStarted.current);
    stepStarted.current = now;
    setWork(current => current && { ...current, at: current.at + 1, times: [...current.times, took] });
  }, []);
  const end = useCallback(() => setWork(null), []);
  return { work, begin, next, end };
}

/** Shown after this long, so quick tasks do not flash their steps. */
const SHOW_AFTER_MS = 400;

/** The task once it has run longer than a blink; null before that and when there is none. */
export function useShownWork(work: Work | null): Work | null {
  const [shown, setShown] = useState(false);
  const title = work?.title;
  useEffect(() => {
    setShown(false);
    if (!title) return undefined;
    const timer = window.setTimeout(() => setShown(true), SHOW_AFTER_MS);
    return () => window.clearTimeout(timer);
  }, [title]);
  return shown ? work : null;
}
