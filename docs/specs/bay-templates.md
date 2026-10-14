# Spec: Bay templates

Decision: [ADR-0044](../decisions/ADR-0044-bay-templates.md). Change: [0067](../changes/2026/0067-bay-templates.md).
Owner decision (2026-10-14): first version has Line feeder, Transformer bay and Bus coupler.

## Purpose

One click adds a whole standard bay, wired and laid out, so engineers do not draw
every disconnector and wire by hand.

## Behaviour contract

- `equipment/bayTemplates.ts` (pure) holds the three templates:
  - **Line feeder**: Q1 bus DS → Q0 CB → T1 CT → Q9 line DS → junction → line Infeed
    (dead), with Q8 earth switch and T5 VT on the junction.
  - **Transformer bay**: Q1 → Q0 → T1 → junction → transformer Infeed (live), with Q8.
  - **Bus coupler**: Q1 and Q2 disconnectors with Q0 breaker between them.
- Labels are `<prefix><n>-<code>` (F for feeder, T for transformer, C for coupler) with
  the next free bay number. Every item gets a fresh id. Nothing is linked to an IED.
- Diagram › Add has a "Whole bays" group (click or drag onto the canvas). With busbar
  junctions selected, the bay's top disconnector(s) are wired to them and the bay hangs
  under the first one; the hint above the tiles says so.
- Adding a bay is one edit: one undo step, revision-guarded save, locked while running.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| BAY-01 | Each bay holds the agreed equipment, no IED links and no overlapping symbols. | `studio/tests/frontend/bayTemplates.test.ts::each bay holds the agreed equipment and no links` |
| BAY-02 | Every added item gets a fresh id; labels use the next free bay number. | `studio/tests/frontend/bayTemplates.test.ts::bays get fresh ids and the next bay number` |
| BAY-03 | A bay connects to the selected busbar junction(s) and hangs under the first. | `studio/tests/frontend/bayTemplates.test.ts::a bay connects to the selected busbar points` |
| BAY-04 | Adding a bay is one undo step, and the result tidies without overlaps. | `studio/tests/frontend/bayTemplates.test.ts::adding a bay is one undo step and tidies cleanly` |

## Acceptance evidence

Change [0067](../changes/2026/0067-bay-templates.md): in an isolated browser instance a
selected busbar junction plus Add › Line feeder and Add › Transformer bay produced
straight bays wired to the bar, saved as one revision each. Pending: owner acceptance.
