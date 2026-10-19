# 1. Test inventory (snapshot 2026-10-19, master @ 3ccd473 + working tree)

Source data in this folder: `scan.txt` (per-file features/imports, produced by `scan.py`),
`collect.txt` (`pytest --collect-only -q`, run with `PYTHONPATH=..` so the two
repo-root tests collect), `classes.txt` (per-file class, produced by `classify.py`).

## Totals

| Suite | Files | Tests | Runner |
| --- | --- | --- | --- |
| Python `tests/unit` | 102 test + 3 helpers | 410 | pytest (`pythonpath=["backend"]`) |
| Python `tests/integration` | 16 | 50 | pytest |
| Python `tests/protocol` | 34 test + 4 rigs/stand-ins | 88 | pytest, most `skipif` native/Npcap missing |
| Python `tests/packaging` | 1 | 2 | pytest, skip unless EXE built |
| Python total | 151 `.py` (144 test files, 6 helpers, conftest) | **550** | |
| Frontend `tests/frontend` | 42 `*.test.ts` | 178 | `node --experimental-strip-types --test` |
| Go `goose_engine` | 5 `_test.go` | 21 `func Test*` | `go test ./...` (via `verify.py`) |

Collection without `PYTHONPATH=..` fails on `unit/test_history_log.py` and
`unit/test_runtime_log_query.py` (`from studio.tools...`) -> 547 + 2 errors. The parity
harness must set the repo root on `sys.path` (one line in `tests/conftest.py`).

## Classes

| Class | Meaning | Files | Tests |
| --- | --- | --- | --- |
| a-pure | HTTP only through `app.test_client()`; internals touched only for teardown (`runtime.close`) | 4 | 11 |
| a-seam | HTTP tests that also inject Python objects, fakes or read internals (`app.extensions[...]`, `monkeypatch`) | 14 | 47 |
| b-app | Whole app over HTTP **plus** independent wire client (`studio_probe.exe`, 104/Modbus stand-ins) | 9 | 18 |
| b-coord | Native run driven through Python `Coordinator`/`ProjectService` directly, no HTTP | 3 | 3 |
| b-ipc | Native worker driven through Python `WorkerProcess` IPC client (+ `import_scl` for cfg) | 19 | 62 |
| b-indep | Wire tests with no `studio_app` import (Go engine + Npcap peer) | 3 | 6 |
| c | Python-internal unit tests (domain, scl, projects, networking, ...) | 75 | 330 |
| c-launcher | Launcher/window/data-folder lease unit tests | 3 | 20 |
| e | Packaging, release, native-build, license tooling, history/log tools | 14 | 53 |
| d | Frontend node:test | 42 | 178 |
| Total Python | | 144 | 550 |

Class (a) as requested = a-pure + a-seam = **18 files / 58 tests**; class (b) = b-app +
b-coord + b-ipc + b-indep = **34 files / 89 tests** (`test_adapter_listing` counted in
b-ipc, but it is really a Windows IP-Helper unit test: port with networking).

## Class (a): black-box HTTP

| File | Tests | Seam it needs today | Harness translation |
| --- | --- | --- | --- |
| integration/test_preferences_api | 4 | none | runs unchanged |
| integration/test_signal_flow_api | 2 | none | unchanged |
| integration/test_signal_layers_api | 3 | none | unchanged |
| integration/test_station_health_api | 2 | none | unchanged |
| integration/test_api | 6 | `projects.restore/add_template/project`, `events.after`, `runtime.state`, `admin_origin=` (dev) | `_parity/inspect` views; `seams.admin_origin` |
| integration/test_import_selection | 5 | `projects.export`, `projects.project` | inspect `project_export`, `project_raw` |
| integration/test_license_api | 4 | injected `LicenseService(hwid, clock, keys, enforce)` | `seams.license` |
| integration/test_mapping_api | 3 | `runtime.state`, `projects` | inspect `runtime_state` |
| integration/test_network_api | 4 | replaces `_list_adapters/_list_rules/_probe/_elevate` with lambdas | `seams.network` (data-driven fakes) - **needs edit** |
| integration/test_network_setup_page | 3 | `network._probe` lambda | `seams.network.reachable` - **needs edit** |
| integration/test_scl_export_api | 4 | `projects.add_template/export` | inspect + `POST /ieds/template` |
| integration/test_sld_import_api | 3 | `projects.project` | inspect `project_raw` |
| integration/test_test_mode_api | 1 | `projects.export/project` | inspect |
| integration/test_tools_api | 2 | explorer/scanner close, own `WorkerProcess` peer | teardown via process stop; peer stays Python (it is the independent side) |
| integration/test_update_api | 1 | injected `UpdateService(opener, clock, keys)` | `seams.updates` (canned answers) |
| unit/test_infeed | 4 | `projects.add_template` + domain calls | split: HTTP part -> a, domain part -> Go unit |
| unit/test_startup_gate | 3 | monkeypatches `recover_startup`, `shutil.copyfile` | `seams.faults` named fault points |
| unit/test_startup_recovery | 4 | fault on `copyfile`/`projects.save`, `supervisor.workers` | `seams.faults` + inspect `workers` |

## Class (b): protocol / native

