// Counts for the type chips of the SCADA lists (ADR-0095). Display only, on rows the page already has.
import { countKinds, type Kind } from '../../../points/pointKind.ts';
import { ALL_KINDS } from '../../../ui/kindFilter.ts';

/** The `counts` the FilterBar wants: the "All types" chip shows every row, each kind chip its own number.
 *  Kinds with no row are left out, so their chips hide. */
export function kindCounts<T>(rows: readonly T[], kindOf: (row: T) => Kind, key = 'kind'): Record<string, Record<string, number>> {
  const counts: Record<string, number> = { [ALL_KINDS]: rows.length };
  for (const [kind, count] of Object.entries(countKinds(rows, kindOf))) if (count) counts[kind] = count;
  return { [key]: counts };
}
