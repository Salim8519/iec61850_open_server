type Listener = () => void;

/** A per-equipment external store keeps unrelated live events out of React nodes. */
export class EquipmentValueStore {
  private values = new Map<string, unknown>();
  private listeners = new Map<string, Set<Listener>>();

  get(key: string) {
    return this.values.get(key);
  }

  set(key: string, value: unknown) {
    if (this.values.has(key) && Object.is(this.values.get(key), value)) return;
    this.values.set(key, value);
    this.listeners.get(key)?.forEach(listener => listener());
  }

  replace(entries: { node_id: string; value: unknown }[]) {
    const incoming = new Map(entries.map(entry => [entry.node_id, entry.value]));
    [...this.values.keys()].forEach(key => {
      if (!incoming.has(key)) {
        this.values.delete(key);
        this.listeners.get(key)?.forEach(listener => listener());
      }
    });
    incoming.forEach((value, key) => this.set(key, value));
  }

  clear() {
    const changed = [...this.values.keys()];
    this.values.clear();
    changed.forEach(key => this.listeners.get(key)?.forEach(listener => listener()));
  }

  subscribe(key: string, listener: Listener) {
    const listeners = this.listeners.get(key) || new Set<Listener>();
    listeners.add(listener);
    this.listeners.set(key, listeners);
    return () => {
      listeners.delete(listener);
      if (!listeners.size) this.listeners.delete(key);
    };
  }
}