| Group | Files | Engine dependence | Plan |
| --- | --- | --- | --- |
| b-app | converter_runtime, energization_runtime, gateway_runtime, interlock_hints_runtime, mapped_equipment_control, master_modbus_runtime, master_runtime, run_single_ied, signal_pulses | create_app + HTTP; also `events.after`, `runtime.start`, `supervisor.read`, `projects.diagram/add_template` | run against both engines via harness; `supervisor.read` -> inspect `worker_read`; `events.after` -> `GET /api/v1/events?after=0` (public) |
| b-coord | csv_runtime, scenario_runtime, unit/test_automation_failure | Python `Coordinator` object | rewrite as b-app (HTTP scenario/trace endpoints exist: `PUT /scenarios`, `POST /traces/preview`) **and** port as Go coordinator tests |
| b-ipc | control_client, explorer_wire, gateway_wire, goose_consistency_wire, goose_inputs_wire, goose_inside, goose_runtime, live_client, master_wire, master_counters, master_modbus_wire, mms_access_wire, native_browse, native_reports, native_samples, native_worker, sv_listen, test_mode_runtime | Tests the C++/Go workers; Python only supplies the IPC client and the cfg from `import_scl` | keep as native contract tests; parametrize the cfg producer (`cfg_for(scl)` uses the selected engine's SCL dump) and add a Go IPC-client twin per file (`internal/worker/*_test.go`) |
| b-indep | goose_conflict_wire, goose_engine_latency, goose_share_wire | none (Go GOOSE engine + Npcap peer) | unchanged; already engine-neutral |

Independent clients that must stay non-engine code: `studio_probe.exe` (native),
`iec104_standin.py`, the Modbus stand-in in `test_master_modbus_wire.py`,
`npcap_peer.py`. These are the "independent client" evidence required by AGENTS.md.

## Class (e): packaging / tools

`packaging/test_portable` (2), `unit/test_packaging_windowed` (1),
`unit/test_protected_packaging` (12), `unit/test_release_guards` (4, also creates the app),
`unit/test_native_build_config` (4), `unit/test_live_contract` (4),
`unit/test_native_enforcement` (8), `unit/test_native_license` (6),
`unit/test_license_admin` (3), `integration/test_license_server_e2e` (3, Python client vs
Go license server), `unit/test_benchmark_provenance` (1), `unit/test_dev_launcher` (2),
`unit/test_history_log` (2), `unit/test_runtime_log_query` (1).
Repo tooling tests stay Python. Packaging tests gain a Go-binary variant only when the
packager starts shipping the Go core (phase G5). `test_native_license` /
`test_license_server_e2e` already prove a Go<->Python licensing contract
(`tests/fixtures/licensing/go_vector.json`) and are the template for every other
cross-language vector.

## Class (d): frontend

42 files / 178 tests, pure TypeScript logic imported from `frontend/src` (copy, equipment,
state, api/sharedPoll, api/signalLayers types). None calls the backend; they are engine
independent and run unchanged. The engine matters for them only through
`frontend/src/api/types.ts`: the golden replay (section 3) is what checks Go responses
against the shapes the frontend consumes.

## How the app is created today

- `tests/conftest.py` only sets `SCADABENCH_UPDATE_CHECK=off`. There are **no shared
  fixtures**; each file builds its own app.
- 28 test files call `studio_app.app_factory.create_app` directly:
  `create_app(Settings(tmp_path, ROOT/"studio/native/build/Release/studio_ied.exe", tmp_path), "<token>"[, admin_origin=..., license_service=..., update_service=...])`.
  `Settings(directory, native_executable, frontend)` - the data dir is `tmp_path`.
- Requests go through Flask's in-process `app.test_client()`; the session handshake is
  `POST /api/v1/session {"token": "<token>"}` (cookie `studio_session`), host `localhost`.
- Teardown is `app.extensions["runtime"].close()` (and `explorer/scanner/gateway/master`
  stops in tool tests). Background threads (`BackgroundRecovery`, update check,
  license verify, gateway resume) start inside `create_app`; tests call
  `extensions["startup"].wait(30)` where they need recovery finished.
- Native binaries come from `STUDIO_NATIVE_BIN` (default `studio/native/build/Release`);
  protocol tests `skipif` when absent, Npcap tests also need `STUDIO_GOOSE_ADAPTER`.
- Internals reached through `app.extensions`: `projects` (add_template, project, export,
  restore, snapshot, diagram), `events.after`, `runtime` (state, start, close,
  supervisor.read, supervisor.workers), `startup` (wait, done), `network` (fakes),
  `explorer`, `scanner`, `gateway.stop`, `master.disconnect`. This is the complete list the
  parity inspect API must cover (section 2).

## Class (c) breakdown by backend area

See `02-inventory-unit.md` for every file with its Go port target (includes
`protocol/test_adapter_listing` under networking).

| Area | Files | Tests |
| --- | --- | --- |
| domain (pure rules) | 29 | 133 |
| projects (document, persistence, mapping, removal) | 12 | 62 |
| scl (import, export, goose/report/sv parsing, templates) | 10 | 31 |
| networking | 8 | 26 |
| licensing | 2 | 23 |
| launcher (c-launcher) | 3 | 20 |
| acquisition | 5 | 17 |
| tools | 3 | 16 |
| simulation / runtime / updates / gateway / observability | 7 | 23 |
| **Total** | **79** | **351** |
