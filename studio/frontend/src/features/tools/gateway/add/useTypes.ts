import { useEffect, useState } from 'react';
import type { Protocol } from '../../../../api/gateway';
import { addApi, type TypeEntry } from '../../../../api/gatewayAdd';

/** The engine's type catalog of a protocol (GET /tools/gateway/types); empty while loading. */
export function useTypes(protocol: Protocol, fail: (message: string) => void): TypeEntry[] {
  const [types, setTypes] = useState<TypeEntry[]>([]);
  useEffect(() => {
    let live = true;
    setTypes([]);
    addApi.types(protocol).then(answer => { if (live) setTypes(answer.types); }).catch(error => fail(error.message));
    return () => { live = false; };
  }, [protocol]);
  return types;
}
