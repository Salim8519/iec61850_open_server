/** Runs the SCADA tools' requests one after another, in click order.
 *
 *  A setting is saved when its field loses focus. Typing a port and pressing Start or Connect at once
 *  therefore produces "save" then "start" in one gesture. If the save disabled the button, the press
 *  was lost and nothing started. Saves are queued quietly instead (no busy state), and the start waits
 *  for them, so it always uses the settings the person just typed. A failed task never blocks the next. */
export function createActionQueue() {
  let tail: Promise<unknown> = Promise.resolve();
  return function enqueue<T>(task: () => Promise<T>): Promise<T> {
    const run = tail.then(task);
    tail = run.catch(() => undefined);
    return run;
  };
}

export type ActionQueue = ReturnType<typeof createActionQueue>;
