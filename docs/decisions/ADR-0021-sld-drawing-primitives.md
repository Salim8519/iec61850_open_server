# ADR-0021: SLD drawing primitives

Accepted: 2026-09-17.

The single-line diagram editor gains drawing primitives that reuse the existing
React Flow equipment engine instead of introducing a second canvas model:

- A new inert `JUNCTION` equipment kind is a small connection point with four
  terminals. It carries no state mapping, control binding, start position or
  rotation; the backend rejects any such binding on a junction. Junctions may
  dangle (no minimum degree), because drawing is free-form.
- All equipment nodes gain east/west terminals beside the existing north/south
  ones, and the canvas uses loose connection mode so any terminal may connect to
  any terminal. Existing saved wires are unaffected.
- A selected busbar exposes drag resize handles. The drawn length persists as
  `EquipmentData.width` (60–2400 px) in the same diagram payload; the backend
  rejects a width on any other kind. Resizing emits the same revision-guarded,
  undo-grouped diagram updates as dragging equipment, and is disabled while a
  run is active by the existing controlled-flow lock.

No migration is needed: `width` is optional and older projects remain valid.
The junction counts as canvas equipment for counts and bulk deletion review,
which treats it like physical drawing equipment that is preserved. No runtime,
native, SCL import/export or HTTP API behavior changes; the width and junction
are engineering-only projection data.

Rejected alternative: a separate free-draw overlay or edge waypoint model.
It would duplicate the diagram engine, double persistence and undo code, and
leave live locking inconsistent between the two layers.
