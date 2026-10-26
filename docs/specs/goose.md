# Spec: GOOSE

Decision: [ADR-0030](../decisions/ADR-0030-goose-with-user-installed-npcap.md); subscriber inputs:
[ADR-0066](../decisions/ADR-0066-goose-inputs-and-mms-access-events.md); GOOSE engine and
"Where GOOSE goes": [ADR-0073](../decisions/ADR-0073-goose-engine-go.md), amended by
[ADR-0119](../decisions/ADR-0119-goose-follows-the-ieds.md) (Go engine); SLD view:
[signal layers](signal-layers.md).
Owner policy (2026-10-13): Npcap is installed by the user with its free licence;
Studio detects and guides, and never bundles it. Since ADR-0073, GOOSE between simulated
IEDs needs no Npcap; only sharing with the network and hearing real devices do.

## Purpose

Simulated IEDs publish the GOOSE control blocks defined in their SCD through the Go GOOSE engine
`studio_goose.exe`. Go engine (ADR-0119): by default GOOSE follows the IEDs: in lab mode it also
goes to the lab adapter (or the adapter chosen under Advanced) without a click, guarded as below;
with IEDs on this PC only it stays inside. *Inside ScadaBench only* keeps it inside. A real device
on the adapter makes it ask once. Python engine (ADR-0073): inside by default, and one confirmed
switch, *Also share with the network*, sends it on an automatically picked wired adapter. Studio shows, per control block,
whether GOOSE is really being received, with a plain explanation of any problem.

## Behaviour contract

- libiec61850 owns GoCB state, retransmission and frame encoding. Studio selects the
  adapter, reports what the stack did, and never fabricates frames or values.
- A GOOSE send from the SLD (ADR-0112: IED component, equipment `goose_link`, point panel row)
  changes the simulated IED's values, never the frame: all values of one send are written in one
  sample batch under one data model lock, so libiec61850 raises stNum exactly once. Real devices
  are never sent for.
- Publishing parameters come only from the SCD: `GSEControl` (name, dataset,
  `appID` as goID, `confRev`) and the matching `Communication/.../GSE` address
  (MAC, APPID, VLAN-ID, VLAN-PRIORITY, MinTime, MaxTime).
- Workers never get an adapter. They publish to the engine (`udp:PORT:MAC`), which gives each
  simulated IED its own locally administered MAC for the run (`02:<random>:<index>`).
- The engine forwards every valid frame to `studio_live`'s inbox with its origin (`app` or
  `wire`). `studio_live` decodes it with libiec61850 and reports `origin` and `src_mac`.
- Go engine (ADR-0119): the choice is `auto` (default) or `local`; on disk `goose_reach` keeps
  Python's `inside`/`outside` and `local` is the Go-only `goose_local: true`, stored only while chosen. GOOSE goes out only with no hold: Npcap missing gives one red line with a download
  link; real IEDs in the project, or GOOSE from a vendor (globally administered) MAC heard on the
  adapter, hold it until the user confirms the warning for that adapter (`goose_share_guid`). The
  engine enforces that hold itself (`confirm_real`, share state `real_hold`); ScadaBench's own
  `02:` MACs never count. After 10 s of sending with nothing heard back while a GOOSE is still
  waiting, the card says the other PC or VM may be on another network.
- **Inside** (default, Python engine): the engine may listen on the picked adapter (needs Npcap) so real
  devices are heard, but it cannot transmit. **Share**: the user confirms a warning that names the
  adapter (stronger when real devices are likely to act on it). The engine listens 3 s first, then
  sends only registered simulated frames (multicast, ≤1518 bytes, ≤1000 per second per IED).
- Smart guard, always on:
  - Frames that come back with our own MAC are dropped.
  - Network frames are never sent back to the network.
  - If a real device sends a simulated IED's gocbRef, that IED's outward copy is held and the
    user is asked *Keep it stopped outside* or *Continue anyway* (Continue needs confirmation).
    The real copy is never delivered inside. The answer lasts for the current Run.
