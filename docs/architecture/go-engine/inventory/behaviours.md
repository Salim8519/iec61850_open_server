# Behaviours the user can notice (launch, session, license, updates, dev vs packaged, texts)

Paths relative to `studio/backend/studio_app/` unless noted. File formats are in persistence-*.md;
process details in processes-*.md; frontend reactions in frontend-events.md.

## 1. Launch sequence (launcher/entrypoint.py)
1. `--network-helper ...` as argv[1] -> run the elevated helper and exit (entrypoint.py:21-23).
   The Go exe must keep this mode (UAC prompt runs the same exe).
2. Attach to the parent console if started from a terminal (windowed.py:58-70).
3. Arguments: `--port` (8765), `--no-browser`, `--data-dir`, `--self-test`, `--dev`,
   `--dev-token` (env `STUDIO_DEV_TOKEN`, "dev"), `--frontend-url` (env `STUDIO_FRONTEND_URL`,
   `http://127.0.0.1:5173`), `--native-exe`. Unknown args -> argparse usage error (exit 2).
4. Packaged + `--dev` -> exit text "Developer mode (--dev) is not available in the installed ScadaBench. Run it from source with studio/tools/run_dev.py instead."
5. Packaged only: move `%LOCALAPPDATA%\SubstationStudio` to `...\ScadaBench` once
   (legacy_folder.py); set `STUDIO_MODULE_PATH`/`STUDIO_MODULE_SIG` when a signed `.pyd`
   ships (native workers verify it; a Go build needs an equivalent signed artefact).
6. `--self-test`: prints one line `{"ok": true, "schema": "SCL2007B4", "mms_value": 42.5, "array_value": 17.25, "edited_export_verified": true, "stopped_recovery_verified": true, "ied_deletion_verified": true, "saved_panel_and_pulse_verified": true, "customer_data_used": false, "clean_machine_verified": false}`
   (json.dumps defaults: insertion order of self_test.py:113, `", "`/`": "` separators) and
   exits; it runs a real native MMS round trip in a temp folder. Packaging tests call it.
7. Logging: terminal -> stderr INFO; windowed -> `data/logs/launcher-YYYYMMDD.log`.
8. Data-folder lease `studio.lock` (directory_lease.py). If held by another process:
   dev -> RuntimeError text "This ScadaBench data folder is already open or cannot be locked. Close its other launcher before opening it again.";
   packaged -> `show_running_instance`: focus window of `launcher.json.window_pid`, else
   SetEvent `Local\ScadaBench-Show-<sha256(normcase(abspath(data)))[:24]>`; success prints
   "ScadaBench is already running; its window was brought to the front." (terminal only),
   exit 0; else message box with the lease error, exit 1. Starting again is near-instant
   because heavy modules load only after the lease.
9. Server: Waitress on 127.0.0.1:`--port`, 16 threads. Busy port: dev exits "Port 8765 is busy. Stop the other ScadaBench backend before starting dev mode."; packaged silently uses an OS-chosen port.
10. Writes `launcher.json` `{"url": "http://127.0.0.1:<port>"}` and prints `ScadaBench: <url>`.
    Packaged prints `Open in your browser: <url>#session=<token>`; dev prints
    `Vite frontend: <frontend>/#session=<token>` and the "never opens tabs" line, plus a
    warning when Vite is not reachable within 15 s.
11. Packaged and not `--no-browser`: open Edge/Chrome `--app=<url>#session=<token>&window=app`
    with profile `data/browser-profile[-2|-3]` (or a temp dir), 1440x900; rewrite
    `launcher.json` with `window_pid`. No browser -> default browser via `webbrowser.open`
    (without `&window=app`). Closing the window stops the app (interrupt main thread), with
    handoff detection through the profile `lockfile`.
12. Every non-dev launch starts the "show yourself" listener (named event above).
13. `create_app` starts: license background verify (packaged), update check thread
    (unless env `SCADABENCH_UPDATE_CHECK=off`), background project recovery, gateway-resume thread.
14. Exit: listener closed, `runtime.close()` (stop the Run, flush/close run log), server closed,
    lease released.

## 2. Startup recovery (projects/startup_recovery.py, background_recovery.py)
- No autosave -> `startup_recovery: null`. OK -> `{"message":"Recovered the last saved substation. All IEDs remain stopped.","status":"restored"}`.
- Unreadable -> copy to `recovery/unreadable-autosave-<hex>.subsim`, result
  `{backup, details: str(error), message:"The last autosave could not be reopened. A separate recovery copy was preserved before allowing new work.", status:"unreadable"}`; the app then starts with an empty project (next save overwrites autosave).
