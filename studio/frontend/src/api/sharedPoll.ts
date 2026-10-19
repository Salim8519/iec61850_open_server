/** One request at a time per path, shared by every view that shows it (change 0110).
 *
 * Several views polled /health and /goose/links on their own timers and kept sending while the
 * backend was still answering, so a slow answer piled up more requests. Here each path has one
 * timer at the fastest interval asked for, never overlaps requests, and gives the last answer
 * to a view that opens later. Display only: the backend stays the authority. */
export type Fetcher = (path: string) => Promise<unknown>;
type Listener = { every: number | null; onValue: (value: unknown) => void; onError?: (error: Error) => void };
type Entry = { listeners: Set<Listener>; timer?: ReturnType<typeof setInterval>; every: number | null;
  busy: boolean; again: boolean; last?: unknown };

export function createPoller(fetcher: Fetcher) {
  const entries = new Map<string, Entry>();

  const load = (path: string, entry: Entry) => {
    if (entry.busy) { entry.again = true; return; }
    entry.busy = true;
    fetcher(path).then(value => {
      entry.last = value;
      entry.listeners.forEach(listener => listener.onValue(value));
    }, (reason: unknown) => {
      const error = reason instanceof Error ? reason : new Error(String(reason));
      entry.listeners.forEach(listener => listener.onError?.(error));
    }).finally(() => {
      entry.busy = false;
      if (entry.again && entry.listeners.size) { entry.again = false; load(path, entry); }
    });
  };

  const retime = (path: string, entry: Entry) => {
    const intervals = [...entry.listeners].map(listener => listener.every).filter((every): every is number => !!every);
    const every = intervals.length ? Math.min(...intervals) : null;
    if (every === entry.every) return;
    if (entry.timer) clearInterval(entry.timer);
    entry.every = every;
    entry.timer = every ? setInterval(() => load(path, entry), every) : undefined;
  };

  /** Show `path` while subscribed: now, then every `every` ms (null: once). Returns unsubscribe. */
  function subscribe<T>(path: string, every: number | null, onValue: (value: T) => void,
    onError?: (error: Error) => void): () => void {
    let entry = entries.get(path);
    if (!entry) { entry = { listeners: new Set(), every: null, busy: false, again: false }; entries.set(path, entry); }
    const listener: Listener = { every, onValue: onValue as (value: unknown) => void, onError };
    entry.listeners.add(listener);
    if (entry.last !== undefined) listener.onValue(entry.last);
    retime(path, entry);
    load(path, entry);
    const current = entry;
    return () => {
      current.listeners.delete(listener);
      retime(path, current);
      if (!current.listeners.size) entries.delete(path);
    };
  }

  /** Ask again now (for example after a change), shared with whatever is already in flight. */
  function refresh(path: string) {
    const entry = entries.get(path);
    if (entry) load(path, entry);
  }

  return { subscribe, refresh, active: () => [...entries.keys()] };
}
