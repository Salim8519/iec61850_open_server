# Spec: Run one IED, equipment test buttons

Decision: [ADR-0067](../decisions/ADR-0067-run-single-ied-and-test-buttons.md).
Change: [0097](../changes/2026/0097-clean-sld-test-buttons-run-one.md).
Owner request (2026-10-18): run or connect a single IED from Devices; test every equipment
kind on simulated IEDs without needing a command link.

## Purpose

Start, stop or connect one IED while the others keep their state, and move simulated plant
(breaker positions, relay signals, measurements) from the SLD as if it changed by itself.

## Behaviour contract

**Run model** (`simulation/coordinator.py`, `simulation/run_selection.py`, `runtime/supervisor.py`,
`acquisition/manager.py`, `POST /runtime/ieds/<id>/start|stop`)
- The top bar *Run* still starts every IED that is not *Off*; *Stop* stops everything. During a
  partial run *Run all* adds every stopped IED that is not *Off* to the run (change 0106).
- Devices header: a simulated IED shows *Run this IED* / *Stop this IED* (its MMS server, reports,
  polling answers, GOOSE in lab mode and all its points); a real IED shows *Connect* /
  *Disconnect* (client read: free report blocks, else polling; sends nothing unless operated).
- One run holds a set of running IEDs (`running_ieds` in `GET /project`). Starting an IED while
  stopped starts the run with only that IED; while running it joins the run. Stopping an IED
  removes only its worker or live session, signal index, sources, sample lock and MMS client
  list; stopping the last one stops the run.
- Endpoint conflicts are checked against every running IED plus the new one. A new worker gets
  its loopback address without changing the addresses of running IEDs.
- Bindings, saved start positions and test modes are applied only to the IED that joins.
- Equipment bound to a stopped IED has no value; its SLD tag says "not running" and the test
  section says "IED not running".
- The top bar says "Running · 2 IEDs" or "Running 1 of 2 IEDs".
- Three run shapes never look alike (change 0106): ○ stopped, ◐ some IEDs (amber dashed run state,
  white *Stop all* and *Run all*, half-filled sidebar dots), ● all IEDs (green). Devices shows one
  line naming the live IEDs with *Run the other N* and lists running IEDs first; the SLD lock badge
  names the live IEDs, and equipment whose state comes only from stopped IEDs is grey and dashed with
  "IED stopped". Wire colours stay the backend's (an unknown switch gives "Maybe live").
- Running any IED locks the drawing and the IED roles, exactly as a full Run does (one rule,
  no partial edit mode). Stop everything to edit.

**Test buttons** (`domain/equipment_test.py`, `simulation/equipment_test.py`,
`POST /equipment/<node>/test`, `features/diagram/EquipmentTest.tsx`, `TestMenu.tsx`)
- Shown only while running, only when every bound IED is simulated and running; hidden for real
  IEDs (they use their command link). In the side sheet and the right-click menu.
- CB/DS/ES: Open · Closed · Moving · Bad state (Dbpos 01/10/00/11; contacts or single points
  are written to match). Relay panel: each row on/off. Trip relay: Trip · Reset (by its state
  polarity). CT/VT: Set value · Bad quality (validity invalid). Source: Live · Dead only when
  bound. Junction: none. Push button: unchanged.
- The write goes through the same sample writer as Devices data-point editing (source
  `equipment_test`); no MMS command is sent and the browser decides nothing.
- A command link is optional: the sheet shows the calm hint "Add a command link to operate it
  like SCADA" instead of a "Needs: command" warning.

## Out of scope

Editing the drawing or roles while some IEDs run; test buttons on real devices.

## Rules

| ID | Rule | Enforced by |
|---|---|---|
| RUNONE-01 | One IED runs alone; an independent client reaches only running IEDs; stopping one keeps the other. | `studio/tests/protocol/test_run_single_ied.py::test_one_ied_runs_alone_then_joins_and_leaves` |
| RUNONE-02 | The top bar counts running IEDs; Devices calls the single-IED endpoint. | `studio/tests/frontend/testButtons.test.ts::the top bar says how many IEDs run` |
| RUNONE-03 | Each equipment kind gets its test buttons; no binding means no buttons. | `studio/tests/unit/test_equipment_test_buttons.py::test_switch_positions_write_dbpos`, `studio/tests/unit/test_equipment_test_buttons.py::test_relay_rows_trip_and_measurements`, `studio/tests/frontend/testButtons.test.ts::every equipment kind gets its test buttons (change 0097)` |
| RUNONE-04 | A test button changes the simulated IED over MMS without any command; real IEDs are refused. | `studio/tests/protocol/test_run_single_ied.py::test_test_button_moves_breaker_without_command` |
| RUNONE-05 | Test buttons only for running simulated IEDs. | `studio/tests/frontend/testButtons.test.ts::test buttons only for running simulated IEDs; real IEDs use commands` |
| RUNONE-06 | Devices status pills sit on their own line so they are never cut. | `studio/tests/frontend/testButtons.test.ts::Devices rows put status pills on their own line so nothing is cut` |
| RUNONE-07 | Stopped, partial and full runs have distinct shapes and words; green only when all run; Devices groups running IEDs first; SLD greys stopped IEDs' equipment in words. | `studio/tests/frontend/runShape.test.ts::the run has three shapes and left-out IEDs never count`, `studio/tests/frontend/runShape.test.ts::Devices groups running IEDs first only during a partial run`, `studio/tests/frontend/runShape.test.ts::top bar is green only when all run; SLD greys equipment of stopped IEDs in words` |
| RUNONE-08 | *Run all* during a partial run adds the remaining IEDs; with nothing left it is refused. | `studio/tests/protocol/test_run_single_ied.py::test_run_all_during_a_partial_run_adds_the_rest` |

## Acceptance evidence

Change [0097](../changes/2026/0097-clean-sld-test-buttons-run-one.md): protocol tests with
`studio_probe`, and the dev app in the browser with the public samples only.
