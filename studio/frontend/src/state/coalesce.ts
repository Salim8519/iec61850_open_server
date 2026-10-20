/** At most one run in flight and one waiting (speed lane F, GET /project).
 *
 * Replayed and live events (runtime, resync, live_status, project_changed) each asked for the
 * project, so the first second sent GET /project up to 7 times. A call while a run is in flight
 * joins the one trailing run, which starts after it; every caller's promise settles with a run
 * that started after its call, so `await refresh()` still sees the state after the caller's
 * action. */
export function coalesce<T>(task: () => Promise<T>) {
  let running: Promise<T> | null = null;
  let trailing: Promise<T> | null = null;

  const start = (): Promise<T> => {
    const run = task().finally(() => { if (running === run) running = null; });
    running = run;
    return run;
  };

  return function call(): Promise<T> {
    if (!running) return start();
    if (!trailing) {
      const ahead = running;
      trailing = ahead.catch(() => undefined).then(() => { trailing = null; return start(); });
    }
    return trailing;
  };
}