- The adapter is picked automatically: the lab adapter in lab mode, otherwise a connected wired
  adapter, preferring the default route. Wi-Fi, VPN, virtual and loopback adapters are never
  auto-picked. Advanced can override the choice. If the adapter in use differs from the one the
  user confirmed, sharing pauses; it never moves to the new adapter.
- Every native executable delay-loads Npcap and starts without it.
- Supervision states are `waiting`, `receiving`, `problem` and `timeout`. Problems are
  `invalid_frame`, `conf_rev_mismatch`, `dataset_mismatch`, `go_id_mismatch`
  ([ADR-0037](../decisions/ADR-0037-station-health.md)), `needs_commission` and
  `timeout`. GOOSE seen on the wire but not in the project is listed as *unknown*.
- A simulated subscriber applies received values to the attributes its own `ExtRef` binds
  (`intAddr` naming exactly one runtime-editable attribute), through the ordinary sample path,
  only while the frame is `receiving` without a problem. Anything else is shown, not applied.
- A GSEControl that the vendor marks `Function="Disabled"` (GE Multilin) is not published.

## Out of scope

R-GOOSE, GOOSE simulation/test-mode injection (Sampled Values listening is its own
[specification](sampled-values.md)), applying GOOSE from blocks not declared in the project,
and guessing what a vendor-specific `intAddr` means.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| GOOSE-01 | Every native build compiles GOOSE; configuration fails when the pinned Npcap SDK is missing instead of silently omitting it. | `studio/tests/unit/test_native_build_config.py::test_goose_requires_pinned_sdk` |
| GOOSE-02 | Native executables delay-load Npcap so they start without it. | `studio/tests/unit/test_native_build_config.py::test_npcap_is_delay_loaded`, `studio/tests/protocol/test_goose_runtime.py::test_worker_reports_goose_readiness` |
| GOOSE-03 | A `GSEControl` without a matching `GSE` address is never published and produces an import warning. | `studio/tests/unit/test_goose_scl.py::test_control_without_address_is_not_published` |
| GOOSE-04 | The native GoCB configuration uses the SCD values exactly. | `studio/tests/unit/test_goose_scl.py::test_gc_line_uses_scd_values` |
| GOOSE-05 | Workers never get an adapter; they publish only to the GOOSE engine (`udp:`), in every mode. | `studio/tests/unit/test_goose_settings.py::test_workers_never_get_an_adapter`, `studio/tests/protocol/test_goose_inside.py::test_worker_refuses_any_interface_but_the_engine` |
| GOOSE-06 | A worker without the `goose_aware` capability is rejected as a stale engine. | `studio/tests/unit/test_goose_settings.py::test_stale_worker_is_rejected` |
| GOOSE-07 | Subscriptions come from `ExtRef`; an unresolved source block is a diagnostic, never a guess. | `studio/tests/unit/test_goose_scl.py::test_unknown_source_block_is_diagnosed`, `studio/tests/unit/test_goose_scl.py::test_extref_resolves_publisher_block` |
| GOOSE-08 | Received GOOSE is written into a simulated IED only as LGOS supervision attributes and as the inputs its own ExtRefs bind (ADR-0066). | `studio/tests/unit/test_goose_routing.py::test_only_lgos_attributes_are_written`, `studio/tests/unit/test_goose_inputs.py::test_relative_intaddr_binds_to_the_subscribers_own_point` |
| GOOSE-09 | Every supervision state and problem has a plain label, an icon and a fix hint. | `studio/tests/frontend/gooseStatus.test.ts::every goose state has text icon and hint` |
| GOOSE-10 | Vendor console output never corrupts the private worker channel. | `studio/tests/protocol/test_goose_runtime.py::test_frames_survive_vendor_console_output` |
| GOOSE-11 | A simulated publisher reaches the separate decoder process through the engine, without an adapter. | `studio/tests/protocol/test_goose_inside.py::test_simulated_publisher_reaches_the_decoder_through_the_engine` |
| GOOSE-12 | Each input's position in the publisher's dataset is read from SCL at import. | `studio/tests/unit/test_goose_inputs.py::test_input_position_in_dataset_is_read_from_scl` |
| GOOSE-13 | A vendor-specific or ambiguous intAddr is shown but never applied. | `studio/tests/unit/test_goose_inputs.py::test_vendor_intaddr_is_shown_but_not_applied` |
| GOOSE-14 | A frame with a problem or a dataset size different from the SCD never drives inputs. | `studio/tests/unit/test_goose_inputs.py::test_frames_with_problems_never_drive_inputs`, `studio/tests/protocol/test_goose_inputs_wire.py::test_conf_rev_mismatch_never_changes_the_subscriber` |
| GOOSE-15 | Values follow the target's SCD type; a structured member needs the publisher's layout. | `studio/tests/unit/test_goose_inputs.py::test_value_types_are_converted_or_reported`, `studio/tests/unit/test_goose_inputs.py::test_structured_member_needs_the_publisher_layout` |
| GOOSE-16 | Over the wire, a simulated subscriber's input follows the publisher, checked by an independent MMS client. | `studio/tests/protocol/test_goose_inputs_wire.py::test_subscriber_input_follows_the_publisher` |

