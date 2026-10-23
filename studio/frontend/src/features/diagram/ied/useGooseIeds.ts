import { useCallback, useEffect, useMemo, useState } from 'react';
import { getGooseProposal } from '../../../api/gooseMessages';
import type { Diagram } from '../../../api/types';
import { GOOSE_IEDS_COPY } from '../../../copy/iedComponent';
import { nextSlot, withIedComponents, type Placement } from './iedNodes';

export type GooseIedsNote = { text: string; ids: string[] };

/** "Add all GOOSE IEDs" and the toolbar list of GOOSE IEDs not on the drawing (ADR-0112). Go decides
 * which IEDs and where; the drawing is saved first so Go sees the latest edit. One edit, one Undo. */
export function useGooseIeds({ diagram, edit, select, prepare, revision, watch }: {
  diagram: Diagram; edit: (next: Diagram) => void; select: (ids: string[]) => void;
  prepare: () => Promise<unknown>; revision: number; watch: boolean;
}) {
  const [missing, setMissing] = useState<Placement[]>([]);
  const [note, setNote] = useState<GooseIedsNote | null>(null);
  useEffect(() => {
    if (!watch) { setMissing([]); return undefined; }
    let live = true;
    getGooseProposal().then(answer => { if (live) setMissing(answer.nodes); }).catch(() => { if (live) setMissing([]); });
    return () => { live = false; };
  }, [watch, revision]);
  const place = useCallback(async (only?: string) => {
    try {
      await prepare();
      const proposal = (await getGooseProposal()).nodes;
      const chosen = only ? nextSlot(proposal, only) : proposal;
      const result = withIedComponents(diagram, chosen, () => crypto.randomUUID());
      if (!result.ids.length) { setNote({ text: GOOSE_IEDS_COPY.none, ids: [] }); return; }
      edit(result.diagram); select(result.ids);
      setNote({ text: GOOSE_IEDS_COPY.added(result.ids.length), ids: result.ids });
    } catch (reason) {
      setNote({ text: reason instanceof Error ? reason.message : String(reason), ids: [] });
    }
  }, [diagram, edit, prepare, select]);
  const addAll = useCallback(() => { void place(); }, [place]);
  const addOne = useCallback((iedId: string) => { void place(iedId); }, [place]);
  const clearNote = useCallback(() => setNote(null), []);
  return useMemo(() => ({ missing, note, addAll, addOne, clearNote }), [missing, note, addAll, addOne, clearNote]);
}
