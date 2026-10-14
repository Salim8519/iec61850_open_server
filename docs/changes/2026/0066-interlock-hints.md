# 0066: Friendly interlock hints

Date: 2026-10-14. Owner decision: hints only, friendly, never blocking.
Specification: [Interlock hints](../../specs/interlock-hints.md) (HINT-01..05).
Decision: [ADR-0043](../../decisions/ADR-0043-interlock-hints.md).

## Change

**Backend**
- `domain/interlock_hints.py`: load, unknown and earth rules.
- `energization_states.py`: hints in the event, `mark_own` and `observe`.
- `equipment_control.py`: marks Studio's own Operate.
- `native_control_feedback.py`: observes external commands.

**Frontend**
- `SwitchHints.tsx`: amber dashed box; identical open/close hints are merged.
- `EquipmentSheet.tsx`
- `copy/hints.ts`: `switch.hints`.

**Tests**
- `tests/unit/test_interlock_hints.py`
- `tests/protocol/test_interlock_hints_runtime.py`
- `tests/frontend/interlockHints.test.ts`

## Evidence

**Tests**
- 3 unit tests.
- 1 native protocol test:
  - Studio's own Operate isn't reported as external;
  - a `studio_probe` external open against a hint is executed (position becomes Open)
    and noted once in Activity.
- 1 frontend test: Open/Close are only disabled while busy.

**Browser**
- F1-Q1 showed "Before you open or close: Open F1-Q0 first…" with the disclaimer, and
  both buttons were enabled.
- Activity showed "An external client opened F1-Q1. Open F1-Q0 first…".

## Limitations

- These rules are generic (bay breaker, earthing), not a station's real interlocking
  logic.
- Without an origin identity, an external client is named "An external client".
- Real SCADA clients and the owner's acceptance are pending.
