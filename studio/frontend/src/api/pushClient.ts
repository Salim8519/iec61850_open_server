import { setStreamFactory, sseStream, type StreamFactory, type StreamHandlers, type StudioStream } from '../state/eventStream.ts';
import type { StudioEvent } from './types.ts';
import { fetchCapabilities, pathTopic, type Capabilities, type PathTopic } from './capabilities.ts';
import type { PushSource, PushWatch } from './liveFeed.ts';
import { createTopicStates } from './pushStream.ts';
import type { Visibility } from './sharedPoll.ts';

/** Speed lane F4: the Go engine's push stream in the browser (ADR-0087).
 *
 * - One EventSource per tab, `/api/v1/push?client=<tab id>&after=<last bus id>`. Bus events (no
 *   `event:` line) go to the event stream handlers exactly as `/api/v1/events` did; `event: push`
 *   frames rebuild topic texts (pushStream) that the live feed shows instead of polling.
 * - One subscription per topic per tab; the last path (parameters) wins and an earlier path of
 *   that topic goes back to polling. `PUT /api/v1/push/topics` carries the watched list (`[]`
 *   while the page is hidden), at most one per frame, never two at once.
 * - Without push (Python, or capabilities failed) the plain stream and polling stay as they are.
 *   When the stream fails, the live feed polls until the stream says hello again. */
export type EventSourceLike = {
  onopen: (() => void) | null;
  onerror: (() => void) | null;
  onmessage: ((message: { data: string; lastEventId: string }) => void) | null;
  addEventListener: (type: 'push', listener: (message: { data: string }) => void) => void;
  close: () => void;
  readyState: number;
};
export type TopicsAnswer = { unknown?: string[] };
export type PushDeps = {
  capabilities: () => Promise<Capabilities | null>;
  open: (url: string) => EventSourceLike;
  /** PUT /api/v1/push/topics; rejects when it failed. */
  put: (body: { client: string; topics: { t: string; params: Record<string, string> }[]; resync: string[] }) =>
    Promise<TopicsAnswer | unknown>;
  tab: () => string;
  visibility: Visibility;
  /** Run once soon (one PUT per frame). */
  soon: (run: () => void) => void;
  /** Run after `ms` (stream retry). */
  later: (run: () => void, ms: number) => void;
  feed: { setPush: (source: PushSource | null) => void };
  /** Today's `/api/v1/events` stream, used when the backend has no push stream. */
  plain: StreamFactory;
};

const CLOSED = 2;
const RETRY_FIRST_MS = 1000;
const RETRY_MAX_MS = 10000;

type Watcher = PathTopic & { path: string; watch: PushWatch };

