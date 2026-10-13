# Spec: Real IEDs (live acquisition)

Decision: [ADR-0031](../decisions/ADR-0031-read-only-live-acquisition.md).
Owner policy (2026-10-13): use reports on free report blocks only, preview the block,
release it on Stop, fall back to polling, and never send commands to real IEDs.

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

## Out of scope

Commands to real IEDs, setting-group changes, file transfer, logs, and editing a
real IED's datasets or report blocks.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| LIVE-01 | `studio_live` exposes no control, setting or data write operation. | `studio/tests/unit/test_live_contract.py::test_operation_table_is_read_only`, `studio/tests/protocol/test_live_client.py::test_write_operations_do_not_exist` |
| LIVE-02 | Only report blocks shown as free are used; a busy block is never taken. | `studio/tests/unit/test_access_policy.py::test_busy_blocks_fall_back_to_polling`, `studio/tests/protocol/test_live_client.py::test_enabled_block_is_not_taken` |
| LIVE-03 | Every block Studio reserved is released on disconnect. | `studio/tests/protocol/test_live_client.py::test_blocks_released_on_disconnect` |
| LIVE-04 | Operating equipment bound to a real IED is refused with a plain reason. | `studio/tests/unit/test_live_roles.py::test_operating_real_ied_is_refused` |
| LIVE-05 | A real IED is never started as a simulated worker. | `studio/tests/unit/test_live_roles.py::test_real_ied_is_not_simulated` |
| LIVE-06 | The comparison classifies every reference as match, missing on device, extra on device or type mismatch, and diffs datasets and report blocks. | `studio/tests/unit/test_model_compare.py::test_signal_categories`, `studio/tests/unit/test_model_compare.py::test_dataset_and_report_differences` |
| LIVE-07 | Live values keep device quality and timestamp and carry a live source; unreadable values are reported, not invented. | `studio/tests/unit/test_live_values.py::test_device_quality_and_source_are_kept`, `studio/tests/unit/test_live_values.py::test_read_errors_are_not_values` |
| LIVE-08 | The real endpoint is prefilled from the SCD and a user override is stored separately. | `studio/tests/unit/test_live_roles.py::test_endpoint_prefill_and_override` |
| LIVE-09 | Connection loss is an explicit state and values are marked stale, never frozen as fresh. | `studio/tests/unit/test_live_values.py::test_connection_loss_marks_values_stale` |
| LIVE-10 | Discovery returns the device model over MMS so it can be compared. | `studio/tests/protocol/test_live_client.py::test_discover_and_read_simulated_stand_in` |

Protocol evidence uses a separate `studio_ied.exe` as a stand-in device on this PC.
Interoperability with third-party IEDs remains an open release gate.
