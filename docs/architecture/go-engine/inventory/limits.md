# Limits, defaults, timeouts and cadences (file:line under studio/backend/studio_app/)

Every value below is user-visible (a refusal text, a timing or a default). Keep the inclusive
bounds and the bool guards exactly. Process-level timeouts are repeated in processes-*.md.

## Project and model size
| Limit | Value | Where |
| --- | --- | --- |
| IEDs per project | 20 (import, restore, bulk delete, preview slots) | projects/service.py:50,178; api/scl.py:28,38; projects/bulk_removal.py:11 |
| Upload body | 100 MiB (`MAX_CONTENT_LENGTH`, 413) | app_factory.py:43 |
| SCL document | 20 MiB ("SCL exceeds the 20 MB import limit") | scl/document.py:6,13 |
| Project archive unpacked | 100 MiB; legacy single-entry archives 256 MiB | projects/archive.py:8-9 |
| Diagram nodes / wires | 2000 / 5000; node id `^[A-Za-z0-9_-]{1,64}$`; wire id <=128 | contracts/project.py:86,94,102-103 |
| Node label | 1..100 chars; relay statuses <=50; positions -100000..100000 finite | contracts/project.py:7-8,61,68 |
| Push-button reset | 100..60000 ms, default 1000 | contracts/project.py:74 |
| Sources | <=50000 definitions; period >= 0.1 s; finite numbers | projects/validation.py (validate_sources); domain/source.py:18-21 |
| CSV trace | 5 MiB, 2..256 columns, <=100000 rows, time 0..86400 s, filename <=255 | domain/csv_trace.py:8-9,18,60,71 |
| Scenarios | <=100; name 1..100; 1..1000 steps; step time 0..86400 s | domain/scenario.py:9,20-27,39 |
| Test panel | <=500 controls; id `[A-Za-z0-9_-]{1,64}`; label 1..100; pulse 100..60000 ms | projects/test_panel.py:16-40 |
| Pulse | 100..60000 ms int, Boolean only | simulation/pulses.py:18 |
| Sample batch | 256 attributes per native write ("Sample batch exceeds the native 256-attribute limit") | simulation/sample_writer.py:131 |
| Automation batch | 80 points per write, pool 20 threads | simulation/automation.py:12,33 |
| SCL export overrides | <=50000 references | scl/exporter.py:14 |
| XSD errors returned | first 100 | scl/schema_validation.py:50 |
| Report count per block / model arrays | 1..64 / 0..256 | scl/reports.py:64; scl/model_builder.py:27 |
| GOOSE id length | <=129, no space/`();{}` | scl/goose_controls.py:10 |
| Project name | truncated to 100 chars on restore | projects/service.py:214 |
| FLOAT32 range | abs <= 3.4028235e38 | domain/signal.py:14 |
| Text signal | <= size UTF-8 bytes | domain/signal.py:39 |

