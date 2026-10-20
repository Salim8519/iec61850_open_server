/** One request at a time per path, shared by every view that shows it (change 0110).
 *
 * Several views polled /health and /goose/links on their own timers and kept sending while the
 * backend was still answering, so a slow answer piled up more requests. Here each path has one
 * timer at the fastest interval asked for, never overlaps requests, and gives the last answer
 * to a view that opens later. Display only: the backend stays the authority.
 *
 * Speed lane F: an answer that is the very same object as the last one is not delivered again
 * (the live feed hands back its cached object when the text did not change), timers pause while
 * the page is hidden, and a path can be held (no timer) while a push source feeds it. */
export type Fetcher = (path: string) => Promise<unknown>;
export type Timers = {
  setInterval: (run: () => void, every: number) => unknown;
  clearInterval: (handle: unknown) => void;
};
/** Whether the page is hidden, and a way to hear when that changes. */
export type Visibility = { hidden: () => boolean; subscribe: (changed: () => void) => () => void };
export type PollOptions = { timers?: Timers; visibility?: Visibility };
type Listener = { every: number | null; onValue: (value: unknown) => void; onError?: (error: Error) => void };
type Entry = { listeners: Set<Listener>; timer?: unknown; every: number | null; busy: boolean; again: boolean;
  last?: unknown; fresh: boolean; held: boolean };

const realTimers: Timers = {
  setInterval: (run, every) => setInterval(run, every),
  clearInterval: handle => clearInterval(handle as ReturnType<typeof setInterval>),
};

export function createPoller(fetcher: Fetcher, options: PollOptions = {}) {
  const entries = new Map<string, Entry>();
  const timers = options.timers ?? realTimers;
  const visibility = options.visibility;
  const hidden = () => !!visibility?.hidden();

  /** Give `value` to every view, unless it is the same object they already have. */
  const deliver = (entry: Entry, value: unknown) => {
    if (!entry.fresh && entry.last !== undefined && Object.is(value, entry.last)) return;
    entry.fresh = false;
    entry.last = value;
    entry.listeners.forEach(listener => listener.onValue(value));
  };

  const load = (path: string, entry: Entry) => {
    if (entry.busy) { entry.again = true; return; }
    entry.busy = true;
    fetcher(path).then(value => deliver(entry, value), (reason: unknown) => {
      const error = reason instanceof Error ? reason : new Error(String(reason));
      entry.fresh = true;  // the next answer is shown even when it equals the one before the error
      entry.listeners.forEach(listener => listener.onError?.(error));
    }).finally(() => {
      entry.busy = false;
      if (entry.again && entry.listeners.size) { entry.again = false; load(path, entry); }
    });
  };

  const fastest = (entry: Entry) => {
    const intervals = [...entry.listeners].map(listener => listener.every).filter((every): every is number => !!every);
    return intervals.length ? Math.min(...intervals) : null;
  };

  const stopTimer = (entry: Entry) => {
    if (entry.timer !== undefined) timers.clearInterval(entry.timer);
    entry.timer = undefined;
  };

  const retime = (path: string, entry: Entry) => {
    const every = entry.held || hidden() ? null : fastest(entry);
    if (every === entry.every) return;
    stopTimer(entry);
    entry.every = every;
    if (every) entry.timer = timers.setInterval(() => load(path, entry), every);
  };

  // Hidden page: no timers. Visible again: ask once now, then on the usual timers.
  // Listened to from the first subscription, so creating a poller has no side effect.
  let watching = false;
  const watch = () => {
    if (watching || !visibility) return;
    watching = true;
    visibility.subscribe(() => entries.forEach((entry, path) => {
      retime(path, entry);
      if (!hidden() && entry.every) load(path, entry);
    }));
  };

  /** Show `path` while subscribed: now, then every `every` ms (null: once). Returns unsubscribe. */
  function subscribe<T>(path: string, every: number | null, onValue: (value: T) => void,
    onError?: (error: Error) => void): () => void {
    watch();
    let entry = entries.get(path);
    const created = !entry;
    if (!entry) {
      entry = { listeners: new Set(), every: null, busy: false, again: false, fresh: true, held: false };
      entries.set(path, entry);
    }
    const listener: Listener = { every, onValue: onValue as (value: unknown) => void, onError };
    entry.listeners.add(listener);
    if (entry.last !== undefined) listener.onValue(entry.last);
    retime(path, entry);
    // A request already in flight answers this view too; asking again would only add a request.
    if ((!entry.held || created) && !entry.busy) load(path, entry);
    const current = entry;
    return () => {
      current.listeners.delete(listener);
      retime(path, current);
      if (!current.listeners.size) { stopTimer(current); entries.delete(path); }
    };
  }

  /** Ask again now (for example after a change); the answer is shown even when unchanged. */
  function refresh(path: string) {
    const entry = entries.get(path);
    if (entry) { entry.fresh = true; load(path, entry); }
  }

  /** An answer that arrived another way (a push frame); same rules as a fetched answer. */
  function offer(path: string, value: unknown) {
    const entry = entries.get(path);
    if (entry) deliver(entry, value);
  }

  /** Stop (true) or restart (false) the timer of `path`, for example while a push source feeds it. */
  function hold(path: string, held: boolean) {
    const entry = entries.get(path);
    if (!entry || entry.held === held) return;
    entry.held = held;
    retime(path, entry);
  }

  return { subscribe, refresh, offer, hold, active: () => [...entries.keys()],
    interval: (path: string) => entries.get(path)?.every ?? null };
}
