import { useEffect, useState } from 'react';
import { liveFeed } from '../../api/polls';
import type { GooseShare } from '../../api/network';

/** True while the Go engine sends simulated GOOSE to the IED network (ADR-0119). No polling:
 * one read when Run starts, then the goose.share push frames keep it current. */
export function useGoesOut(running: boolean): boolean {
  const [state, setState] = useState('');
  useEffect(() => {
    if (!running) { setState(''); return undefined; }
    return liveFeed.subscribe<GooseShare>('/goose/share', null, share => setState(share.state));
  }, [running]);
  return state === 'sharing';
}
