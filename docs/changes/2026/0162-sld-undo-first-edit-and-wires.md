# 0162: SLD Undo: the first edit is undoable; a deleted item comes back with its wires in one step

Date: 2026-10-22. Found by the UI review [0160](0160-ui-review-no-clipping.md). Frontend only.

## Why

- The first diagram edit after opening the page could not be undone, including "Build from SCD › Replace
  my drawing", whose dialog promises "Undo brings them back". On every connect the push stream replays
  older `project_changed` events of earlier tabs; each one cleared the history identity (and cancelled a
  pending diagram save), and the next edit then rebuilt the history from the already-edited diagram.
- React Flow reports a deleted item's wires and the item in two calls, so one Delete made two Undo
  steps; a wired item deleted together with an unrelated selected wire brought that wire back.

## What changed

- `state/projectChange.ts`: a `project_changed` event is `own`, `replayed` (already in the last project
  read) or `other`; only `other` drops the history and the pending save and reloads.
- `state/historySession.ts`: pure history helpers; a local edit never starts a new history.
- `features/diagram/removalGesture.ts`, `canvasChanges.ts`, `Canvas.tsx`: removals reported in one tick
  build on each other and share one history group; the next tick starts a new one.

## Evidence

- New `historySession.test.ts` (7) and `removalGesture.test.ts` (8); frontend 372/372 merged; tsc clean.
- Worker Playwright run on the demo station (46 items, 48 wires): unfixed code 20/32 checks, fixed 35/35
  (add then Undo; Replace then Undo; delete wired item then one Undo gives 46/48; two deletions are two
  steps; two-tab change still resets the other tab's history).
- Merged-branch check: build from SCD (46/48), Undo enabled at once; delete a wired breaker (45/46), one
  Undo gives 46/48.
