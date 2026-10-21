/** Choosing several rows of a table to delete them together (Select mode). */
export type Picked = ReadonlySet<string>;

export function togglePick(picked: Picked, key: string): Set<string> {
  const next = new Set(picked);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  return next;
}

/** The header box: ticks every row shown by the filter (all pages), or clears them when all are ticked. */
export function toggleShown(picked: Picked, shown: readonly string[]): Set<string> {
  const all = shown.length > 0 && shown.every(key => picked.has(key));
  const next = new Set(picked);
  for (const key of shown) {
    if (all) next.delete(key);
    else next.add(key);
  }
  return next;
}

/** Ticked rows that are still in the table (a row removed elsewhere drops out of the count). */
export function stillThere(picked: Picked, keys: readonly string[]): string[] {
  return keys.filter(key => picked.has(key));
}

export function plural(count: number, noun: string) {
  return `${count.toLocaleString('en-US')} ${count === 1 ? noun : `${noun}s`}`;
}
