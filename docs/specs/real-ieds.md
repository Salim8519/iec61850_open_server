# Spec: Real IEDs (live acquisition)

Decision: [ADR-0031](../decisions/ADR-0031-read-only-live-acquisition.md).
Owner policy (2026-10-13): use reports on free report blocks only, preview the block,
release it on Stop, fall back to polling. Owner decision 2026-10-17 (ADR-0057): commands
to real IEDs are allowed from Tools only (IED Explorer after a confirmation, and the gateway
while "Allow SCADA commands to real IEDs" is on), through the separate `studio_control`.
Owner decision 2026-11-11 (ADR-0138): a monitored real IED is read only until its own
"Allow commands" switch in Devices is on; the gateway needs both switches. On connect the
device's own control settings are compared with the CID. Owner decision 2026-11-23 (ADR-0147):
Devices sends every command class of the CID, one click or Select/Operate/Cancel step by step,
from the Commands pop-up or a Data points row, and explains refusals in words.

## Purpose

An IED in the project can be set to *Connect to real device*. Studio then connects to
the real IED as a client, shows its live values, and compares its data model with
the SCD so configuration differences are easy to find.

## Behaviour contract

- Roles per IED: `simulated` (default), `real`, `off`. A real IED never starts a
  simulated worker, and a real and a simulated IED never share an endpoint.
- The real endpoint is prefilled from the SCD `ConnectedAP/Address` and labelled
  "from SCD". A user override is stored separately and never rewrites the SCL.
- Access `auto_reports` (default): use a report block only when the IED shows it as
  not enabled, not reserved and without an owner; show which block before
  connecting; release it on Stop or disconnect. When no block is free, poll the
  watched signals instead and say why. Access `polling` never touches report blocks.
- `studio_live` has no control, setting or data write operation. Report-block
  reservation attributes (`Resv`, `RptEna`, `GI`) are its only writes.
- Values from the device keep the device's quality and timestamp and carry source
  `live_report`, `live_poll` or `live_goose`.
- `live.allow_commands` is off by default and stored only while on. IED Explorer does not
  use it: it connects by typed address and asks before every command.
- After connecting (and on Refresh health) the session reads `ctlModel`, `sboTimeout` of SBO
  controls and `Loc.stVal` of LLN0 and of each control's node, read only, and classes each
  CID control `match`, `differs` or `unreadable` (`GET /live/{id}/controls`).

## Out of scope

