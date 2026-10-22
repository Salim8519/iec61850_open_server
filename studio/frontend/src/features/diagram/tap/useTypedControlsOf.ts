import { useEffect, useState } from 'react';
import { api } from '../../../api/client';
import type { ExtendedControl, Ied, TypedControlList } from '../../../api/types';

const NONE: readonly ExtendedControl[] = [];

/** The typed controls (BSC, ISC, ENC …) of every IED from GET /controls/typed (ADR-0104), for the tap
 *  changer's selects and their "N fitting" counts (change 0176). An IED whose answer has not arrived, or
 *  failed, has none; the list is asked again when the IEDs or their size change. */
export function useTypedControlsOf(ieds: readonly Pick<Ied, 'id' | 'signal_count'>[]) {
  const key = ieds.map(ied => `${ied.id}:${ied.signal_count}`).join('|');
  const [lists, setLists] = useState<{ byIed: Record<string, readonly ExtendedControl[]> }>({ byIed: {} });
  useEffect(() => {
    if (!key) return undefined;
    let cancelled = false;
    const ids = key.split('|').map(item => item.slice(0, item.lastIndexOf(':')));
    Promise.all(ids.map(id => api<TypedControlList>(`/controls/typed?ied_id=${encodeURIComponent(id)}`)
      .then(data => [id, data.extended_controls ?? NONE] as const).catch(() => [id, NONE] as const)))
      .then(entries => { if (!cancelled) setLists({ byIed: Object.fromEntries(entries) }); });
    return () => { cancelled = true; };
  }, [key]);
  const byIed = lists.byIed; // kept while a newer answer is on its way, so the selects do not blink
  return (ied: Pick<Ied, 'id'> | undefined): readonly ExtendedControl[] => (ied && byIed[ied.id]) || NONE;
}