| GOOSE-17 | Inside works without Npcap or an adapter and sends nothing out. | `studio/tests/protocol/test_goose_inside.py::test_simulated_publisher_reaches_the_decoder_through_the_engine`, `studio/goose_engine/internal/engine/engine_test.go::TestSimulatedGooseReachesTheInboxWithoutAnyAdapter` |
| GOOSE-18 | Inside never transmits on the adapter, while real GOOSE on it is still heard. | `studio/tests/protocol/test_goose_share_wire.py::test_inside_listens_to_real_devices_but_never_sends`, `studio/goose_engine/internal/engine/engine_test.go::TestInsideModeListensButNeverTransmits` |
| GOOSE-19 | Sharing is refused without confirm, without Npcap, or when the adapter differs from the one shown; Continue after a conflict needs confirm. | `studio/tests/unit/test_goose_share.py::test_sharing_is_refused_without_confirm_npcap_or_the_seen_adapter`, `studio/tests/unit/test_goose_share.py::test_continue_after_a_conflict_needs_confirm` |
| GOOSE-20 | Shared frames reach an independent peer only after the 3 s check; each IED has its own locally administered MAC. | `studio/tests/protocol/test_goose_share_wire.py::test_share_waits_for_the_check_then_sends_each_ied_with_its_own_mac` |
| GOOSE-21 | A conflict holds the IED's outward copy until Stop or Continue; the real copy is never delivered inside; inside traffic continues. | `studio/goose_engine/internal/guard/guard_test.go::TestRealDeviceWithOurGocbRefHoldsTheGateOnceAndIsNeverDelivered`, `studio/tests/protocol/test_goose_conflict_wire.py::test_conflict_holds_the_ied_until_stop_or_continue`, `studio/tests/protocol/test_goose_conflict_wire.py::test_inside_traffic_continues_while_held` |
| GOOSE-22 | Reflections of our own frames are dropped and network frames are never sent back out. | `studio/goose_engine/internal/guard/guard_test.go::TestOurReflectionIsDroppedAndNeverAConflict`, `studio/tests/protocol/test_goose_conflict_wire.py::test_reflections_are_dropped_and_wire_frames_are_never_resent` |
| GOOSE-23 | Changing the network mode keeps where GOOSE goes; old lab projects get a one-time notice. | `studio/tests/unit/test_goose_reach.py::test_mode_change_keeps_reach`, `studio/tests/unit/test_goose_reach.py::test_lab_projects_saved_before_reach_get_a_one_time_notice` |
| GOOSE-24 | The adapter is auto-picked in a fixed order and is never Wi-Fi, VPN, virtual, loopback or unplugged. | `studio/tests/unit/test_goose_reach.py::test_pick_uplink_prefers_advanced_then_lab_then_connected_cable_with_default_route`, `studio/tests/unit/test_goose_reach.py::test_pick_uplink_never_auto_picks_wifi_vpn_virtual_loopback_or_unplugged` |
| GOOSE-25 | The share warning is stronger when real devices may act on the GOOSE; Cancel has the focus. | `studio/tests/frontend/gooseReach.test.ts::GOOSE-25: the share warning is stronger when real devices may act on it`, `studio/tests/unit/test_goose_reach.py::test_share_risk_is_high_when_real_devices_may_act_on_it` |
| GOOSE-26 | `udp:` interface ids go to the engine; numeric ids keep the vendor Npcap HAL (Sampled Values unchanged). | `studio/tests/unit/test_native_build_config.py::test_ethernet_hal_routes_udp_to_the_engine_and_numbers_to_npcap` |
| GOOSE-27 | The backend refuses a missing or stale engine and opens it before any worker starts. | `studio/tests/unit/test_goose_engine_process.py::test_missing_or_stale_engine_is_refused`, `studio/tests/unit/test_goose_engine_process.py::test_supervisor_asks_the_engine_before_any_worker_starts` |
| GOOSE-28 | One SLD GOOSE send is one batch: two values change together and the receiver sees stNum rise exactly once. | `studio/tests/protocol/test_goose_send_batch.py::test_two_values_in_one_batch_raise_st_num_once`, `studio/engine/internal/runtime/goose_send_test.go::TestGooseSendWritesOneBatchPerAction` |
| GOOSE-29 | LGOS supervision binds whether `setSrcRef` uses the MMS form (`LD/LLN0$GO$cb`) or the ObjectReference form (`LD/LLN0.cb`) (Go engine). | `studio/engine/internal/scl/vendor_station_test.go::TestLGOSAcceptsTheObjectReferenceForm` |
| GOOSE-30 | A link from a publisher in the file but not in the project counts as outside the project, never as "not heard" (Go engine). | `studio/engine/internal/acquisition/goosedom/link_outside_test.go::TestUnheardPublisherOutsideTheProjectIsRemote` |
| GOOSE-31 | A `GSE` without APPID gives one "has no APPID" warning, not a second "no network address" one (Go engine). | `studio/engine/internal/scl/vendor_blocks_test.go::TestGSEWithoutAPPIDIsOneWarning` |
| GOOSE-32 | GOOSE follows the IEDs: `auto` by default, lab or chosen adapter sends, this-PC-only IEDs stay inside; the stored file stays readable by the Python engine (Go engine, ADR-0119). | `studio/engine/internal/project/goose_reach_test.go::TestGooseReachChoicesKeepTheFilePythonReadable`, `studio/engine/internal/acquisition/goosedom/reach_auto_test.go::TestGooseGoesOutWithTheIEDsNetwork` |
| GOOSE-33 | A real device (real IED in the project, or a vendor MAC heard) holds sending until confirmed once for that adapter; another ScadaBench never does (Go engine, ADR-0119). | `studio/goose_engine/internal/engine/real_test.go::TestARealDeviceHoldsUnconfirmedSharingUntilConfirmed`, `studio/goose_engine/internal/engine/real_test.go::TestARealDeviceStopsSharingThatAlreadyRuns`, `studio/goose_engine/internal/engine/real_test.go::TestAnotherScadaBenchIsNotARealDevice`, `studio/engine/internal/acquisition/goosedom/reach_auto_test.go::TestRealDevicesAskOncePerAdapter` |
| GOOSE-34 | Without Npcap, GOOSE that should go out shows one red line and a download link (Go engine, ADR-0119). | `studio/engine/internal/acquisition/goosedom/reach_auto_test.go::TestNpcapMissingIsOneClearLine`, `studio/tests/frontend/gooseReach.test.ts::ADR-0119: real devices are confirmed with the adapter shown; a conflict needs two clicks to continue` |

The GOOSE-18, GOOSE-20, GOOSE-21 and GOOSE-22 wire tests need Npcap and a lab adapter. Set
`STUDIO_GOOSE_ADAPTER` to its GUID; otherwise they are skipped, and the change record must say so.
The independent peer is `studio/tests/protocol/npcap_peer.py`, which shares no code with the engine.
