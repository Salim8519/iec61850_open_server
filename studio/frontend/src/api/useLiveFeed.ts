import { useEffect, useState } from 'react';
import { liveFeed } from './polls';

type Seen<T> = { path: string | null; value: T | null; error: string };

/** A view's live answer of GET `path` (null: nothing), every `every` ms (null: once).
 * Changing `again` asks once more (for example a new project revision). The component re-renders
 * only when the answer's text changed. */
export function useLiveFeed<T>(path: string | null, every: number | null, again?: unknown): { value: T | null; error: string } {
  const [seen, setSeen] = useState<Seen<T>>({ path, value: null, error: '' });
  useEffect(() => {
    if (!path) return undefined;
    return liveFeed.subscribe<T>(path, every, value => setSeen({ path, value, error: '' }),
      error => setSeen(current => ({ path, value: current.path === path ? current.value : null, error: error.message })));
  }, [path, every, again]);
  return seen.path === path ? seen : { value: null, error: '' };
}
