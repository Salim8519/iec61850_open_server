import type { FrameBatch } from './frameBatch.ts';

/** One Activity row, formatted once when it arrives (not on every render). */
/** `count` > 1 when repeated events were folded into this row (change 0177). */
export type LogEntry<E> = { id: number; kind: string; label: string; time: string; event: E; details?: string; count: number };
type Logged = { id: number; kind: string; timestamp: number };

/** The Activity log outside React (speed lane F): a ring of the newest `capacity` events, the time
 * text made once per event, a count per kind, and one notification per frame. Before, every event
 * copied a 500-item array into App state and re-rendered the whole app. */
export function createEventLogStore<E extends Logged>(batch: FrameBatch, options: {
  capacity?: number; formatTime?: (timestamp: number) => string; describe?: (event: E, count: number) => string;
  /** A key for events that repeat a lot (for example "report created"): those with the same key in the same
   *  minute fold into one row with a count, so they never push the rest of Activity out of the ring. */
  fold?: (event: E) => string | null;
} = {}) {
  const capacity = options.capacity ?? 500;
  const formatTime = options.formatTime ?? ((timestamp: number) => new Date(timestamp * 1000).toLocaleTimeString());
  const ring: (LogEntry<E> | undefined)[] = new Array(capacity);
  let next = 0;
  let size = 0;
  let version = 0;
  let lastSecond = NaN;
  let lastTime = '';
  const counts = new Map<string, number>();
  const folds = new Map<string, { entry: LogEntry<E>; slot: number; minute: number }>();
  const listeners = new Set<() => void>();
  const notify = () => { version += 1; listeners.forEach(listener => listener()); };

  const time = (timestamp: number) => {
    // Events of the same second share their text: a burst formats the clock once.
    const second = Math.floor(timestamp);
    if (second !== lastSecond) { lastSecond = second; lastTime = formatTime(second); }
    return lastTime;
  };

  return {
    add(event: E) {
      counts.set(event.kind, (counts.get(event.kind) ?? 0) + 1);
      const key = options.fold?.(event) ?? null;
      const minute = Math.floor(event.timestamp / 60);
      const folded = key === null ? undefined : folds.get(key);
      if (folded && folded.minute === minute && ring[folded.slot] === folded.entry) {
        folded.entry.count += 1; folded.entry.event = event; folded.entry.details = undefined;
        batch.schedule(notify);
        return;
      }
      const entry: LogEntry<E> = { id: event.id, kind: event.kind, label: event.kind.replaceAll('_', ' '), time: time(event.timestamp), event, count: 1 };
      ring[next] = entry;
      if (key !== null) folds.set(key, { entry, slot: next, minute });
      next = (next + 1) % capacity;
      size = Math.min(size + 1, capacity);
      batch.schedule(notify);
    },
    /** Row `index` counted from the newest (0) to the oldest (size - 1). */
    at(index: number): LogEntry<E> | undefined {
      if (index < 0 || index >= size) return undefined;
      return ring[(next - 1 - index + capacity * 2) % capacity];
    },
    /** The details column, worked out the first time the row is shown. */
    details(entry: LogEntry<E>) {
      if (entry.details === undefined) entry.details = options.describe?.(entry.event, entry.count) ?? '';
      return entry.details;
    },
    size: () => size,
    /** Events of `kind` received since the page opened (not limited to the ring). */
    count: (kind: string) => counts.get(kind) ?? 0,
    version: () => version,
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => { listeners.delete(listener); };
    },
  };
}

export type EventLogStore<E extends Logged> = ReturnType<typeof createEventLogStore<E>>;
