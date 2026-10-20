/** Hand-driven clocks for the speed lane F store tests (no DOM, no real timers). */
export function fakeFrames() {
  let next = 1;
  const pending = new Map<number, () => void>();
  return {
    clock: {
      request: (run: () => void) => { const id = next++; pending.set(id, run); return id; },
      cancel: (handle: unknown) => { pending.delete(handle as number); },
    },
    /** Paint one frame: run what was asked before it. */
    frame() {
      const now = [...pending.values()];
      pending.clear();
      now.forEach(run => run());
    },
    waiting: () => pending.size,
  };
}

export function fakeTimers() {
  let next = 1;
  const timers = new Map<number, { run: () => void; every: number }>();
  return {
    timers: {
      setInterval: (run: () => void, every: number) => { const id = next++; timers.set(id, { run, every }); return id; },
      clearInterval: (handle: unknown) => { timers.delete(handle as number); },
    },
    /** Fire every running interval once. */
    tick() { [...timers.values()].forEach(timer => timer.run()); },
    intervals: () => [...timers.values()].map(timer => timer.every),
  };
}

export function fakeVisibility() {
  let hidden = false;
  const listeners = new Set<() => void>();
  return {
    visibility: {
      hidden: () => hidden,
      subscribe: (changed: () => void) => { listeners.add(changed); return () => listeners.delete(changed); },
    },
    set(value: boolean) { hidden = value; listeners.forEach(listener => listener()); },
  };
}

export const settle = () => new Promise(resolve => setTimeout(resolve, 0));
