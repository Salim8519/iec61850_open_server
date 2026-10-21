import { createPoller, type PollOptions } from './sharedPoll.ts';

/** Speed lane F: every poll of the app goes through one live feed.
 *
 * - One request per path at a time, shared by every view, at the fastest interval asked for
 *   (shared poller, change 0110); the intervals are the ones each view used before.
 * - The answer is compared as raw text: an unchanged answer is not parsed again and no view is
 *   called, so an idle page does no React work.
 * - Timers pause while the page is hidden and ask once when it shows again.
 * - A push source (the Go engine's push stream, a later lane) can feed a path instead of the
 *   timer: `setPush` with an object that says which paths it covers and sends their JSON text.
 *   Views keep the same `subscribe` call either way. */
export type TextFetcher = (path: string) => Promise<string>;
/** What a push source can tell the feed about one path (speed lane F4). */
export type PushWatch = {
  /** The full JSON text of GET /api/v1{path}. */
  text: (text: string) => void;
  /** The route answered an error: ask once over HTTP, so the view shows it as polling would. */
  ask?: () => void;
  /** The source no longer sends this path (another path of its topic won): poll it again. */
  release?: () => void;
};
export type PushSource = {
  /** True when this source sends the answer of `path` (same JSON as GET /api/v1{path}). */
  covers: (path: string) => boolean;
  /** Start sending `path`; every frame is the full JSON text. Returns stop. */
  watch: (path: string, onText: (text: string) => void, more?: Omit<PushWatch, 'text'>) => () => void;
};

/** Flask ends an answer with a newline that the push stream's text leaves out; compare without it. */
const bare = (text: string) => text.endsWith('\n') ? text.slice(0, -1) : text;

export function createLiveFeed(fetchText: TextFetcher, options: PollOptions = {}) {
  const cache = new Map<string, { text: string; value: unknown }>();
  const decode = (path: string, raw: string) => {
    const text = bare(raw);
    const known = cache.get(path);
    if (known && known.text === text) return known.value;
    const value: unknown = JSON.parse(text);
    cache.set(path, { text, value });
    return value;
  };
  // Pushed frames per path: a request that was answered while a frame arrived may be older than
  // that frame, so the frame's answer is kept (the stream sends every later change anyway).
  const frames = new Map<string, number>();
  // Incremental feeds (traffic logs ask "after N") register a task under a key instead of a path.
  const tasks = new Map<string, () => Promise<unknown>>();
  const nothing = new Map<string, unknown>();
  const fetch = (path: string) => {
    const task = tasks.get(path);
    if (!task) {
      const seen = frames.get(path);
      return fetchText(path).then(text => {
        const pushed = frames.get(path) !== seen ? cache.get(path) : undefined;
        return pushed ? pushed.value : decode(path, text);
      });
    }
    // A task answers null when nothing is new: hand back the last answer, so no view is called.
    return task().then(value => value === null ? nothing.get(path) ?? null : (nothing.set(path, value), value));
  };
  const poller = createPoller(fetch, options);
  let push: PushSource | null = null;
  const pushed = new Map<string, () => void>();

  const attach = (path: string) => {
    if (!push || pushed.has(path) || !push.covers(path)) return;
    poller.hold(path, true);
    const onText = (text: string) => {
      frames.set(path, (frames.get(path) ?? 0) + 1);
      poller.offer(path, decode(path, text));
    };
    pushed.set(path, push.watch(path, onText, { ask: () => poller.refresh(path),
      release: () => { detach(path); poller.hold(path, false); } }));
  };
  const detach = (path: string) => {
    pushed.get(path)?.();
    pushed.delete(path);
    frames.delete(path);
  };

  /** Show `path` now and then every `every` ms (null: once). Unchanged answers are not delivered. */
  function subscribe<T>(path: string, every: number | null, onValue: (value: T) => void,
    onError?: (error: Error) => void): () => void {
    const stop = poller.subscribe<T>(path, every, onValue, onError);
    attach(path);
    return () => {
      stop();
      if (!poller.active().includes(path)) { detach(path); cache.delete(path); }
    };
  }

  /** Like `subscribe`, for a feed that is not one fixed GET (for example "traffic after N"): `task`
   * runs on the same shared, never-overlapping, hidden-paused timer; it answers null when nothing
   * is new. One task per key: the latest subscriber's task is used. */
  function subscribeTask<T>(key: string, every: number | null, task: () => Promise<T | null>,
    onValue: (value: T) => void, onError?: (error: Error) => void): () => void {
    tasks.set(key, task);
    const stop = poller.subscribe<T | null>(key, every, value => { if (value !== null) onValue(value); }, onError);
    return () => {
      stop();
      if (!poller.active().includes(key)) { tasks.delete(key); nothing.delete(key); }
    };
  }

  /** Use `source` for the paths it covers (null: back to polling everything). */
  function setPush(source: PushSource | null) {
    [...pushed.keys()].forEach(path => { detach(path); poller.hold(path, false); });
    push = source;
    poller.active().forEach(attach);
  }

  return { subscribe, subscribeTask, setPush, refresh: poller.refresh, active: poller.active, interval: poller.interval,
    pushed: () => [...pushed.keys()] };
}

export type LiveFeed = ReturnType<typeof createLiveFeed>;
