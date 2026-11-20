import type { Diagram, Project } from '../api/types';
import { sameEngineering } from './diagramHistory.ts';

export const SAVED = 'All changes saved';
export const SAVING = 'Saving...';
export const NOT_SAVED = 'Not saved';

export type DiagramSaveDeps = {
  /** The project on screen now (the hook's `current` ref). */
  project: () => Project | null;
  /** Puts a project on screen. */
  show: (project: Project) => void;
  put: (body: { diagram: Diagram; revision: number; tab_id: string }) => Promise<{ revision: number }>;
  /** Reads the project again from the engine (only after a revision conflict). */
  reload: () => Promise<unknown>;
  status: (text: string) => void;
  error: (message: string) => void;
  tabId: string;
  delay?: number;
};

/** True when the engine refused the save because another tab changed the project first. */
export function isConflict(reason: unknown): boolean {
  return (reason as { data?: { code?: unknown } } | null)?.data?.code === 'conflict';
}

/** A project read keeps the drawing on screen while an edit of it is not saved yet (fix plan A1).
 *  The edit was made on the revision this tab knows, so that revision goes with it: a read that
 *  started before the last save cannot make the next save look like a conflict. */
export function keepLocalEdit(read: Project, local: Project | null, pending: boolean): Project {
  if (!pending || !local || local.session_id !== read.session_id) return read;
  return { ...read, diagram: local.diagram, revision: local.revision };
}

/** Saves diagram edits 400 ms after the last one, one request at a time, newest drawing wins.
 *  A failed save keeps the drawing and says "Not saved"; the next edit or Save tries again.
 *  Only a revision conflict reads the engine's drawing back (fix plan A2). */
export function createDiagramSave(deps: DiagramSaveDeps) {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let queue: Promise<void> = Promise.resolve();
  let generation = 0;
  let unsaved = false;
  let saved: Diagram | null = null;  // the drawing the engine last accepted from this tab
  const stop = () => { if (timer) clearTimeout(timer); timer = null; };

  async function save(): Promise<void> {
    // Nothing waits to be sent: only let a save already on its way finish. Sending the same drawing
    // again made the engine record an empty Undo step, so Undo right after a drag seemed to do
    // nothing the first time (found in trial U2, change 0224).
    if (!unsaved && !timer) { await queue.catch(() => {}); return; }
    stop();
    const start = deps.project();
    if (!start || start.runtime !== 'stopped') return;
    const diagram = start.diagram;
    const mine = generation;
    const run = queue = queue.catch(() => {}).then(async () => {
      if (mine !== generation || diagram === saved) return;  // the save ahead already sent this drawing
      const active = deps.project();
      if (!active || active.runtime !== 'stopped') return;
      try {
        const result = await deps.put({ diagram, revision: active.revision, tab_id: deps.tabId });
        saved = diagram;
        const now = deps.project();
        if (now?.session_id === active.session_id) {
          deps.show({ ...now, revision: result.revision });
          if (now.diagram === diagram) unsaved = false;
        }
      } catch (reason) {
        if (isConflict(reason)) {
          drop();
          await deps.reload().catch(() => {});  // the conflict stays the reason shown
        }
        throw reason;
      }
    });
    try { await run; } catch (reason) { deps.status(NOT_SAVED); throw reason; }
    if (mine === generation && !unsaved) deps.status(SAVED);
  }

  /** Forgets the pending edit: another tab changed the project, or another project replaced it. */
  function drop() { generation += 1; stop(); unsaved = false; }

  function edit(diagram: Diagram) {
    const active = deps.project();
    if (!active || active.runtime !== 'stopped') return;
    const unchanged = sameEngineering(diagram, active.diagram);
    deps.show({ ...active, diagram });
    if (unchanged) return;
    unsaved = true;
    deps.status(SAVING);
    stop();
    timer = setTimeout(() => save().catch((reason: Error) => deps.error(reason.message)), deps.delay ?? 400);
  }

  return { edit, save, drop, pending: () => unsaved, dispose: stop };
}
