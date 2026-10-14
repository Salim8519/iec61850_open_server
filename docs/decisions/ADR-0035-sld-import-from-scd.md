# ADR-0035: Single-line diagram import from the SCD Substation section

Accepted: 2026-10-14. Owner goal: reach and exceed SCD-driven station test tools.
Specification: [SLD import](../specs/sld-import.md).

Engineers receive a whole-substation SCD whose `Substation` section already describes
voltage levels, bays, equipment, connectivity nodes and, in some files, drawing
coordinates (`esld`). Studio didn't read this section, so every diagram was drawn by
hand.

**Decision**
- **Layering:**
  - A read-only parser (`scl/substation.py`) turns the stored original SCL into a flat
    topology.
  - Pure domain modules (`domain/sld_layout.py`, `sld_auto_layout.py`,
    `sld_bindings.py`) turn it into an ordinary format-12 diagram.
- **No new kind, no format change:**
  - Busbars and tee points become rows of `JUNCTION` nodes.
  - Transformers, lines, arresters and other equipment are listed as *not drawn*.
- **Bindings use only `LNode`s written on the equipment:**
  - Position comes from `XCBR`/`XSWI.Pos`.
  - Commands come from `CSWI.Pos`, otherwise a controllable `XCBR`/`XSWI.Pos`.
  - Nothing is inferred from naming conventions.
- **Preview first:** `POST /project/diagram/from-scl/preview` never changes the project.
  The browser applies the returned diagram through the existing revision-guarded
  `PUT /project/diagram`, so the import is a single undo step.
- **Deterministic node ids** (a hash of source and SCL path) let "add beside" skip items
  already drawn.
- **Junction size:** the junction drawing point is rendered at its intended 40 px. The
  card stylesheet had sized it as a 108 px card, which made rows of junctions unreadable.

**Compatibility:** additive. There are no project-format, contract or worker changes.
Existing diagrams and hand-drawn junctions are unchanged, apart from the junction box
size on screen.
