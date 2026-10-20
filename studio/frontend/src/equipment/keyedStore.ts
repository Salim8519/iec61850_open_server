type Listener = () => void;

/** A record shown key by key (speed lane F): `replace` with a whole new record tells only the
 * listeners of keys whose value changed. Used for the SLD mode badges, which arrive as a full
 * equipment -> mode map every 3 s; before, every equipment node re-rendered on each answer. */
export function createKeyedStore<V>() {
  let values: Record<string, V> = {};
  const listeners = new Map<string, Set<Listener>>();
  return {
    get: (key: string): V | undefined => values[key],
    replace(next: Record<string, V>) {
      const changed = new Set<string>();
      for (const key of Object.keys(values)) if (!(key in next)) changed.add(key);
      for (const [key, value] of Object.entries(next)) if (!Object.is(values[key], value)) changed.add(key);
      values = { ...next };
      changed.forEach(key => listeners.get(key)?.forEach(listener => listener()));
      return changed.size;
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
  };
}

export type KeyedStore<V> = ReturnType<typeof createKeyedStore<V>>;
