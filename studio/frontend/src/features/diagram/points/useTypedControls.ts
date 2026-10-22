import { useEffect, useState } from 'react';
import { api } from '../../../api/client';
import type { ExtendedControl, Ied, TypedControlList } from '../../../api/types';

const NONE: readonly ExtendedControl[] = [];

/** The typed controls (INC, ENC, APC, BAC, BSC, ISC) of the panel's IED from GET /controls/typed (ADR-0104).
 *  Empty until the answer arrives, and when the engine has no such list: the SPC/DPC rows still work. */
export function useTypedControls(ied: Pick<Ied, 'id' | 'signal_count'> | undefined): readonly ExtendedControl[] {
  const [list, setList] = useState<{ id: string; controls: readonly ExtendedControl[] }>({ id: '', controls: NONE });
  const id = ied?.id ?? '';
  const size = ied?.signal_count;
  useEffect(() => {
    if (!id) return undefined;
    let cancelled = false;
    api<TypedControlList>(`/controls/typed?ied_id=${encodeURIComponent(id)}`)
      .then(data => { if (!cancelled) setList({ id, controls: data.extended_controls ?? NONE }); })
      .catch(() => { if (!cancelled) setList({ id, controls: NONE }); });
    return () => { cancelled = true; };
  }, [id, size]);
  return list.id === id ? list.controls : NONE;
}