## API paging and request caps
| Limit | Value | Where |
| --- | --- | --- |
| Signals page | `limit` 1..250 (default 100), `offset` >= 0 | api/signals.py:36-37 |
| Real-IED read | 1..500 refs per request; session reads <= 2000; watched points <= 500 | api/live.py:46; acquisition/real_session.py:120; acquisition/manager.py:29 |
| Explorer read / watch | 1..200 attributes; interval 500..60000 ms (default 1000) | tools/explorer.py:11-12,133,161-165 |
| Explorer key-point watch | <=16 nodes, <=40 points, per node MMXU 8, MMXN 4, MSQI 4 | domain/explorer_watch.py:9-11 |
| Tool memory | 50 hosts, 200 watch items, host<=64, ref<=200, fc<=4, choice<=200 | tools/tool_memory.py:11-12,333-359; api/tools.py:118 |
| Find IEDs | <=256 addresses (/24), 64 parallel, TCP connect 0.5 s, identify 10 s (`timeout_ms` 3000) | domain/scan_range.py:4; tools/ied_scan.py:9-10,96 |
| Gateway points | <=5000 per protocol; description <=120; CSV <=2 MiB | domain/gateway_points.py:15; domain/gateway_rules.py:85,102; domain/gateway_csv.py:7 |
| Gateway logs | traffic 1000 lines, commands 200; `after` returns last 300 (commands 200) | gateway/traffic_log.py:6-7,22; api/gateway.py:138 |
| Master | ranges <=20, probe <=10000 addresses, labels <=20000 (key<=40, text<=80), routes <=500, unticked <=20000 | domain/master_rules.py:24-26,133,152; api/gateway_master.py:91-108 |
| Master tables | coils 2000, discrete 2000, holding 125, input 125 per read | domain/master_rules.py:22 |
| Bulk IED selection | 1..20 unique ids | projects/bulk_removal.py:11 |
| List filters | 40 lists, 16 keys, text <=120, list id `^[a-z][a-z0-9-]{0,39}$`, key `^[a-z][a-zA-Z0-9_]{0,31}$` | projects/list_filters.py:7-12 |
| Preferences | tour_seen_version 0..1000; sld layers 0..1 | projects/preferences.py:126 |
| Update check | body <= 16 KiB; notes <= 600; versions `\d{1,4}.\d{1,4}.\d{1,4}` | updates/service.py:29; updates/versions.py:10,162 |
| License server reply | 1 MiB read; PoW difficulty 1..26 | licensing/client.py:9,328 |
| SSE bus | 5000 events kept | runtime/event_bus.py:8 |
| IPC frame | 1..16 MiB, 4-byte little-endian length + UTF-8 JSON, `"version":1` | runtime/worker_protocol.py:5,19-29 |
| Run log segment | 10 MiB per `events-NNNN.jsonl` | observability/log_writer.py:7 |
| Network helper result text | 500 chars | networking/helper/runner.py:25; networking/journal.py:24 |
| Manual address | prefix 8..30 | networking/address_plan.py:95-96 |
| Re-pick rounds | 5 | networking/service.py:19 |

## Defaults
| Item | Default | Where |
| --- | --- | --- |
| New project | name "Untitled substation", empty lists, `DEFAULT_NETWORK`, `empty_gateway()` | projects/service.py:35-37 |
| Restored project name | "Imported substation" | projects/service.py:214 |
| Template IED name | "BayController" | api/scl.py:47 |
| Network | mode loopback, nominal_hz 50 (50/60), goose_reach "inside" | domain/network_settings.py:11-13 |
| Real IED live | ip "", port 102, access auto_reports, poll_ms 1000 (250..60000) | domain/network_settings.py:14,61 |
| IED port | 102 (1..65535) | projects/ied_settings.py:66 |
| Loopback addresses | `127.50.0.2` .. `127.50.0.254`, first free in project order | networking/loopback.py:10-16 |
| Gateway | role gateway, protocol iec104, listen this_pc (127.0.0.1; network = 0.0.0.0), real_commands true, auto_start false | domain/gateway_rules.py:16-23 |
| Gateway IEC 104 | port 2404, CA 1, t1 15, t2 10, t3 20, k 12, w 8 | domain/gateway_rules.py:8 |
| Gateway Modbus / DNP3 | 502, unit 1 / 20000, outstation 10, master 1, unsolicited true | domain/gateway_rules.py:9-10 |
| Master IEC 104 | 2404, CA 1, OA 0, t0 10, t1 15, t2 10, t3 20, k 12, w 8, time_tagged false, gi_minutes 15 | domain/master_rules.py:11-12 |
| Master DNP3 / Modbus | 20000, outstation 10, master 1, integrity 15 min, event 5 s, unsolicited true / 502, unit 1, poll_ms 1000 (200..60000) | domain/master_rules.py:13-15,20 |
| Control options | originator remote (bay 1, station 2, remote 3, maintenance 7), interlock_check true, synchro/test false, or_ident "ScadaBench" | tools/control_client.py:10-23,55 |
| Gateway manual command originator | station; SCADA commands remote | gateway/commands.py:65 |
| Source spec | offset 0, amplitude 1, period 10, seed 1 | domain/source.py:14-16 |
| Pulse | active_value true, duration 1000 ms | api/test_panel.py:33-34 |
| Explorer/scan port | 102 | api/tools.py:41,126,132 |
| HTTP port | 8765, falls back to an OS-chosen port (packaged only); bind 127.0.0.1, 16 threads | launcher/entrypoint.py:26,159-163 |
| Dev frontend | `http://127.0.0.1:5173` (run_dev picks 5173..5199), dev token "dev" | launcher/entrypoint.py:32-35; tools/run_dev.py |
| License server | `https://license.scadabench.com` | licensing/release_key.py:7 |
| Tour version | 1 | projects/preferences.py:12 |
| App version string | `0.1.14+dev` in source; real version written into the protected build | version.py:8 |
| Run log `application` field | literal "0.1.0" | observability/log_writer.py:29 |

