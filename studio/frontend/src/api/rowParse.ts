import type { TextParts } from './pushStream.ts';

/** Change 0224 (trial U4): a keyed topic (the gateway and master point tables) arrives as a head, row
 *  texts and a tail. Parsing only the rows whose text is new keeps every unchanged row the same
 *  object, so a 5000-point table where 20 values moved parses 20 rows, and a row component that
 *  is memoised by its row object does not render again. Display only: the values are the engine's
 *  exact JSON, the same as parsing the whole text. */
export type RowParser = (parts: TextParts) => unknown;

// The list member the head ends with, for example `{"points":[` (top level only: checked below).
const LIST_START = /"((?:[^"\\]|\\.)*)"\s*:\s*\[\s*$/;

/** A parser for one path; it remembers the row objects of its last answer. Returns undefined when
 *  the parts do not look like a top-level list (the caller then parses the whole text). */
export function createRowParser(): RowParser {
  let seen = new Map<string, unknown>();
  return parts => {
    const found = LIST_START.exec(parts.head);
    if (!found) return undefined;
    const member = JSON.parse(`"${found[1]}"`) as string;
    const value = JSON.parse(`${parts.head}${parts.tail}`) as Record<string, unknown> | null;
    const list = value && typeof value === 'object' ? value[member] : undefined;
    if (!Array.isArray(list) || list.length) return undefined;
    const next = new Map<string, unknown>();
    value![member] = parts.rows.map(text => {
      const row = seen.has(text) ? seen.get(text) : JSON.parse(text);
      next.set(text, row);
      return row;
    });
    seen = next;
    return value;
  };
}
