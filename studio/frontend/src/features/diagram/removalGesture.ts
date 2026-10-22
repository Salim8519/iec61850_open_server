import type { Diagram } from '../../api/types';
import type { DiagramStep } from './canvasChanges.ts';

export type Removal = { diagram: Diagram; group: string };

let serial = 0;

/** Deleting a wired item makes React Flow call onEdgesChange (the wires) and then onNodesChange (the
 * item) in the same tick. Taken one by one that is two history steps, so one Undo brought the item
 * back without its wires, and the second call built on the diagram from before the first, so a wire
 * deleted together with the item could come back.
 *
 * A gesture joins the removals of one tick: each builds on the result of the one before and all share
 * one history group, so the deletion is one Undo step. The next tick starts a new group (the counter
 * is shared by every gesture so a group name never repeats), so two deletions never merge. */
export function createRemovalGesture(endOfTick: (end: () => void) => void = queueMicrotask) {
  let open: Removal | null = null;
  return {
    /** Apply `step` to the diagram this tick has so far (`diagram` when it is the first removal). */
    apply(diagram: Diagram, step: DiagramStep): Removal {
      if (!open) {
        serial += 1;
        open = { diagram, group: `remove:${serial}` };
        endOfTick(() => { open = null; });
      }
      open = { group: open.group, diagram: step(open.diagram) };
      return open;
    },
  };
}
