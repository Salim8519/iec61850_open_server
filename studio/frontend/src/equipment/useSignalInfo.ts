import { useEffect, useState } from 'react';
import type { Signal, SignalBinding } from '../api/types';
import { api } from '../api/client';

/** Signal descriptions asked once per page ("iedId:ref"); a failed lookup is asked again later. */
const cache = new Map<string, Promise<Signal | null>>();

function lookup(iedId: string, ref: string): Promise<Signal | null> {
  const key = `${iedId}:${ref}`;
  let found = cache.get(key);
  if (!found) {
    found = api<{ items: Signal[] }>(`/ieds/${encodeURIComponent(iedId)}/signals?ref=${encodeURIComponent(ref)}`)
      .then(result => result.items[0] ?? null)
      .catch(() => { cache.delete(key); return null; });
    cache.set(key, found);
  }
  return found;
}

/** The bound signal's description (kind, FC, quality point); undefined while loading or unknown. */
export function useSignalInfo(binding: SignalBinding): Signal | undefined {
  const [signal, setSignal] = useState<Signal | undefined>();
  useEffect(() => {
    let active = true;
    setSignal(undefined);
    lookup(binding.ied_id, binding.ref).then(found => { if (active) setSignal(found ?? undefined); });
    return () => { active = false; };
  }, [binding.ied_id, binding.ref]);
  return signal;
}
