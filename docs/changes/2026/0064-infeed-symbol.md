# 0064: Infeed symbol

Date: 2026-10-14. Owner request: a power-source symbol for live colouring that can't
corrupt any engine. Specification: [Live colouring](../../specs/energization.md) (INFEED-01..04).
Decision: [ADR-0041](../../decisions/ADR-0041-infeed-symbol.md).

## Change

**Backend**
- `contracts/project.py`: kind `INFEED`, plus `infeed_type` and `infeed_state`.
- `projects/validation.py`: drawing-only rules, defaults, and infeed keys dropped on
  other kinds.

**Frontend**
- `api/types.ts`, `kindInfo.ts`, `registry.ts`.
- `EquipmentSymbol.tsx`: line, transformer and grid glyphs.
- `EquipmentNode.tsx`: "Set live" / "Set dead" pill.
- `AddMenu.tsx`: "Power in and drawing" group.
- Sheet fields in `equipmentFields.ts` / `SheetSettings.tsx`.
- `copy/hints.ts`: `infeed.intro`.
- `tidy.ts`: parking order.

**Tests:** `tests/unit/test_infeed.py`.

## Evidence

- 4 Infeed tests:
  - SCL export is byte-identical with Infeeds on the diagram;
  - SLD import of both repository SCDs produces no Infeed;
  - none of the ten existing kinds stores an infeed key.
- All backend tests pass (287, 7 skipped as before).

## Limitations

- An older build can't open a project that contains an Infeed (the validation error is
  kept by startup recovery).
- The owner's visual acceptance is pending.