- Symlink/non-file autosave or failed backup -> recovery error -> all engineering calls 503
  `startup_recovery_failed` (http-conventions.md).
- Restore never persists (`persist=False`) and never starts IEDs; it bumps revision once.
- A legacy format (< 15) shows `migration_notice {message, details[], backup}` in `/project`
  with format-specific texts (projects/service.py:218-242); lab projects without `goose_reach`
  get the "GOOSE now stays inside ScadaBench unless you share it." notice (projects/goose_reach.py).

## 3. Sessions, tabs and dev session reuse
- The token travels only in the URL fragment `#session=`; FE posts it to `/session` and strips it.
- Dev (`start-studio.cmd` -> `studio/tools/run_dev.py`): if port 8765 is busy, probes Vite
  ports 5173..5199 with `POST <vite>/api/v1/session {"token": <dev token>}` + `Origin: <vite>`
  (0.5 s timeout) and reuses the first that answers 200, printing
  "ScadaBench development mode is already running: <url>" and "Open in your browser: <url>/#session=<token>".
  So the Go dev backend must: accept the fixed dev token, accept requests proxied by Vite whose
  `Origin`/`X-Studio-Frontend` equals `--frontend-url`, and answer 200 `{"ok":true}`.
- Dev: a tab from an old Vite origin gets 403 + `active_frontend` and is redirected (client.ts).
- Dev rebuilds: run_dev rebuilds the Go GOOSE engine if stale; the backend reloads on Python
  source changes and on the native worker exe change (`run_with_reloader`, interval 0.5 s,
  extra_files = studio_ied.exe). A Go backend needs an equivalent watch/restart that keeps the
  same port, token and cookie secret (`studio-dev-secret`) so open tabs stay signed in.
- AGENTS.md: a launcher must identify an existing session before reusing it, and must never
  silently run incompatible backend/native code (capability checks below).

## 4. Capability and staleness checks (refuse instead of running old code)
| Process | Required capabilities | Refusal text |
| --- | --- | --- |
| studio_ied | versioned_updates, atomic_control_feedback, control_selection_events, standard_controls, goose_aware, goose_bus | "Native engine is outdated. Use the rebuilt portable application or rebuild the native worker before Run" (runtime/worker_settings.py:4-7) |
| studio_live (real IED) | live_discover, live_reports, live_poll, read_only_client | "The live client is outdated; rebuild the native programs" |
| studio_live (Explorer) | explore | same |
| studio_live (GOOSE) | `goose_start` must return `inbox_port` | "The GOOSE listener is outdated (no goose_inbox). Rebuild studio_live.exe." |
| studio_goose | goose_engine_v1 | "The GOOSE engine is outdated. Rebuild it with studio/tools/build_goose_engine.py before Run"; missing: "The GOOSE engine (studio_goose.exe) is not built yet; run studio/tools/build_goose_engine.py" |
| studio_control | control_operate | "The control program is outdated; rebuild the native programs" |
| studio_gateway | gateway_iec104, gateway_modbus, gateway_dnp3, commands_via_backend | "The gateway program is outdated; rebuild the native programs" |
| studio_master | master_<protocol> | "ScadaBench cannot act as <IEC 104/DNP3/Modbus TCP> master in this build yet" |
Native path: dev `studio/native/build/Release/` (or `--native-exe`), installed `_MEIPASS/native/`,
one-file portable: workers staged to `ScadaBench-data/workers/<content key>` (persistence-1.md).

## 5. License gating (packaged only; `enforce = sys.frozen`)
- Every `/api/v1/*` except `/api/v1/license*` and `/api/v1/session` answers 402 until licensed;
  FE fires `studio-license-required` and shows the license screen.
- Status evaluation order: integrity (debugger -> "ScadaBench cannot run while a debugger is attached.",
  changed native files -> "ScadaBench program files were changed or damaged. Reinstall ScadaBench."),
  revoked reason from `license_state.json`, missing cert -> NO_LICENSE text, bad signature ->
  certificate error text, then domain/license_policy.evaluate_certificate texts (wrong machine,
  clock rollback with 300 s tolerance, trial ended, expired, unsupported version, damaged).
