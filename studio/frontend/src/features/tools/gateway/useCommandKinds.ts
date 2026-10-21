import { useEffect, useState } from 'react';
import { gatewayApi, type Protocol } from '../../../api/gateway';
import { kindsById, type LoggedCommand, type PointKinds } from './commandKinds.ts';

const NONE: PointKinds = new Map();

/** Which type each logged command's point has. The point list is asked once, and again only when a
 *  new command names a point that is not known yet (no polling: the list can have thousands of rows). */
export function useCommandKinds(protocol: Protocol, entries: LoggedCommand[]): PointKinds {
  const [known, setKnown] = useState<{ protocol: Protocol; kinds: PointKinds }>({ protocol, kinds: NONE });
  const newest = entries.length ? entries[entries.length - 1].seq : 0;
  useEffect(() => {
    const kinds = known.protocol === protocol ? known.kinds : NONE;
    if (!entries.some(entry => entry.point_id && !kinds.has(entry.point_id))) return undefined;
    let alive = true;
    gatewayApi.points(protocol).then(answer => { if (alive) setKnown({ protocol, kinds: kindsById(answer.points) }); })
      .catch(() => {});  // display only: without the list the badge says "unknown type"
    return () => { alive = false; };
  }, [protocol, newest]);
  return known.protocol === protocol ? known.kinds : NONE;
}
