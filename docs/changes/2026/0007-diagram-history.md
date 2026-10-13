# Change 0007: diagram undo/redo and initial native state

Date: 2026-09-09.

The diagram has Undo/Redo buttons and Ctrl+Z, Ctrl+Shift+Z and Ctrl+Y shortcuts.
Shortcuts do not intercept text fields, dialogs or running simulations. History
stores up to 50 engineering states, preserves identifiers, groups drag updates
and ignores selection-only changes. Opening a project or changing the project IED
set resets history, preventing undo from reviving references to a deleted IED.
Deleting a node also removes its incident wires from the saved diagram.

Three dependency-free TypeScript tests cover undo/redo, stale-redo invalidation,
drag grouping, selection-only changes, immutable snapshots and the history bound.
Command: `node --experimental-strip-types --test studio/tests/frontend/diagramHistory.test.ts`.
Frontend production build passes. Visual/interactive acceptance is still pending.

Starting a run now reads mapped values and their quality/time companions from
native workers before publishing the initial canvas state. Pause no longer clears
displayed live values. A native scenario test verifies the initial breaker state.
Failure while obtaining initial state rolls back workers and emits stopped state.

A previously flaky import test compared complete ZIP bytes, including changing
ZIP timestamps. It now compares decoded complete project payloads, preserving
the intended no-mutation assertion without depending on archive creation time.

Portable rebuild initially failed on a Windows permission error writing the
intermediate executable. A fresh intermediate build directory succeeded without
changing security settings; the frozen startup/security smoke test passed.
