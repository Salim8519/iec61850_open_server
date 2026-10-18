# Spec: Gateway simulator (IEC 60870-5-104, Modbus TCP, DNP3)

Decisions: [ADR-0057](../decisions/ADR-0057-gateway-explorer-commands.md),
[ADR-0059](../decisions/ADR-0059-separate-run-running-dots.md) (separate Run, page commands),
[ADR-0061](../decisions/ADR-0061-scada-roles-master-converter.md) (roles: this spec is the Station gateway role;
the SCADA master and protocol converter are in [scada-master.md](scada-master.md)).
Change: [0082](../changes/2026/0082-gateway-simulator.md). Models: the FreyrSCADA IEC 104 and
DNP3 simulators (point table, traffic log) and station gateways.

## Purpose

ScadaBench acts as the station gateway, so a SCADA (104 controlling station, DNP3 master or Modbus
client) can be tested against the project's IEDs over IEC 104, Modbus TCP or DNP3. Toward SCADA
ScadaBench is the controlled station (104 slave), DNP3 outstation or Modbus server.

## Behaviour contract

- **Workers.**
  - `studio_gateway` serves one protocol at a time on the chosen port: "This PC only" binds
    127.0.0.1, "All network adapters" binds 0.0.0.0.
  - On "All network adapters" the tool checks Windows Firewall and offers, after review and one
    UAC approval, an allow rule for the gateway worker and switching off Windows' leftover
    Block rules for it (ADR-0062). Clean-up undoes both.
  - It has no IEC 61850 client. Every SCADA command is answered after the backend has run
    it through `studio_control`.
- **Values.** Points follow the coordinator's signal index every 0.1 s.
  - A point is invalid while Run is off, while its IED is off, or while a real IED's value is
    not fresh. A value is never made up.
  - IEC 61850 quality maps to 104 IV/NT/SB/BL/OV and to DNP3 flags.
- **Start and Run are separate (ADR-0059).** Start proposes the point list when it is empty and
  listens; it never starts the IEDs. While the IEDs are off, the page says so and offers a
  separate "Run the IEDs" button.
  - A successful start is remembered (`auto_start`), and a user Stop clears it.
  - After the project is restored, a remembered gateway starts listening again (never the IEDs);
    a failure shows in its status.
  - Running or stopped, the settings stay in the left column with the status, inline errors and
    Start/Stop at its bottom, and the Points, Traffic and Commands panel fills the rest of the
    window with its own scroll (ADR-0064, [tool layout](tool-layout.md); the running-only status
    bar of ADR-0059 is replaced). Each tab has the shared filter bar.
- **Smart point list.** The proposal keeps:
  - positions (double);
  - switchgear and supervision statuses and protection trips (single);
  - measurements and harmonic totals (float or integer);
  - logical-device health (integer);
  - every controllable object (command).

  Generic I/O (GGIO) values are kept only when they are in a dataset or described in the SCD.
  When the 5,000-point limit is reached, generic values are dropped first and a note says so.
  Every address can be edited. The list imports and exports as CSV
  (`type,address,ied,reference,scale,description`), with line diagnostics.
- **Commands.**
  - IEC 104 single and double commands, with select/execute.
  - Modbus coils (FC5/FC15).
  - DNP3 CROB.

  The page offers the same commands in command rows (Close/Open or On/Off). Each one is sent
  only after a warning that names the IED and object (stronger for a real device), with
  `confirm: true`, and is logged as "You · this page".
  Commands operate simulated IEDs. Real IEDs are operated while "Allow SCADA commands to real
  IEDs" is on (the default); otherwise the worker answers negatively and the backend refuses.
  The command log keeps request, applied and observed results separate.
- **Evidence before claims.** Each protocol is checked against an independent master:
  - the lib60870 client (`studio_test_iec104_master`);
  - the opendnp3 master (`studio_test_dnp3_master`);
  - a raw-socket Modbus TCP client.

  Third-party SCADA masters are not yet recorded.

