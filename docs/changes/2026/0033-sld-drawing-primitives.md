# 0033: SLD drawing primitives

Date: 2026-09-17. User-requested SLD drawing capability while reusing the
existing diagram engine.

The substation canvas now supports free-form drawing with three additions,
all built on the existing React Flow equipment engine:

- A Junction palette item adds a small connection-point node with four
  terminals. It is a drawing element only: the backend rejects state or
  control mappings for it and the Inspector offers only name, duplicate and
  delete.
- Every equipment node gained east and west terminals and the canvas now uses
  loose connection mode, so wires may run in any direction between any
  terminals. Saved diagrams and existing north/south wiring are unchanged.
- A selected busbar shows drag resize handles; its drawn length persists as
  optional `width` data (60–2400 px) in the normal diagram payload with the
  same revision guard, save queue and undo/redo history as other edits.
  Resizing is blocked while a run is active by the existing structure lock.

Evidence:

- 7 new unit tests in `studio/tests/unit/test_diagram_drawing.py` pass:
  junction acceptance, drawn busbar width persistence, bounds, non-busbar
  width rejection and junction mapping rejection. Full unit suite: 97 pass.
- 17 integration tests pass, including diagram revision persistence.
- 14 frontend node tests pass, including 5 new `diagramResize.test.ts`
  tests covering bounded rounding, non-busbar isolation and identity
  stability for unchanged widths.
- `npm run check` and the Vite production build pass.
- `studio/tools/verify.py` passes file-length, architecture, documentation
  and history checks.

Visual acceptance with a running browser session and a saved/reopened
project round-trip of junctions and a resized busbar remain release gates.
