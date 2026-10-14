# ADR-0044: Bay templates in the Add menu

Accepted: 2026-10-14. Owner request: one-click bays (Line feeder, Transformer bay, Bus
coupler). Specification: [Bay templates](../specs/bay-templates.md).

**Decision**
- `equipment/bayTemplates.ts` (pure) defines the three bays:
  - drawn on the Tidy up grid;
  - labelled `<prefix><n>-<code>`;
  - no IED links;
  - the feeder and transformer bays end in an Infeed (ADR-0041).
- Bays are added through the normal diagram edit, so each is one undo step. With busbar
  junctions selected, they are wired to them.
- Ids are mapped directly rather than through the clipboard paste, because paste
  renumbers labels (Q1 → Q2), which is right for copies but wrong for a new bay.

**Compatibility:** additive. A bay is ordinary diagram content.
