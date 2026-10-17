import { useEffect, useState } from 'react';
import { activityApi, type Activity } from '../api/activity';

const EVERY_MS = 2000;

/** Polls what runs in the tools so every place can show its own running dot. */
export function useActivity(enabled: boolean) {
  const [activity, setActivity] = useState<Activity | null>(null);
  useEffect(() => {
    if (!enabled) return;
    let alive = true;
    const load = () => activityApi.get().then(value => { if (alive) setActivity(value); }).catch(() => {});
    load();
    const timer = setInterval(load, EVERY_MS);
    return () => { alive = false; clearInterval(timer); };
  }, [enabled]);
  return activity;
}