Commands to real IEDs from SLD and Test (Devices › Data points with Allow commands, or Tools, see [IED Explorer](ied-explorer.md) and
[Gateway simulator](gateway.md)), setting-group changes, file transfer, logs, and editing a
real IED's datasets or report blocks.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| LIVE-01 | `studio_live` exposes no control, setting or data write operation. | `studio/tests/unit/test_live_contract.py::test_operation_table_is_read_only`, `studio/tests/protocol/test_live_client.py::test_write_operations_do_not_exist`, `studio/tests/unit/test_live_contract.py::test_explorer_operations_use_only_read_services` |
| LIVE-02 | Only report blocks shown as free are used; a busy block is never taken. | `studio/tests/unit/test_access_policy.py::test_busy_blocks_fall_back_to_polling`, `studio/tests/protocol/test_live_client.py::test_enabled_block_is_not_taken` |
| LIVE-03 | Every block Studio reserved is released on disconnect. | `studio/tests/protocol/test_live_client.py::test_blocks_released_on_disconnect` |
| LIVE-04 | Operating equipment bound to a real IED is refused with a plain reason. | `studio/tests/unit/test_live_roles.py::test_operating_real_ied_is_refused` |
| LIVE-05 | A real IED is never started as a simulated worker. | `studio/tests/unit/test_live_roles.py::test_real_ied_is_not_simulated` |
| LIVE-06 | The comparison classifies every reference as match, missing on device, extra on device or type mismatch, and diffs datasets and report blocks. | `studio/tests/unit/test_model_compare.py::test_signal_categories`, `studio/tests/unit/test_model_compare.py::test_dataset_and_report_differences` |
| LIVE-07 | Live values keep device quality and timestamp and carry a live source; unreadable values are reported, not invented. | `studio/tests/unit/test_live_values.py::test_device_quality_and_source_are_kept`, `studio/tests/unit/test_live_values.py::test_read_errors_are_not_values` |
| LIVE-08 | The real endpoint is prefilled from the SCD and a user override is stored separately. | `studio/tests/unit/test_live_roles.py::test_endpoint_prefill_and_override` |
| LIVE-09 | Connection loss is an explicit state and values are marked stale, never frozen as fresh; a device that answers no point of a read counts as lost (Go). | `studio/tests/unit/test_live_values.py::test_connection_loss_marks_values_stale`, `studio/engine/internal/acquisition/silent_read_test.go::TestSilentDeviceMarksValuesStale`, `studio/engine/internal/acquisition/silent_read_test.go::TestPartlyAnsweringDeviceStaysLive` |
| LIVE-10 | Discovery returns the device model over MMS so it can be compared. | `studio/tests/protocol/test_live_client.py::test_discover_and_read_simulated_stand_in` |
| LIVE-11 | A real IED takes SCADA commands through the gateway only while its own Allow commands and the gateway's real-command switch are both on; Allow commands is off by default and stored only while on. | `studio/engine/internal/project/netset/allow_commands_test.go::TestAllowCommandsIsOffUntilTurnedOn`, `studio/engine/internal/gateway/service/real_commands_test.go::TestRealIEDCommandsNeedBothSwitches`, `studio/engine/internal/gateway/service/real_commands_test.go::TestRealIEDPointsAreMonitorOnlyUntilAllowed`, `studio/tests/frontend/iedRoles.test.ts::Monitor real device promises no commands until Allow commands is on` |
| LIVE-12 | On connect the device's own ctlModel, select time and Loc are read and every CID control is classed match, differs or unreadable; an unreadable value is never filled from the CID and a failed read never ends the session. | `studio/engine/internal/acquisition/control_check_test.go::TestControlCheckClassesEveryControl`, `studio/engine/internal/acquisition/control_check_test.go::TestControlCheckFailureNeverEndsTheSession`, `studio/engine/internal/acquisition/real_ied_test.go::TestRealIEDSession` |
| LIVE-13 | Every command read from SCL (public fixtures and the local vendor corpus) has a control model 1 to 4, a feedback point of the same object and a whole-ms select time; ctlModel on the instance wins, as a name or a number. | `studio/engine/internal/scl/controls_vendor_test.go::TestEveryVendorControlIsUsable`, `studio/engine/internal/scl/controls_vendor_test.go::TestControlModelOnTheInstanceWins` |
| LIVE-14 | Devices sends a command to a real IED only while its Allow commands is on and its session is live, only after the confirmation, with the device's own control model; refusals are in words and request, applied and observed stay apart. | `studio/engine/internal/realcmd/service_test.go::TestRefusalsAreInWords`, `studio/engine/internal/realcmd/service_test.go::TestCommandIsSentAndObserved`, `studio/tests/frontend/iedRoles.test.ts::Send buttons appear only while Allow commands is on and the device is live` |
| LIVE-15 | Stop or Disconnect while the live client still waits for a device that does not answer (connect, data model) ends the client at once, since no report block is reserved yet; once one may be reserved, "disconnect" gives every block back as before; Stop closes the sessions and the simulated IEDs together (change 0223). | `studio/engine/internal/acquisition/session_close_test.go::TestStopWhileConnectingKillsTheClient`, `studio/engine/internal/acquisition/session_close_test.go::TestStopAfterReservingStillDisconnects`, `studio/tools/trials/real_stop_ab.py` |
| LIVE-16 | Devices sends every CID command class (SPC, DPC, INC, ENC, APC, BAC, BSC, ISC) in every control model as one click or Select, Operate and Cancel steps; step by step, Operate needs its Select; the chosen card is used; a device refusal says what it means (interlocking, Local, Test mode, Off) before the device's own answer. | `studio/engine/internal/realcmd/wire_test.go::TestDevicesCommandsEveryClassOneClick`, `studio/engine/internal/realcmd/wire_test.go::TestDevicesCommandsStepByStepThroughACard`, `studio/engine/internal/realcmd/wire_test.go::TestDevicesInterlockRefusalInWords`, `studio/engine/internal/realcmd/steps_test.go::TestRefusalsAreExplained`, `studio/engine/internal/realcmd/steps_test.go::TestStepRefusals`, `studio/tests/frontend/commandSearch.test.ts::only the ctlVal row of Oper belongs to a command` |

Protocol evidence uses a separate `studio_ied.exe` as a stand-in device on this PC.
Interoperability with third-party IEDs remains an open release gate.
