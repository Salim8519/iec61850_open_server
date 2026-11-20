import { useEffect, useRef } from 'react';
import { liveFeed } from './polls';

/** Ask the live feed for `path` once more whenever `key` changes (for example the project
 *  revision), without subscribing again (change 0224, trial U2). Subscribing again sent a request
 *  and a new push subscription on every diagram edit; `liveFeed.again` sends nothing for a path
 *  the push stream carries (the engine re-sends it after a project change when it changed) and
 *  asks a polled path once, as before. Nothing happens on the first render or when `path` changes
 *  (the view's own subscription asks then). */
export function useAskAgain(path: string | null, key: unknown) {
  const seen = useRef({ path, key });
  useEffect(() => {
    const before = seen.current;
    seen.current = { path, key };
    if (path && before.path === path && !Object.is(before.key, key)) liveFeed.again(path);
  }, [path, key]);
}
