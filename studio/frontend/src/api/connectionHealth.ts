/** Whether the page still reaches the engine (change 0177). Every API answer counts: an answer means the
 *  engine is there, while a failed request (engine stopped or crashed, network gone) or a lost session
 *  (401) means the values on screen may be old. The sidebar dot and words follow this, not "a project was
 *  loaded once". No extra requests: the page's own polls are the evidence. */
export type ConnectionState = 'connecting' | 'connected' | 'lost';

export const CONNECTION_COPY = {
  connecting: 'Connecting…',
  connected: 'Engine connected',
  lost: 'Connection lost, values may be old',
  lostHint: 'ScadaBench cannot reach its engine, so nothing on screen updates. It keeps trying; if this stays, restart ScadaBench.',
};

export function createConnectionHealth() {
  let state: ConnectionState = 'connecting';
  const listeners = new Set<() => void>();
  const set = (next: ConnectionState) => {
    if (next === state) return;
    state = next;
    listeners.forEach(listener => listener());
  };
  return {
    /** An HTTP answer arrived; `status` 401 means the session is gone, so nothing new will come. */
    answered: (status: number) => set(status === 401 ? 'lost' : 'connected'),
    /** The request failed before any answer (engine gone or unreachable). */
    failed: () => set(state === 'connecting' ? 'connecting' : 'lost'),
    state: () => state,
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => { listeners.delete(listener); };
    },
  };
}

/** The page's one connection health, fed by `api/client.ts`. */
export const connectionHealth = createConnectionHealth();
