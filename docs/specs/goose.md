# Spec: GOOSE

Decision: [ADR-0030](../decisions/ADR-0030-goose-with-user-installed-npcap.md).
Owner policy (2026-10-13): Npcap is installed by the user with its free licence;
Studio detects and guides, and never bundles it.

## Purpose

Simulated IEDs publish the GOOSE control blocks defined in their SCD on the lab
adapter. Studio also listens on that adapter and shows, per control block, whether
GOOSE is really being received, with a plain explanation of any problem.

## Behaviour contract

- libiec61850 owns GoCB state, retransmission and frame encoding. Studio selects the
  adapter, reports what the stack did, and never fabricates frames or values.
- Publishing parameters come only from the SCD: `GSEControl` (name, dataset,
  `appID` as goID, `confRev`) and the matching `Communication/.../GSE` address
  (MAC, APPID, VLAN-ID, VLAN-PRIORITY, MinTime, MaxTime).
- GOOSE is active only in lab mode with an adapter selected and Npcap present.
  Loopback mode cannot carry Ethernet frames, so it never publishes.
- Every native executable delay-loads Npcap and starts without it.
- `studio_live` listens with one receiver per adapter for (a) GOOSE subscribed by
  simulated IEDs through `ExtRef`, (b) GOOSE published by simulated IEDs (wire
  evidence) and (c) GOOSE published by IEDs set to *real*.
- Supervision states are `waiting`, `receiving`, `problem` and `timeout`. Problems are
  `invalid_frame`, `conf_rev_mismatch`, `dataset_mismatch`, `go_id_mismatch`
  ([ADR-0037](../decisions/ADR-0037-station-health.md)), `needs_commission` and
  `timeout`. GOOSE seen on the wire but not in the project is listed as *unknown*.

## Out of scope

R-GOOSE, GOOSE simulation/test-mode injection (Sampled Values listening is its own
[specification](sampled-values.md)), and writing received
GOOSE values into a simulated subscriber's own data model (only LGOS supervision).

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| GOOSE-01 | Every native build compiles GOOSE; configuration fails when the pinned Npcap SDK is missing instead of silently omitting it. | `studio/tests/unit/test_native_build_config.py::test_goose_requires_pinned_sdk` |
| GOOSE-02 | Native executables delay-load Npcap so they start without it. | `studio/tests/unit/test_native_build_config.py::test_npcap_is_delay_loaded`, `studio/tests/protocol/test_goose_runtime.py::test_worker_reports_goose_readiness` |
| GOOSE-03 | A `GSEControl` without a matching `GSE` address is never published and produces an import warning. | `studio/tests/unit/test_goose_scl.py::test_control_without_address_is_not_published` |
| GOOSE-04 | The native GoCB configuration uses the SCD values exactly. | `studio/tests/unit/test_goose_scl.py::test_gc_line_uses_scd_values` |
| GOOSE-05 | Loopback mode never passes a GOOSE adapter to a worker. | `studio/tests/unit/test_goose_settings.py::test_loopback_mode_never_publishes` |
| GOOSE-06 | A worker without the `goose_aware` capability is rejected as a stale engine. | `studio/tests/unit/test_goose_settings.py::test_stale_worker_is_rejected` |
| GOOSE-07 | Subscriptions come from `ExtRef`; an unresolved source block is a diagnostic, never a guess. | `studio/tests/unit/test_goose_scl.py::test_unknown_source_block_is_diagnosed`, `studio/tests/unit/test_goose_scl.py::test_extref_resolves_publisher_block` |
| GOOSE-08 | Received GOOSE is written into a simulated IED only as LGOS supervision attributes. | `studio/tests/unit/test_goose_routing.py::test_only_lgos_attributes_are_written` |
| GOOSE-09 | Every supervision state and problem has a plain label, an icon and a fix hint. | `studio/tests/frontend/gooseStatus.test.ts::every goose state has text icon and hint` |
| GOOSE-10 | Vendor console output never corrupts the private worker channel. | `studio/tests/protocol/test_goose_runtime.py::test_frames_survive_vendor_console_output` |
| GOOSE-11 | Publisher frames are received by an independent subscriber process on the same adapter. | `studio/tests/protocol/test_goose_wire.py::test_simulated_publisher_reaches_listener` |

GOOSE-11 needs a lab adapter: set `STUDIO_GOOSE_ADAPTER` to its GUID, otherwise the
test is skipped and the change record must say so.
