import type { FrameBatch } from './frameBatch.ts';

type Listener = () => void;

/** Live data point values outside React (speed lane F).
 *
 * Keys are "iedId:reference". A `signal` event used to copy the whole value map (up to 90k keys)
 * and re-render every page; here a write only records the newest value of its key (last wins),
 * and once per frame the listeners of the keys that really changed are told. A table row reads
 * its own key, so a burst repaints only the rows whose value changed. */
export function createValueStore(batch: FrameBatch) {
  const values = new Map<string, unknown>();
  const pending = new Map<string, unknown>();
  const listeners = new Map<string, Set<Listener>>();
  const any = new Set<Listener>();
  let cleared = false;
  let version = 0;

  const flush = () => {
    const changed: string[] = [];
    if (cleared) {
      changed.push(...values.keys());
      values.clear();
      cleared = false;
    }
    pending.forEach((value, key) => {
      if (values.has(key) && Object.is(values.get(key), value)) return;
      values.set(key, value);
      changed.push(key);
    });
    pending.clear();
    if (!changed.length) return;
    version += 1;
    new Set(changed).forEach(key => listeners.get(key)?.forEach(listener => listener()));
    any.forEach(listener => listener());
  };

  return {
    get: (key: string) => values.get(key),
    has: (key: string) => values.has(key),
    /** Record a value; listeners hear about it once before the next paint. */
    set(key: string, value: unknown) {
      pending.set(key, value);
      batch.schedule(flush);
    },
    /** Forget every value (Stop, a new project). */
    clear() {
      pending.clear();
      cleared = true;
      batch.schedule(flush);
    },
    subscribe(key: string, listener: Listener) {
      const set = listeners.get(key) || new Set<Listener>();
      set.add(listener);
      listeners.set(key, set);
      return () => {
        set.delete(listener);
        if (!set.size) listeners.delete(key);
      };
    },
    /** Hear about any change (once per frame). */
    subscribeAll(listener: Listener) {
      any.add(listener);
      return () => { any.delete(listener); };
    },
    version: () => version,
    size: () => values.size,
  };
}

export type ValueStore = ReturnType<typeof createValueStore>;

/** The value keys a `signal` or `control_applied` event writes: its reference and its `updates`. */
export function eventValues(event: { kind: string; ied_id?: string; ref?: string; status_ref?: string; value?: unknown;
  updates?: unknown }): [string, unknown][] {
  if (event.kind !== 'signal' && event.kind !== 'control_applied') return [];
  const ref = event.status_ref || event.ref;
  const result: [string, unknown][] = [[`${event.ied_id}:${ref}`, event.value]];
  for (const [reference, value] of Object.entries((event.updates || {}) as Record<string, unknown>)) {
    result.push([`${event.ied_id}:${reference}`, value]);
  }
  return result;
}
