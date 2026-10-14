# 0067: Bay templates

Date: 2026-10-14. Owner request: one-click Line feeder, Transformer bay and Bus coupler.
Specification: [Bay templates](../../specs/bay-templates.md) (BAY-01..04).
Decision: [ADR-0044](../../decisions/ADR-0044-bay-templates.md).

## Change

- `equipment/bayTemplates.ts`: templates, bay numbering and attaching to busbars.
- `AddMenu.tsx`: "Whole bays" group with an attach hint.
- `DiagramPage.tsx`: adds a bay attached to the selected junctions.
- `Canvas.tsx`: drop support.
- `tests/frontend/bayTemplates.test.ts`

## Evidence

**Tests:** 4 frontend tests (equipment, no links, no overlap, fresh ids and numbering,
attaching, one undo step, tidies cleanly). All frontend tests: 87 passing.

**Browser:** with BB1 selected, Add › Line feeder hung F1-Q1…F1-Line straight under the
junction, and Add › Transformer bay did the same under the second junction. Each was one
saved revision.

## Limitations

- Bays aren't linked to IEDs; the sheet's "needs a link" list guides that.
- No double-busbar feeder template yet.
- The owner's visual acceptance is pending.
