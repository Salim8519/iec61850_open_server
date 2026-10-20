/** One flush per animation frame for live updates (speed lane F).
 *
 * Stores outside React collect changes and ask for a flush; whatever arrives before the next frame
 * is shown once. A hidden page gets no animation frames, so a visibility change flushes at once
 * (nothing waits for the page to show again). The clock is injectable so node tests can drive it. */
export type FrameClock = {
  request: (run: () => void) => unknown;
  cancel: (handle: unknown) => void;
};
export type FrameBatch = {
  /** Run `task` once before the next paint; the same task asked twice in a frame runs once. */
  schedule: (task: () => void) => void;
  /** Run everything pending now. */
  flush: () => void;
  pending: () => number;
};

export function createFrameBatch(clock: FrameClock, onVisibility?: (changed: () => void) => void): FrameBatch {
  const tasks = new Set<() => void>();
  let handle: unknown = null;

  const flush = () => {
    if (handle !== null) { clock.cancel(handle); handle = null; }
    // Tasks scheduled while flushing wait for the next frame, so a busy stream cannot starve paint.
    const now = [...tasks];
    tasks.clear();
    for (const task of now) task();
  };

  const schedule = (task: () => void) => {
    tasks.add(task);
    if (handle === null) handle = clock.request(() => { handle = null; flush(); });
  };

  onVisibility?.(flush);
  return { schedule, flush, pending: () => tasks.size };
}

/** The browser's clock: requestAnimationFrame, and a flush on every visibility change. */
export function browserFrameBatch(): FrameBatch {
  return createFrameBatch({
    request: run => window.requestAnimationFrame(run),
    cancel: handle => window.cancelAnimationFrame(handle as number),
  }, changed => document.addEventListener('visibilitychange', changed));
}
