# 1b. Class (c) Python-internal unit tests and their Go ports

78 files / 350 tests (75 `c` + 3 `c-launcher`) plus `protocol/test_adapter_listing`
(Windows IP Helper, listed under networking). Area = the backend package the file
mainly imports (`tools`, `domain` overrides applied by hand where the main import misled).

## Porting rules (identical fixtures, identical expectations)

1. **One Go test file per Python test file**, same base name, in the Go package that owns
   the ported module: `unit/test_goose_reach.py` -> `studio/core/internal/domain/goosereach/goose_reach_test.go`.
   Each Python `def test_x` becomes `func TestX` with the docstring copied as a comment and
   a trailing `// py: tests/unit/test_goose_reach.py::test_x` marker.
2. **Fixtures are data, never re-typed.** Inline Python literals (dicts, SCL strings, CSV
   text) are extracted once by a generator script into
   `studio/tests/vectors/<area>/<test_file>/<test_name>.json` with `{input, expected}`.
   The Python test is then changed to read the same vector (`load_vector(__file__, name)`),
   so both languages assert on one file. Helpers `goose_samples.py`, `sld_fixtures.py`,
   `license_signer.py` get a `--dump` that writes their generated SCL/keys as vectors.
3. **Expectations are compared after one normaliser** shared with the golden replay
   (section 3): uuid hex -> `<uuid:n>`, timestamps -> `<ts>`. Error text is byte-equal.
4. **`monkeypatch` seams become named fault points** (`faults.Inject("projects.save")` in
   Go; same name in the Python seam table) - never a Go global you swap.
5. A Go port is "done" only when `check_specs.py` lists the Go test next to the Python
   test for every rule the Python test enforces (the checker already accepts `.go`
   `func Name(` references).
6. Windows-only and Npcap tests keep their `skipif` conditions verbatim (`t.Skip` with the
   same reason text) so skips are counted identically in both suites.
7. Tests that read `test_cids/` (private) never produce committed vectors; their Go twin
   asserts the same structural facts and runs only where the folder exists.

## Per-file table

