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

acquisition 5 17
domain 29 133
gateway 1 2
launcher 3 20
licensing 2 23
networking 8 26
observability 1 2
projects 12 62
runtime 2 6
scl 10 31
simulation 2 7
tools 3 16
updates 1 6