## Timings and cadences
| Item | Value | Where |
| --- | --- | --- |
| Automation tick | 0.1 s, max backlog 5 (skipped_ticks), `automation_late` when lateness >= 0.1 s | simulation/cadence.py:6; coordinator.py:241 |
| Pulse checker | every 20 ms | simulation/pulses.py:65 |
| Gateway value sync | every 0.1 s, `update` batches of 500 | gateway/manager.py:10,152 |
| Gateway command observe | 3.0 s, check every 0.1 s | gateway/commands.py:14,87 |
| Own-command window | 20 s (MMS access and interlock hints) | simulation/mms_access.py:7; energization_states.py:8 |
| MMS reader shown as reading | 5 s; 12 nodes shown | domain/mms_access.py:7-8 |
| Real IED | connect 15 s, discover 120 s, rcb 30 s, read 30 s, disconnect 15 s, mode refresh 10 s, retry 2 s doubling to 30 s, join 20 s | acquisition/real_session.py:10,71-83,120-156 |
| Health mode freshness | 30 s | domain/station_health.py:11 |
| GOOSE / SV listener start | 30 s | acquisition/goose_monitor.py:36; sv_monitor.py:40 |
| Default native request | 10 s; close: `stop` 3 s, wait 3 s, terminate 3 s, kill | runtime/worker_process.py:57,84-97 |
| studio_probe control | 15 s subprocess | runtime/mms_client.py:21 |
| Gateway start/halt | 15 s / 10 s; join sync 5 s | gateway/manager.py:98,170-173 |
| Master | connect t0+10 s, interrogate 60 s, poll 30 s, command 40 s, stop 10 s, retry 10 s, probe 1500 s | gateway/master_manager.py:17,68,110,120,202,219; gateway/modbus_probe.py:8 |
| Control client | 30 s per operate (one at a time) | tools/control_client.py:55 |
| Explorer | connect 15 s (`timeout_ms` 5000), explore node 60 s, others 30 s, disconnect 10 s, poll join 10 s | tools/explorer.py:89,109,125,200,214 |
| Adapter cache | 10 s | networking/service.py:41 |
| PowerShell/firewall queries | 20 s / 60 s; helper 60 s | networking/adapters.py:28; firewall_rules.py:8,35,50; helper/runner.py:15 |
| Startup gate | 300 s | api/startup_gate.py:9 |
| License | status cache 60 s, online verify every 7 days, HTTPS timeout 20 s, clock tolerance 300 s | licensing/service.py:16-17; client.py:8; domain/license_policy.py:9 |
| Update check | first after 20..120 s random, then hourly (+0..600 s) when due (24 h), manual repeat gap 60 s, timeout 6 s | updates/service.py:25-28,149-156 |
| SSE | `retry: 1000`, heartbeat after 10 s idle | api/events.py:15-19 |
| Shutdown | coordinator thread join 3 s; feedback 3 s; pulses 12 s | simulation/coordinator.py:253; native_control_feedback.py:103; pulses.py:70 |
| App window | quick-exit 15 s, handoff grace 5 s, poll 0.5 s; size 1440x900 | launcher/app_window.py:24-26,136 |
| Dev frontend wait | 15 s (connect checks every 0.25 s) | launcher/entrypoint.py:85-94 |

## Parity risks
- Python `isinstance(x, int)` accepting `true` where no bool guard exists (explorer/scan port).
- Inclusive ranges and the exact refusal texts that quote them.
- `noise` generator uses Python `random.Random(seed + int(elapsed*10)).uniform` (Mersenne Twister
  with Python int seeding): Go must re-implement CPython's seeding to give the same values.
- Cadence arithmetic (`floor((lateness + period*1e-8)/period) + 1`) and float accumulation of `elapsed`.
