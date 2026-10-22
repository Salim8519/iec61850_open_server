/** "Save always": a value is sent the moment it is valid, with no timer.
 *
 *  At most one request is in flight. A value pushed meanwhile waits as the one trailing save, and a newer
 *  push replaces it, so the latest value always wins and two requests never run side by side. A failed
 *  request keeps the typed value on the page (the caller owns it) and reports the reason; the next push
 *  tries again. This file has no React in it so it can be tested on its own (see state/useAutoSave.ts). */
export type SavePhase = 'idle' | 'saving' | 'saved' | 'failed' | 'waiting';
export type SaveState = { phase: SavePhase; message: string };

export const IDLE: SaveState = { phase: 'idle', message: '' };
/** What a page shows while its value is wrong and therefore not sent (the reason is the red line). */
export const NOT_SAVED: SaveState = { phase: 'failed', message: '' };

/** Thrown by a save that cannot be done right now but is fine later (for example the gateway runs):
 *  the value is kept and sent when `hold(false)` or `retry()` is called, or when a newer value is pushed. */
export class SaveLater extends Error {}

export type AutoSaver<T> = {
  /** Save this value now (or right after the request in flight). Replaces a value still waiting. */
  push: (value: T) => void;
  /** While held, nothing is sent; the latest pushed value waits. Releasing sends it. */
  hold: (on: boolean) => void;
  /** Try again a value that was refused for now (SaveLater). */
  retry: () => void;
  /** Forget a value that is waiting and clear the status (the request in flight, if any, still finishes). */
  drop: () => void;
  /** Whether a request is in flight or a value waits. */
  pending: () => boolean;
  /** Resolves when nothing is in flight and nothing waits (a held value counts as waiting for later). */
  settled: () => Promise<void>;
};

export function createAutoSaver<T>(run: (value: T) => Promise<unknown>, onState: (state: SaveState) => void): AutoSaver<T> {
  let busy = false;
  let waiting: { value: T } | null = null;
  let held = false;
  let later = false; // the last request was refused for now
  let done: Array<() => void> = [];

  const finish = () => { const callbacks = done; done = []; callbacks.forEach(call => call()); };
  const pump = (): void => {
    if (busy) return;
    if (!waiting) { finish(); return; }
    if (held || later) { onState({ phase: 'waiting', message: '' }); finish(); return; }
    const { value } = waiting;
    waiting = null;
    busy = true;
    onState({ phase: 'saving', message: '' });
    run(value).then((): SaveState => ({ phase: 'saved', message: '' }), (error): SaveState => {
      if (error instanceof SaveLater) {
        later = true;
        waiting ??= { value }; // a newer value, if any, already replaces this one
        return { phase: 'waiting', message: '' };
      }
      return { phase: 'failed', message: error instanceof Error ? error.message : String(error) };
    }).then(state => {
      busy = false;
      if (waiting && !held && !later) pump(); // the trailing save: the failure or success above is already out of date
      else { onState(state); finish(); }
    });
  };

  return {
    push(value) { waiting = { value }; later = false; pump(); }, // a newer value is a new attempt
    hold(on) { held = on; if (!on) { later = false; pump(); } else if (waiting && !busy) onState({ phase: 'waiting', message: '' }); },
    retry() { later = false; pump(); },
    drop() { waiting = null; if (!busy) onState(IDLE); },
    pending: () => busy || waiting !== null,
    settled: () => new Promise(resolve => { if (!busy && (!waiting || held || later)) resolve(); else done.push(resolve); }),
  };
}
