# Spec: Test mode and simulation

Decision: [ADR-0039](../decisions/ADR-0039-test-mode.md). Change: [0062](../changes/2026/0062-test-mode.md).
GOOSE simulation-bit publishing stays out of scope in [GOOSE](goose.md); see TMODE-06.

## Purpose

IEC 61850 Ed2 lets parts of a live station be tested safely: logical devices are put
in *test* or *blocked* mode, and test equipment sends traffic marked as *simulation*.
Studio makes this state visible for the whole station and warns about dangerous
leftovers, such as a relay still in test or simulation frames still on the network. It
also lets Studio's simulated IEDs act as properly marked test sources.

## Behaviour contract

**Reading mode from real IEDs** (read-only, ADR-0031)
- For each logical device, Studio reads `LLN0.Mod.stVal` and `LLN0.Beh.stVal`, and it
  reads `LPHD.Sim.stVal` once per IED. All three use the existing read operation.
- The values are shown as text using the standard names:
  1 on, 2 on-blocked, 3 test, 4 test/blocked, 5 off.
- Anything other than *on* is shown as a badge with text and an icon, never colour
  alone. The badge appears on the device row, in Station health and on diagram
  equipment bound to that logical device.
- A missing data object (for example an Ed1 IED without `Beh`) reads "not available in
  this IED", never *on*.

**Simulation-flagged traffic on the wire**
- GOOSE frames with the simulation/test bit and SV frames with the simulation bit are
  shown per stream as "Simulation traffic".
- Station health shows the notice **"Simulation traffic present"**, naming every source
  (GOOSE or SV, in the project or not). Leftover test traffic is a classic commissioning
  risk. When the source is an IED in the project, that IED's row is also a Warning, so the
  station verdict stays the worst IED verdict (HEALTH-03).
- When the subscriber named in the SCD is a real IED whose `LPHD.Sim` is false, the hint
  says the frames should be ignored by that IED. Studio doesn't claim what the IED
  actually did.

**Studio's simulated IEDs**
- For each simulated IED the user can set:
  - `Mod` for each logical device (on, test, test/blocked, off);
  - `LPHD.Sim` (true or false).
- `Beh` is derived from `Mod` following IEC 61850-7-4 (an LD's `Beh` takes the
  "highest" of its own `Mod` and its parent's).
- Mode and simulation settings are saved per IED in the project (no format change).
  They are written at Run and immediately when changed while running.
- **Mark GOOSE as simulation** is retired: libiec61850 1.6.1 has no API to set the
  simulation bit on server-published GOOSE, and vendor sources stay unmodified.
- None of these settings ever change a real IED. A real IED offers no such controls.
- How a simulated IED treats controls in test or off mode follows libiec61850's
  behaviour. Studio states what the stack does and does not add its own blocking logic.

## Out of scope

- Writing `Mod` or `Sim` to real IEDs.
- Test-command (`Test=true`) automation.
- Protection test sequences or COMTRADE playback.
- Mirroring a real IED into a simulated twin.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| TMODE-01 | Mode and simulation status of real IEDs are only read, never written. | `studio/tests/unit/test_live_contract.py::test_mode_status_is_read_only` |
| TMODE-02 | `Mod`/`Beh` values 1–5 are shown with their standard names; any other value is shown raw and marked unknown. | `studio/tests/unit/test_test_mode.py::test_mode_values_have_standard_names` |
| TMODE-03 | A missing `Mod`, `Beh` or `Sim` reads "not available in this IED", never *on*. | `studio/tests/unit/test_test_mode.py::test_missing_mode_is_not_on` |
| TMODE-04 | Any simulation-flagged GOOSE or SV frame raises the notice "Simulation traffic present", naming its source. | `studio/tests/unit/test_test_mode.py::test_simulation_traffic_warns` |
| TMODE-05 | A simulated IED's `Beh` is derived from `Mod` according to IEC 61850-7-4 and can't be set directly. | `studio/tests/unit/test_test_mode.py::test_beh_is_derived_from_mod` |
| TMODE-06 | Retired: libiec61850 1.6.1 exposes no simulation-bit setter for server GOOSE and vendor sources stay unmodified (ADR-0039). | — |
| TMODE-07 | Real IEDs offer no Mod, Sim or simulation-marking controls. | `studio/tests/frontend/testMode.test.ts::real ied has no mode controls` |
| TMODE-08 | Mode badges use text and an icon, and appear on device rows, health rows and bound diagram equipment. | `studio/tests/frontend/testMode.test.ts::mode badges have text and icon` |
| TMODE-09 | Settings apply only to simulated IEDs, persist in the project and survive other IED edits; real IEDs are refused. | `studio/tests/integration/test_test_mode_api.py::test_real_ied_is_refused_and_settings_persist` |
| TMODE-10 | INT8/16/32 and Enum attributes of simulated IEDs are really written (read back by an independent client); out-of-range values are refused. | `studio/tests/protocol/test_test_mode_runtime.py::test_integer_and_enum_attributes_are_written` |
| TMODE-11 | Mode writes reach the simulated IED and are read back by an independent client. | `studio/tests/protocol/test_test_mode_runtime.py::test_mode_writes_reach_the_simulated_ied` |
| TMODE-12 | Equipment badges follow the logical device of the equipment's binding; normal service shows none. | `studio/tests/unit/test_test_mode.py::test_equipment_badges_follow_bound_logical_device` |

## Acceptance evidence

Recorded in change [0062](../changes/2026/0062-test-mode.md):
- Isolated Studio run, IED1_XCBR simulated:
  - with the logical device set to test/blocked before Run, `studio_probe` reads
    `Beh` = 4 on LLN0 and XCBR1 right after start;
  - switching to Test while running reads 3.
  - The device row, the QA1/QB1 diagram cards and the test-mode card all show the mode
    as text.
- While building this, a native bug was found and fixed: integer attributes narrower
  than 64 bits (INT8/16/32, Enum, INT8U/16U) were silently never written.
- GOOSE simulation-bit publishing is retired (TMODE-06). Simulation detection on
  received GOOSE and SV is covered by HEALTH and SV-12 evidence.
- Real-relay evidence of mode reading is still needed before any interoperability claim.

## Answered questions

1. Leftover simulation traffic is a **Warning** (intentional during commissioning).
2. No project-format bump: `test_mode` is an optional per-IED setting with a default.
   (Owner delegated the proposed answers, 2026-10-14.)
