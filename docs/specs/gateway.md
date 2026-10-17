# Spec: Gateway simulator (IEC 60870-5-104, Modbus TCP, DNP3)

Decision: [ADR-0057](../decisions/ADR-0057-gateway-explorer-commands.md).
Change: [0082](../changes/2026/0082-gateway-simulator.md). Models: the FreyrSCADA IEC 104 and
DNP3 simulators (point table, traffic log) and station gateways.

## Purpose

ScadaBench acts as the station gateway, so a SCADA master can be tested against the project's
IEDs over IEC 104, Modbus TCP or DNP3.

## Behaviour contract

- **Workers.**
  - `studio_gateway` serves one protocol at a time on the chosen port: "This PC only" binds
    127.0.0.1, "All network adapters" binds 0.0.0.0.
  - It has no IEC 61850 client. Every SCADA command is answered after the backend has run
    it through `studio_control`.
- **Values.** Points follow the coordinator's signal index every 0.1 s.
  - A point is invalid while Run is off, while its IED is off, or while a real IED's value is
    not fresh. A value is never made up.
  - IEC 61850 quality maps to 104 IV/NT/SB/BL/OV and to DNP3 flags.
- **Autonomous start.** Start proposes the point list when it is empty and starts Run when it
  is stopped.
  - A successful start is remembered (`auto_start`), and a user Stop clears it.
  - After the project is restored, a remembered gateway starts again; a failure shows in its
    status.
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
| GW-06 | A 104 master operates a simulated breaker end to end, after an autonomous start. | `studio/tests/protocol/test_gateway_runtime.py::test_iec104_master_operates_a_simulated_breaker`, `studio/tests/unit/test_tool_autonomy.py::test_gateway_prepares_points_and_run_by_itself` |
| GW-07 | Real-IED commands follow the owner switch, in the backend and in the native monitor-only flag. | `studio/tests/protocol/test_control_client.py::test_gateway_commands_a_real_role_ied_and_respects_the_switch`, `studio/tests/unit/test_tool_autonomy.py::test_monitor_only_follows_role_and_real_command_setting`, `studio/tests/unit/test_gateway_safety.py::test_backend_refuses_unsafe_or_unknown_targets` |
| GW-08 | The gateway worker has no IEC 61850 client and refuses to listen without a license. | `studio/tests/unit/test_gateway_safety.py::test_native_gateway_has_no_iec61850_client`, `studio/tests/unit/test_native_enforcement.py::test_gateway_refuses_to_listen_without_license` |
| GW-09 | A remembered gateway starts again by itself and never raises. | `studio/tests/unit/test_tool_autonomy.py::test_autostart_only_when_remembered_and_never_raises` |
