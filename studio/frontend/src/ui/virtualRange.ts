/** Which rows of a long list are on screen (speed lane F, virtual rows).
 *
 * Rows may wrap, so a row's height is the one measured on screen when known (`heightOf`) and
 * `rowHeight` until then. `overscan` extra rows above and below keep fast scrolling from showing
 * blanks. The spacer heights keep the scrollbar as tall as the full list. */
export type VirtualRange = { start: number; end: number; before: number; after: number };

export function virtualRange(input: { count: number; rowHeight: number; scrollTop: number; viewport: number;
  overscan?: number; heightOf?: (index: number) => number | undefined }): VirtualRange {
  const { count, rowHeight, heightOf } = input;
  const overscan = input.overscan ?? 8;
  if (count <= 0 || rowHeight <= 0) return { start: 0, end: 0, before: 0, after: 0 };
  const height = (index: number) => heightOf?.(index) ?? rowHeight;
  const top = Math.max(0, input.scrollTop);
  const bottom = top + Math.max(rowHeight, input.viewport);
  const offsets = new Array<number>(count + 1);
  offsets[0] = 0;
  for (let index = 0; index < count; index++) offsets[index + 1] = offsets[index] + height(index);
  let first = 0;
  while (first < count - 1 && offsets[first + 1] <= top) first++;
  let last = first;
  while (last < count - 1 && offsets[last + 1] < bottom) last++;
  const start = Math.max(0, first - overscan);
  const end = Math.min(count, last + 1 + overscan);
  return { start, end, before: offsets[start], after: offsets[count] - offsets[end] };
}

/** True when two ranges draw the same rows with the same spacers (no render needed). */
export function sameRange(a: VirtualRange, b: VirtualRange) {
  return a.start === b.start && a.end === b.end && a.before === b.before && a.after === b.after;
}