| Area | File | Tests | Port note |
| --- | --- | --- | --- |
| acquisition | test_goose_consistency | 3 |  |
| acquisition | test_goose_engine_process | 5 | spawns studio_goose.exe stale check |
| acquisition | test_goose_monitor_stats | 2 |  |
| acquisition | test_goose_share | 4 |  |
| acquisition | test_live_values | 3 |  |
| domain | test_access_policy | 3 |  |
| domain | test_csv_trace | 12 | CSV parse/format of floats |
| domain | test_energization | 4 |  |
| domain | test_equipment_control | 4 |  |
| domain | test_equipment_state | 3 |  |
| domain | test_equipment_test_buttons | 5 |  |
| domain | test_gateway_domain | 9 |  |
| domain | test_goose_inputs | 6 |  |
| domain | test_goose_link_map | 9 |  |
| domain | test_goose_link_map_vendor_files | 2 | private `test_cids/` only; local structural parity, never golden |
| domain | test_goose_reach | 7 |  |
| domain | test_goose_routing | 3 |  |
| domain | test_health_mode_points | 2 |  |
| domain | test_interlock_hints | 3 |  |
| domain | test_master_device_table | 2 |  |
| domain | test_master_rules | 7 |  |
| domain | test_mms_access | 5 |  |
| domain | test_model_compare | 3 |  |
| domain | test_passthrough | 3 |  |
| domain | test_real_ied_refusal_text | 1 |  |
| domain | test_samples | 2 |  |
| domain | test_scenarios | 3 |  |
| domain | test_signal_flow | 6 |  |
| domain | test_sld_bindings | 4 |  |
| domain | test_sld_layout | 9 | monkeypatch; float layout vectors |
| domain | test_station_health | 4 |  |
| domain | test_sv_supervision | 4 |  |
| domain | test_sv_values | 3 |  |
| domain | test_test_mode | 5 |  |
| gateway | test_gateway_safety | 2 |  |
| launcher | test_app_window | 12 | Windows window lifecycle; launcher may stay Python shell |
| launcher | test_directory_lease | 2 | spawns a 2nd process; lock-file format shared by both engines |
| launcher | test_portable_runtime | 6 | launcher |
| licensing | test_licensing | 17 | shares `license_signer` keys; extend go_vector.json |
| licensing | test_licensing_legacy | 6 | legacy files read-compat vectors |
| networking | protocol/test_adapter_listing | 1 | Windows-only, compares with Get-NetAdapter |
| networking | test_address_plan | 3 |  |
| networking | test_change_plan | 4 | plan digest = json.dumps(sort_keys) bytes |
| networking | test_endpoints | 3 |  |
| networking | test_gateway_firewall | 5 | Windows calls faked |
| networking | test_ip_helper | 4 | Windows IP Helper (ctypes) -> Go syscall |
| networking | test_network_helper | 3 | elevated helper runner (may stay Python helper) |
| networking | test_network_journal | 3 | journal JSONL sort_keys bytes |
| observability | test_append_only_logs | 2 | monkeypatch; JSONL byte vectors |
| projects | test_bulk_removal | 2 | monkeypatch -> Go fault seam |
| projects | test_compact_archive | 4 | monkeypatch; project.json byte vectors |
| projects | test_diagram_drawing | 8 |  |
| projects | test_equipment_colors | 9 |  |
| projects | test_equipment_mapping | 12 | largest projects suite (12) |
| projects | test_format_11_migration | 3 |  |
| projects | test_ied_removal | 1 |  |
| projects | test_live_roles | 3 |  |
| projects | test_project_persistence | 6 |  |
| projects | test_relay_statuses | 9 |  |
| projects | test_stored_import | 2 | monkeypatch import counter |
| projects | test_test_panel | 3 | monkeypatch |
| runtime | test_command_endpoint | 4 | spawns native; ADR-0069 |
| runtime | test_goose_settings | 2 |  |
| scl | test_goose_scl | 5 |  |
| scl | test_real_world_scl | 2 | vendor patterns as `make_template` variants -> corpus cases |
| scl | test_report_clients | 3 |  |
| scl | test_report_mapping | 3 |  |
| scl | test_schema_validation | 5 | lxml XSD; message text from libxml2 |
| scl | test_scl_export | 4 | exported XML byte-compare |
| scl | test_scl_import | 3 | error strings byte-compare |
| scl | test_substation_scl | 3 |  |
| scl | test_sv_scl | 1 |  |
| scl | test_templates | 2 |  |
| simulation | test_cadence | 4 |  |
| simulation | test_initial_state | 3 |  |
| tools | test_explorer_connect | 4 | fake worker; IPC transcript vectors |
| tools | test_explorer_find | 6 |  |
| tools | test_tool_autonomy | 6 |  |
| updates | test_update_check | 6 | fake opener; also checks website version |

## Totals by area (class c + c-launcher + adapter_listing)

| Area | Files | Tests | Go package (proposed) |
| --- | --- | --- | --- |
| domain | 29 | 133 | `internal/domain/...` (pure, no I/O; first to port) |
| projects | 12 | 62 | `internal/projects` (document, archive, mapping migration, removal) |
| scl | 10 | 31 | `internal/scl` (parse, import, cfg builder, export, schema) |
| networking | 8 | 26 | `internal/networking` (+ Windows syscall layer) |
| licensing | 2 | 23 | `internal/licensing` (vectors already shared with Go license server) |
| launcher | 3 | 20 | `cmd/scadabench` launcher (last phase) |
| acquisition | 5 | 17 | `internal/acquisition` |
| tools | 3 | 16 | `internal/tools` |
| simulation | 2 | 7 | `internal/simulation` |
| runtime | 2 | 6 | `internal/runtime` |
| updates | 1 | 6 | `internal/updates` |
| gateway | 1 | 2 | `internal/gateway` |
| observability | 1 | 2 | `internal/observability` (append-only JSONL writer) |
| **Total** | **79** | **351** | |
