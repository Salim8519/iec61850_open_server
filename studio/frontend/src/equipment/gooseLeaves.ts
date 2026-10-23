import type { GooseMembers } from '../api/gooseMessages';

/** Which messages carry a changed value (ADR-0112): IED id -> exact leaf ref -> gocbRefs, from Go's
 * GET /goose/members (functional constraint aware). Built once per answer; looked up per event. */
export type LeafIndex = Map<string, Map<string, string[]>>;

export function leafIndex(members: GooseMembers | null | undefined): LeafIndex {
  const index: LeafIndex = new Map();
  for (const [iedId, blocks] of Object.entries(members?.ieds ?? {})) {
    const refs = new Map<string, string[]>();
    for (const [gocbRef, leaves] of Object.entries(blocks)) {
      for (const ref of leaves) refs.set(ref, [...(refs.get(ref) ?? []), gocbRef]);
    }
    index.set(iedId, refs);
  }
  return index;
}

/** The messages one leaf of one IED is sent in ([] when none). */
export const blocksOf = (index: LeafIndex, iedId: string, ref: string): string[] => index.get(iedId)?.get(ref) ?? [];

/** The messages a point row's value is sent in (ADR-0112): its exact leaf, or every leaf under it
 * when the row names a data object ("XCBR1.Pos" holds "XCBR1.Pos.stVal"). [] when none. */
export function messagesOfRef(index: LeafIndex, iedId: string | null | undefined, ref: string | null | undefined): string[] {
  const refs = iedId && ref ? index.get(iedId) : undefined;
  if (!refs || !ref) return [];
  const exact = refs.get(ref);
  if (exact) return exact;
  const found = new Set<string>();
  for (const [leaf, blocks] of refs) if (leaf.startsWith(`${ref}.`)) blocks.forEach(block => found.add(block));
  return [...found];
}