- Machine ID: v2 hwid, v1 legacy hwid only for certificates issued before 1814400000 (2027-07-01).
- Background verify (lifetime only, every 7 days): `revoked`/`moved` renames `license.cert` to
  `license.cert.<verdict>-<unix>` and stores the reason ("This license was cancelled. Contact the seller if this is a mistake." / "This license key was moved to another computer.").
- Dev: `GET /license/status` -> licensed, mode `development`, plan `dev`; actions -> 400 code `development`.
- `STUDIO_LICENSE_CERT_PATH` env is set for native workers (they check the cert themselves).

## 6. Update check (ADR-0071, updates/service.py)
- Never downloads; only `GET <license server>/api/v1/release` with `If-None-Match` ETag,
  verifies Ed25519 over `b"scadabench-release-v1\n" + payload`, cleans the record
  (scadabench.com hosts only, https, sha256 links).
- Automatic: 20..120 s after start, then hourly checks run when 24 h passed and `auto` true.
- `available` = latest > current (`0.1.14+dev` compares as 0.1.14); `important` when
  `min_version` > current; `dismissed` hides only non-important for that exact version.
- Errors are stored codes, never HTTP errors. FE refreshes status every 30 min.

## 7. Dev vs packaged differences that are allowed (and must stay the only ones)
| Aspect | Dev | Packaged |
| --- | --- | --- |
| Frontend | Vite on 5173.. proxies `/api` to 8765 | served by backend (`frontend.pak` or folder) |
| Origin check | `X-Studio-Frontend`/Origin must equal `--frontend-url` | Origin netloc must equal Host |
| Token/secret | fixed `dev` / `studio-dev-secret` | random per launch |
| License | not enforced | enforced (402) |
| Port busy | exit with message | OS-chosen port |
| Window | never opens anything | app window, single-instance show |
| Data folder | `studio/data` (run_dev passes `--data-dir`) | `<exe>/data` if exists, else `%LOCALAPPDATA%\ScadaBench\data`; portable `ScadaBench-data` |
| Reload | backend auto-restarts on source/worker change | none |
| Version | `0.1.14+dev` | release version |
| Lease conflict | raises | focus running instance |
Everything else (defaults, project files, protocol semantics, texts) is identical (AGENTS.md).

## 8. Backend text shown verbatim in the UI (sources to port word for word)
- `error` of every 4xx/5xx (frontend client.ts shows `data.error`; 404 replaced by FE text).
- Status `message` fields: runtime/live session (`real_session.py` "Connecting to H:P",
  "Reading the device data model", "Receiving live values by report, and polling the rest", ...),
  GOOSE/SV status messages (goose_monitor.py:258-273, sv_monitor.py:450-467, domain/goose_reach.py),
  gateway/master/explorer/scan messages (api-routes-3.md), network plan `hint`s and change
  `description`s (networking/service.py, change_plan.describe/describe_removal), health check
  `message`s (domain/station_health.py, health_model_checks.py, goose_consistency.py), comparison
  hints (domain/model_compare.py), mode badges, interlock hint texts (domain/interlock_hints.py),
  explorer value `text` (domain/explorer_values.py), master `value_text`/`quality_text`
  (domain/master_points.py), gateway `value_text` (domain/gateway_values.py), Npcap message,
  migration notices, license reasons, update error codes (FE maps codes to words).
- Event `message`s (events.md) appear in Activity.
- Health HTML report and CSV exports (domain/health_export.py) are user files.

## 9. Concurrency semantics visible to users
- `cycle_lock` serialises Run/Stop/pause, deletes, network/IED settings, test panel save, equipment
  operations and every automation tick; a long Stop blocks those requests (not reads).
- Waitress 16 threads: up to 16 concurrent requests incl. one per open SSE stream; extra
  requests queue. Frontends poll at 300 ms..3 s (frontend-callers-2.md polling summary).
- Startup requests block up to 300 s while the project opens (no FE timeout).

## Parity risks
- Lease + named-event + `launcher.json` contract must interoperate with already-installed
  Python builds sharing the same data folder (one must detect the other).
- Self-test JSON line and `--network-helper` mode are used by the installer and UAC flow.
- Signed-module check by native workers (`STUDIO_MODULE_PATH`) is tied to the Python `.pyd`.
- Dev session reuse depends on `/session` answering through the Vite proxy with the dev token.