| ID | Rule | Enforced by |
| --- | --- | --- |
| GW-01 | The proposal maps positions, statuses, measurements and commands to free addresses; Modbus floats take two registers. | `studio/tests/unit/test_gateway_domain.py::test_proposal_maps_positions_statuses_measurements_and_commands`, `studio/tests/unit/test_gateway_domain.py::test_modbus_floats_take_two_registers_without_overlap` |
| GW-02 | The smart list keeps only what SCADA needs, leaves out spare generic inputs and ranks them last. | `studio/tests/unit/test_gateway_domain.py::test_smart_selection_keeps_only_what_scada_needs`, `studio/tests/unit/test_gateway_domain.py::test_spare_generic_inputs_stay_out_and_rank_last` |
| GW-03 | Every invalid point, setting or CSV line is explained; stored lists keep points of removed IEDs. | `studio/tests/unit/test_gateway_domain.py::test_validation_explains_every_problem`, `studio/tests/unit/test_gateway_domain.py::test_settings_follow_the_protocol_rules`, `studio/tests/unit/test_gateway_domain.py::test_csv_round_trip_and_line_diagnostics`, `studio/tests/unit/test_gateway_domain.py::test_stored_gateway_keeps_points_of_removed_ieds` |
| GW-04 | Stopped, off or stale sources are sent invalid, never as made-up values. | `studio/tests/unit/test_gateway_domain.py::test_sampling_marks_stopped_or_stale_sources_invalid` |
| GW-05 | Independent masters read, receive spontaneous changes and command over all three protocols. | `studio/tests/protocol/test_gateway_wire.py::test_iec104_interrogation_commands_and_refusals`, `studio/tests/protocol/test_gateway_wire.py::test_iec104_spontaneous_change_and_wrong_common_address`, `studio/tests/protocol/test_gateway_wire.py::test_modbus_tcp_reads_and_coil_commands`, `studio/tests/protocol/test_gateway_wire.py::test_dnp3_integrity_scan_and_crob`, `studio/tests/protocol/test_gateway_wire.py::test_listen_failure_is_explained` |
| GW-06 | A 104 master operates a simulated breaker end to end; Start prepares the points and leaves Run alone. | `studio/tests/protocol/test_gateway_runtime.py::test_iec104_master_operates_a_simulated_breaker`, `studio/tests/unit/test_tool_autonomy.py::test_gateway_prepares_points_but_leaves_run_alone`, `studio/tests/frontend/running.test.ts::the gateway never starts Run by itself; Run is offered as a button` |
| GW-07 | Real-IED commands follow the owner switch, in the backend and in the native monitor-only flag. | `studio/tests/protocol/test_control_client.py::test_gateway_commands_a_real_role_ied_and_respects_the_switch`, `studio/tests/unit/test_tool_autonomy.py::test_monitor_only_follows_role_and_real_command_setting`, `studio/tests/unit/test_gateway_safety.py::test_backend_refuses_unsafe_or_unknown_targets` |
| GW-08 | The gateway worker has no IEC 61850 client and refuses to listen without a license. | `studio/tests/unit/test_gateway_safety.py::test_native_gateway_has_no_iec61850_client`, `studio/tests/unit/test_native_enforcement.py::test_gateway_refuses_to_listen_without_license` |
| GW-09 | A remembered gateway starts again by itself and never raises. | `studio/tests/unit/test_tool_autonomy.py::test_autostart_only_when_remembered_and_never_raises` |
| GW-10 | Commands from the page are sent only after the warning, with confirmation, and run and log like SCADA commands. | `studio/tests/frontend/running.test.ts::gateway commands are sent only from the warning, with the confirmation flag`, `studio/tests/unit/test_tool_autonomy.py::test_saved_points_are_found_while_stopped_and_manual_commands_explain_refusals`, `studio/tests/protocol/test_gateway_runtime.py::test_iec104_master_operates_a_simulated_breaker` |
| GW-11 | "All network adapters" is reachable from other computers only through a reviewed, journaled firewall change: an allow rule for the gateway worker only, Block rules switched off (never deleted) and re-enabled by clean-up. | `studio/tests/unit/test_gateway_firewall.py::test_plan_allows_the_gateway_port_and_switches_off_block_rules`, `studio/tests/unit/test_gateway_firewall.py::test_helper_accepts_only_the_gateway_worker_and_plain_rule_names`, `studio/tests/unit/test_gateway_firewall.py::test_cleanup_finds_the_gateway_changes_in_the_journal`, `studio/tests/unit/test_gateway_firewall.py::test_preview_then_apply_sends_only_what_was_reviewed` |
