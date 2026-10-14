# 0062: Test mode and simulation

Date: 2026-10-14. Roadmap feature D2. Specification: [Test mode](../../specs/test-mode.md).
Decision: [ADR-0039](../../decisions/ADR-0039-test-mode.md).

## Change

**Backend**
- `domain/test_mode.py`: Beh derivation, mode points, validation, writes and summary.
- `domain/mode_badges.py`: device and equipment badges, and simulation sources.
- `simulation/test_mode_apply.py`: applies modes at Run and on change.
- `PUT /ieds/<id>/test-mode`, accepted for simulated IEDs only.
- `/health` now includes `modes` and `simulation_sources`.
- `test_mode` is saved per IED and survives other setting edits.

**Native (`worker/updates/values.cpp`):** integers are written with the setter of the
declared type, with range checks and a read-back check.

**Frontend**
- A test-mode card in the device overview: editable for simulated IEDs, read-only badges
  for real ones.
- A mode badge on device rows and on bound diagram equipment.
- A "Simulation traffic present" notice in Health check.

## Evidence

- **Tests:**
  - pytest: 243 unit and integration passing;
  - protocol: 38 passing with the new build and the VMware adapter, including
    `test_test_mode_runtime.py`;
  - frontend: 69 passing.
- **Isolated run, IED1_XCBR simulated:**
  - With test/blocked saved, `studio_probe` reads `Beh` = 4 on LLN0 and XCBR1 right after
    Run.
  - Switching to Test while running reads 3.
  - The device row, the QA1/QB1 cards and the test-mode card all show the mode as text.

**Bug found and fixed:** before this change, every INT8/16/32 and Enum write to a
simulated IED stayed at its old value. The Python side reported success, but the native
model kept the value; `studio_probe` read 0 after writing 4.

## Limitations

- Studio can't set the simulation bit on the GOOSE its simulated IEDs publish
  (TMODE-06, retired).
- Simulated IEDs don't block their controls by mode beyond what libiec61850 does.
- No real-relay evidence of mode reading.
