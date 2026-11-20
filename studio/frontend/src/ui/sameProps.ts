import { useRef } from 'react';

/** React.memo comparison for table rows (change 0224, trial U4): the row renders again only when a
 *  value it shows changed (compared with Object.is). Functions are not compared, so every handler a
 *  memoised row gets must give the same result when an older copy of it is called: it reads the
 *  latest list through a ref or uses a functional state update, and anything it captures that can
 *  change (an index, a flag) is also passed as a plain prop. */
export function sameData<P extends object>(before: P, after: P): boolean {
  const keys = Object.keys(after) as (keyof P)[];
  if (keys.length !== Object.keys(before).length) return false;
  return keys.every(key => typeof after[key] === 'function' || Object.is(before[key], after[key]));
}

/** `list`, or the previous list when it holds the very same items in the same order, so a memoised
 *  child that gets it does not render again when nothing in it changed. */
export function useSameList<T>(list: readonly T[]): readonly T[] {
  const kept = useRef(list);
  const before = kept.current;
  if (before !== list && (before.length !== list.length || list.some((item, at) => item !== before[at]))) kept.current = list;
  return kept.current;
}