export function createPushClient(deps: PushDeps) {
  const states = createTopicStates();
  const watchers = new Map<string, Watcher>();
  const resync = new Set<string>();
  let sent = new Map<string, string>(); // topic -> parameter key, as the last PUT sent it
  let sentCount = 0;
  let want = false;
  let scheduled = false;
  let putting = false;
  let online = false;
  let paths: Capabilities['paths'] = {};
  let onFailure = () => {};

  const hidden = () => deps.visibility.hidden();

  /** The watched list changed (or must be sent again): one PUT soon. */
  function dirty() {
    want = true;
    if (!online || scheduled || putting) return;
    scheduled = true;
    deps.soon(flush);
  }

  function flush() {
    scheduled = false;
    if (!want || putting || !online) return;
    want = false;
    putting = true;
    const list = hidden() ? [] : [...watchers];
    const number = ++sentCount;
    sent = new Map(list.map(([t, watcher]) => [t, watcher.key]));
    const body = { client: deps.tab(), topics: list.map(([t, watcher]) => ({ t, params: watcher.params })),
      resync: [...resync] };
    resync.clear();
    deps.put(body).then(answer => {
      putting = false;
      states.acked(number).forEach(t => resync.add(t));
      const unknown = (answer as TopicsAnswer | null)?.unknown;
      if (Array.isArray(unknown)) unknown.forEach(t => watchers.get(t)?.watch.release?.());
      if (resync.size) want = true;
      if (want) dirty();
    }, () => {
      putting = false;
      want = true;
      onFailure();
    });
  }

  /** The live feed's push source: covers the capabilities' paths while the stream is up. */
  const source: PushSource = {
    covers: path => pathTopic(paths, path) !== null,
    watch(path, onText, more) {
      const topic = pathTopic(paths, path);
      if (!topic) return () => {};
      const watcher: Watcher = { ...topic, path, watch: { text: onText, ask: more?.ask, release: more?.release } };
      const previous = watchers.get(topic.t);
      watchers.set(topic.t, watcher);
      previous?.watch.release?.();
      if (!previous || previous.key !== topic.key || !states.tracked(topic.t)) {
        const changed = sent.has(topic.t) && sent.get(topic.t) !== topic.key;
        states.track(topic.t, changed ? sentCount + 1 : 0);
      } else {
        const held = states.text(topic.t); // same parameters, other spelling: the held text is its answer
        if (held !== null) onText(held);
      }
      dirty();
      return () => {
        if (watchers.get(topic.t) !== watcher) return;
        watchers.delete(topic.t);
        states.untrack(topic.t);
        dirty();
      };
    },
  };

  function frame(data: string, epoch: { value: string }) {
    const outcome = states.apply(data);
    if (outcome.kind === 'hello') {
      if (outcome.epoch !== epoch.value) {
        epoch.value = outcome.epoch;
        states.resetAll();
        sent = new Map(); // the restarted server holds no subscription of this tab
      }
      if (!online) { online = true; deps.feed.setPush(source); }
      dirty(); // always: after a long outage the server may have forgotten this tab
    } else if (outcome.kind === 'text') {
      watchers.get(outcome.t)?.watch.text(outcome.text);
    } else if (outcome.kind === 'status') {
      watchers.get(outcome.t)?.watch.ask?.(); // the route answered an error: show it as polling would
    } else if (outcome.kind === 'resync') {
      resync.add(outcome.t);
      dirty();
    }
  }

  function offline() {
    if (!online) return;
    online = false;
    deps.feed.setPush(null); // the views poll until the stream is back
  }

  /** Connect with push; the returned stream closes everything. */
  function connect(caps: Capabilities, handlers: StreamHandlers): StudioStream {
    paths = caps.paths;
    const epoch = { value: caps.epoch };
    let current: EventSourceLike | null = null;
    let fallback: StudioStream | null = null;
    let lastId = '0';
    let delay = RETRY_FIRST_MS;
    let stopped = false;

    const start = () => {
      const tab = deps.tab();
      const stream = deps.open(`/api/v1/push?client=${encodeURIComponent(tab)}&after=${lastId}`);
      current = stream;
      stream.onopen = () => handlers.open();
      stream.onmessage = message => {
        if (/^\d+$/.test(message.lastEventId)) lastId = message.lastEventId;
        handlers.event(JSON.parse(message.data) as StudioEvent);
      };
      stream.addEventListener('push', message => {
        frame(message.data, epoch);
        if (online) delay = RETRY_FIRST_MS;
      });
      stream.onerror = () => {
        offline(); // EventSource reconnects by itself unless it closed or the tab id changed
        if (stream.readyState === CLOSED || deps.tab() !== tab) retry();
      };
    };

    const retry = () => {
      offline();
      current?.close();
      current = null;
      const wait = delay;
      delay = Math.min(delay * 2, RETRY_MAX_MS);
      deps.later(() => {
        if (stopped || current) return;
        deps.capabilities().then(again => {
          if (stopped || current) return;
          if (again) { paths = again.paths; start(); } else fallback = deps.plain(handlers);
        }, () => { if (!stopped) retry(); });
      }, wait);
    };

    onFailure = retry;
    start();
    const unsubscribe = deps.visibility.subscribe(() => dirty());
    return { close: () => {
      stopped = true;
      unsubscribe();
      current?.close();
      current = null;
      fallback?.close();
      offline();
    } };
  }

  /** The event stream factory: asks the capabilities once the session is set up (the factory runs
   * after the session step), then opens the push stream or, without push, today's plain stream. */
  const factory: StreamFactory = handlers => {
    let closed = false;
    let inner: StudioStream | null = null;
    deps.capabilities().then(caps => caps, () => null).then(caps => {
      if (closed) return;
      inner = caps ? connect(caps, handlers) : deps.plain(handlers);
    });
    return { close: () => { closed = true; inner?.close(); } };
  };

  return { factory, source, online: () => online, watched: () => [...watchers.keys()],
    text: states.text, rev: states.rev };
}

export type PushClient = ReturnType<typeof createPushClient>;

const TAB_KEY = 'studio-push-tab';
const TAB_ID = /^[A-Za-z0-9._-]{1,64}$/;

/** This tab's push id, kept in sessionStorage so a reload finds its subscriptions again. A
 * duplicated tab copies sessionStorage: the tabs ask each other once over a BroadcastChannel and
 * the newcomer takes a new id (else both streams would keep replacing each other). */
export function browserTabId(): () => string {
  const fresh = () => {
    const id = `tab-${crypto.randomUUID()}`;
    try { sessionStorage.setItem(TAB_KEY, id); } catch { /* storage blocked: one id per page load */ }
    return id;
  };
  let id = '';
  try { id = sessionStorage.getItem(TAB_KEY) ?? ''; } catch { /* storage blocked */ }
  if (!TAB_ID.test(id)) id = fresh();
  if (typeof BroadcastChannel === 'function') {
    const channel = new BroadcastChannel(TAB_KEY);
    channel.onmessage = ({ data }: MessageEvent<{ ask?: string; taken?: string }>) => {
      if (data?.ask === id) channel.postMessage({ taken: id });
      else if (data?.taken === id) id = fresh();
    };
    channel.postMessage({ ask: id });
  }
  return () => id;
}

/** Install the push client in the page (main.tsx): the event stream factory asks the capabilities
 * after the session step; with push the live feed's covered paths stop polling. */
export function installPushClient(env: { feed: PushDeps['feed']; put: PushDeps['put'] }) {
  const client = createPushClient({
    capabilities: () => fetchCapabilities((input, init) => fetch(input, init), window.location.origin),
    open: url => new EventSource(url) as unknown as EventSourceLike,
    put: env.put,
    tab: browserTabId(),
    visibility: {
      hidden: () => document.visibilityState === 'hidden',
      subscribe: changed => {
        document.addEventListener('visibilitychange', changed);
        return () => document.removeEventListener('visibilitychange', changed);
      },
    },
    soon: run => { setTimeout(run, 16); },
    later: (run, ms) => { setTimeout(run, ms); },
    feed: env.feed,
    plain: sseStream,
  });
  setStreamFactory(client.factory);
  return client;
}
