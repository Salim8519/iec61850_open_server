# Spec: SCADA master and protocol converter (IEC 104, DNP3, Modbus TCP)

Decision: [ADR-0061](../decisions/ADR-0061-scada-roles-master-converter.md). Change:
[0088](../changes/2026/0088-scada-master-converter.md). Related: [gateway.md](gateway.md)
(the station gateway role).

## Purpose

ScadaBench plays the asking side of SCADA: the IEC 104 controlling station, the DNP3 master or
the Modbus TCP client. It connects to a real or simulated gateway, RTU, PLC or outstation, finds
every point by itself, and sends confirmed commands. As a protocol converter it serves what it
reads to another SCADA.

## Behaviour contract

- **Roles.** "Act as" chooses Station gateway, SCADA master or Protocol converter. Only one
  role runs at a time, and switching is refused while anything runs.
- **Words.** Each protocol uses its own words: controlling/controlled station (104 master/slave),
  DNP3 master/outstation, Modbus client/server.
- **Discovery.**
  - IEC 104: station interrogation at connect and every `gi_minutes`, followed by a general
    counter interrogation (C_CI_NA_1, read without freeze) so integrated totals are found too; a
    device that refuses it does not fail discovery.
  - DNP3: integrity poll at connect and every `integrity_minutes`, plus event polls and
    unsolicited events.
  - Modbus: the listed ranges every `poll_ms`.
  - Every point is watched by default; unticking only hides it from the watch list, the CSV and
    the converter.
- **Quality.** Raw quality bits are kept and worded:
  - 104: IV, NT, SB, BL, OV;
  - DNP3: ONLINE, RESTART, COMM_LOST, forced, over range;
  - Modbus: "no answer" for a refused range.

  A value is shown as last known while the link is down; its quality then reads
  "not current (link lost)" and is never shown as good. A repeated interrogation of the same
  value is not counted as a change.
- **Commands.** Commands are sent only after the REAL-device warning, with `confirm: true`.
  - 104: C_SC, C_DC, C_SE (float, scaled, normalised), C_CS; select/execute; ACT_CON and
    ACT_TERM.
  - DNP3: CROB and analog outputs; select-and-operate or direct.
  - Modbus: FC5, FC6 and FC16.

  Refusals are explained in words: negative confirmation, the DNP3 status, or the Modbus
  exception.
- **Find registers.**
  - At most 10,000 addresses, at most 10 requests per second, read only.
  - A preview shows how many requests it sends.
  - Found spans become ranges of the chosen data type, split to the per-request limit.
- **Converter.**
  - Watched device points are served upstream at the same address, or at the next free one
    with a note.
  - Values are invalid upstream while the device link is down or the value is bad.
  - DNP3 binary outputs and Modbus coils are relayed automatically; other commands need a
    command route.
  - The real-device switch applies.

| ID | Rule | Enforced by |
| --- | --- | --- |
| MASTER-01 | The 104 controlling station discovers every point by interrogation and receives spontaneous changes. | `studio/tests/protocol/test_master_wire.py::test_master_discovers_every_point_by_general_interrogation` |
| MASTER-02 | 104 commands report select, execute and termination; refusals and invalid values are explained. | `studio/tests/protocol/test_master_wire.py::test_master_commands_get_confirmation_termination_and_refusals`, `studio/tests/protocol/test_master_wire.py::test_master_explains_an_unreachable_device_and_disconnects_cleanly` |
| MASTER-03 | The DNP3 master discovers by integrity poll, receives events, operates a CROB and explains a refusal or an absent outstation. | `studio/tests/protocol/test_master_wire.py::test_dnp3_master_integrity_poll_events_and_crob`, `studio/tests/protocol/test_master_wire.py::test_dnp3_master_explains_an_unreachable_outstation` |
| MASTER-04 | The Modbus client reads typed ranges and writes a coil on our server, and finds registers, reads and writes on an independent server. | `studio/tests/protocol/test_master_modbus_wire.py::test_modbus_client_reads_ranges_from_our_gateway_and_writes_a_coil`, `studio/tests/protocol/test_master_modbus_wire.py::test_modbus_client_finds_registers_on_an_independent_server_and_explains_refusals` |
| MASTER-05 | Through the API the master discovers, names and unticks points, exports CSV, sends only confirmed commands and remembers the connection. | `studio/tests/protocol/test_master_runtime.py::test_master_role_discovers_names_and_commands_a_device` |
| MASTER-06 | Settings, ranges and the format 13 → 14 upgrade are validated; quality is worded and only real changes count. | `studio/tests/unit/test_master_rules.py::test_master_settings_use_defaults_and_explain_every_problem`, `studio/tests/unit/test_master_rules.py::test_modbus_ranges_are_bounded_per_request`, `studio/tests/unit/test_master_rules.py::test_format_13_gateway_sections_upgrade_to_station_gateway_role`, `studio/tests/unit/test_master_rules.py::test_discovered_points_merge_word_quality_and_keep_labels` |
| MASTER-07 | Find registers is bounded, paced and read-only, and found spans become valid ranges. | `studio/tests/unit/test_master_rules.py::test_find_registers_is_bounded_and_found_spans_become_read_ranges`, `studio/tests/protocol/test_master_modbus_runtime.py::test_modbus_client_finds_reads_and_writes_through_the_api` |
| MASTER-08 | Each protocol view uses its own official words; master commands leave the page only from the REAL-device confirmation. | `studio/tests/frontend/scadaTerms.test.ts::each protocol uses its own official words for both sides (ADR-0061)`, `studio/tests/frontend/scadaTerms.test.ts::master commands leave the page only from the REAL-device confirmation`, `studio/tests/frontend/scadaTerms.test.ts::the tool card says what the master is doing` |
| MASTER-09 | The converter keeps device addresses where free, moves clashes with a note, and serves only watched points. | `studio/tests/unit/test_passthrough.py::test_watched_points_keep_their_address_or_move_to_the_next_free_one` |
| MASTER-10 | Command routes need a free upstream address; values are invalid upstream while the device is bad or away. | `studio/tests/unit/test_passthrough.py::test_command_routes_need_a_free_upstream_address`, `studio/tests/unit/test_passthrough.py::test_remote_values_are_invalid_upstream_while_the_device_is_bad_or_away` |
| MASTER-11 | An independent 104 master reads a DNP3 outstation through the converter and its command reaches the outstation. | `studio/tests/protocol/test_converter_runtime.py::test_dnp3_device_is_served_as_iec104_and_scada_commands_reach_it` |
| MASTER-12 | The master worker refuses to connect without a license and is packaged and sealed. | `studio/tests/unit/test_native_enforcement.py::test_master_refuses_to_connect_without_license`, `studio/tests/unit/test_protected_packaging.py::test_protected_build_refuses_without_release_key_and_dry_run_lists_steps` |
