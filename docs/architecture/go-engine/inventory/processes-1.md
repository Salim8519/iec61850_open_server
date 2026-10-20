# Process & IPC parity checklist (1/3): shared contract, paths, env, studio_ied.exe

Scope: every external process/IPC contract the Python backend (`studio/backend/studio_app`)
talks to. Paths below are relative to `studio/`. `py:` = Python side, `cc:` = native C++,
`go:` = `goose_engine`. Files: 1 = shared framing + studio_ied, 2 = studio_live, studio_goose,
studio_probe, 3 = studio_control, studio_gateway, studio_master, network helper, PowerShell
and other OS contracts.

## 1. Process inventory

| Executable | Launched by (py) | Style | Instances |
|---|---|---|---|
| studio_ied.exe | runtime/supervisor.py:57 via WorkerProcess | framed pipe, long-lived | 1 per running simulated IED |
| studio_live.exe | acquisition/goose_monitor.py:33, sv_monitor.py:39, real_session.py:68, tools/explorer.py:63, tools/ied_scan.py:90 | framed pipe | 1 GOOSE + 1 SV + 1 per real IED + 1 explorer + 1 per scan |
| studio_goose.exe (Go) | acquisition/goose_engine.py:52 | framed pipe | 1 per Run (lazily) |
| studio_probe.exe | runtime/mms_client.py:19, launcher/self_test.py:31 | one-shot CLI, JSON on stdout | 1 per control click / self-test read |
| studio_control.exe | tools/control_client.py:44 | framed pipe, warm singleton | 1 (respawned if dead) |
| studio_gateway.exe | gateway/manager.py:94 | framed pipe | 0..1 |
| studio_master.exe | gateway/master_manager.py:64 | framed pipe | 0..1 |
| powershell.exe | networking/adapters.py:30, firewall_rules.py:10, helper/runner.py:13 | one-shot CLI | per query / per helper item |
| ScadaBench.exe --network-helper (frozen) / python helper/run.py (dev) | networking/elevation.py:27-57 (ShellExecuteExW runas) | elevated one-shot, file IPC | per apply/cleanup |
| msedge.exe / chrome.exe --app | launcher/app_window.py:67-79 | detached GUI | 1 (packaged only) |

Unused native ops (Python never sends; Go must not need them): studio_ied `block`
(cc worker/main.cpp:48-56); studio_live `npcap_status`, `adapters`, `goose_stop`, `sv_stop`,
`rcb_release` (cc live/main.cpp:28-41); studio_control `close_all` (control/main.cpp:16).

## 2. Executable path resolution (py settings.py:28-43)

| Mode | data dir | native_executable (studio_ied.exe) |
|---|---|---|
| dev (source) | `<studio>/data` (run_dev passes `--data-dir studio/data`) | `<studio>/native/build/Release/studio_ied.exe`; override `--native-exe` (entrypoint.py:36,50-51) |
| frozen, installed | `<exe dir>/data` if exists, else `%LOCALAPPDATA%/ScadaBench/data` (settings.py:20-26) | `sys._MEIPASS/native/studio_ied.exe` (settings.py:41) |
| frozen, one-file portable | `<exe dir>/ScadaBench-data` (portable_runtime.py:25-26) | staged copy `ScadaBench-data/workers/<sha16>/native/studio_ied.exe` (portable_runtime.py:36-60) |

- Every other worker = `native_executable.with_name(<name>)` (same folder): live
  (settings.py:15-17, coordinator.py:56), goose (goose_engine.py:39), probe (mms_client.py:15),
  control (app_factory.py:52), gateway (gateway/manager.py:20), master (master_manager.py:30).
- Portable staging key: sha256 over sorted `*.exe` names+`\0`+sha256(content), first 16 hex
  (portable_runtime.py:29-33); copied into `<key>.partial-<pid>` then `os.replace`. Firewall rules
  name this path, so the Go port must produce the same folder name or users get new UAC prompts.
- Packaged set (packaging/studio.spec:9-25): ied, live, goose, probe, gateway, control, master.
- Go engine built to `native/build/Release/studio_goose.exe` (tools/build_goose_engine.py:16),
  `CGO_ENABLED=0 GOOS=windows GOARCH=amd64` (:57). run_dev rebuilds it `--if-stale` (run_dev.py:111).
- Missing-binary checks happen before spawn (`Path.exists()`), each with its own message (see per process).

## 3. Environment, working dir, window flags (all framed workers)

| Item | Value | Ref |
|---|---|---|
| argv | `[exe]` only, no arguments | py runtime/worker_process.py:18 |
| cwd | not set: inherits backend cwd (dev: repo root, run_dev.py:127; packaged: launch cwd) | worker_process.py:18 |
| env | inherited unchanged from backend process | worker_process.py:18 |
| `STUDIO_MODULE_PATH`, `STUDIO_MODULE_SIG` | set in backend env only when frozen and `studio_app` is a `.pyd` with a `.sig` beside it (entrypoint.py:74-82); workers ed25519-verify sha256(module) (cc common/license_guard.cpp:272-275, integrity.cpp:72-84) | GO PORT: the module is now the Go exe; key/sig pipeline must change or check is skipped |
| `STUDIO_LICENSE_CERT_PATH` | set when license enforcement is on (frozen): `%LOCALAPPDATA%/ScadaBench/license.cert` (licensing/service.py:61-62); worker default identical (license_guard.cpp:256-260); `.time_anchor` read from same folder (:280-282) | |
| `LOCALAPPDATA`, `SystemDrive`, `SystemRoot` | read by workers (license_guard.cpp:147,258; go main.go timer) | |
| creation flags | `CREATE_NO_WINDOW` on nt (worker_process.py:17) | |
| pipes | stdin/stdout/stderr all PIPE, `bufsize=0` (worker_process.py:18-20) | |
| ms timer | studio_ied `timeBeginPeriod(1)` (cc worker/main.cpp:15-19); studio_goose no longer raises it (event-driven uplink, change 0129) | not backend's job |

License gate: ops `start` (ied), `goose_start`/`sv_start`/`connect`/`identify` (live), `operate`
(control), `start` (gateway), `connect` (master), whole process (probe) call
`license::require_valid_license()`; errors surface as `ok:false` with e.g. "No license on this
computer. Activate Studio first." / "License check failed: ..." / "ScadaBench cannot run while a
debugger is attached" (license_guard.cpp:278,290; integrity.cpp:45). Backend separately hashes
shipped workers against `EXPECTED` (licensing/integrity.py:35-46, empty in source).

## 4. Framing (identical for ied/live/control/gateway/master/goose)

| Aspect | Python | Native C++ | Go engine |
|---|---|---|---|
| Header | 4-byte little-endian uint32 length (worker_protocol.py:19,29) | manual LE (worker/ipc/framing.cpp:39-40,52-54) | `binary.LittleEndian` (go internal/ipc/framing.go:31,65) |
| Max | `0 < len <= 16 MiB`, read error `ValueError("Invalid native frame length N")` (worker_protocol.py:5,20-21); write >16 MiB `ValueError("Native request exceeds frame limit")` (:27-28) | 0 or >16 MiB -> exception "Invalid frame length" -> stderr + exit 1 (framing.cpp:41; main loops) | `MaxFrame` 16 MiB, "invalid frame length" (framing.go:14,32-34) |
| Payload | UTF-8 JSON, `separators=(",",":")`, `allow_nan=False` (NaN/Inf -> ValueError before send), `ensure_ascii` default True (worker_protocol.py:26) | `dump(-1,' ',false, replace)`: raw UTF-8, invalid UTF-8 replaced; NaN dumps as `null` (framing.cpp:50) | `json.Marshal` (escapes `<>&`, NaN -> error, event lost + stderr log) (framing.go:57; main.go:19-23) |
| stdout protection | n/a | `protect_frame_output()` dups real stdout for frames and redirects fd 1 to stderr, so libiec61850 printf becomes diagnostics (framing.cpp:17-34) | none: writes `os.Stdout` directly (main.go:18); logs go to stderr (main.go:15) |
| Concurrency | writes under `self.lock` RLock (worker_process.py:60-65) | `emit_frame` mutex (framing.cpp:49) | `Writer.mu` (framing.go:66) |

### Request / response envelope

- Request: `{"version":1,"id":<uuid4 hex>,"op":<name>, ...fields}` (worker_process.py:58,65).
- Reply: `{"id","ok":true,"result":<any>}` or `{"id","ok":false,"error":<string>}`; stop reply has
  no `result` (cc worker/main.cpp:58,60,62; go server.go:25-28,31-36).
- Version check: C++ every op incl. stop: "Protocol version must be 1" (worker/main.cpp:34);
  Go checks all ops except `stop`: "protocol version must be 1" (go server.go:25-28,48-50).
- Unknown op: ied "Unknown operation"; live/control/gateway/master "Unknown operation <op>";
  go "unknown operation <op>" (go also converts panics to "internal error in <op>: ..." server.go:43-46).
- Events: any frame containing key `event` is unsolicited (worker_process.py:31). Replies matched
  by `id`; replies with unknown id (timed-out request) are dropped silently (:34-37).
- Requests are answered strictly in order by a single-threaded main loop in every native worker
  and in go `Serve` (go server.go:14-40). A long op blocks all later requests to that process.

### WorkerProcess semantics to reproduce (py runtime/worker_process.py)

| Behaviour | Detail | Line |
|---|---|---|
| Reader thread | daemon; reads frames forever; events delivered synchronously, in order, on the reader thread (a slow consumer delays replies) | 21-45 |
| Event consumer errors | logged `"Native event consumer failed"`, never kill the reader | 47-51 |
| stderr thread | each line -> event `{"event":"native_diagnostic","message":line.decode(utf-8, replace).strip()}` | 53-55 |
| Pipe failure | EOF -> "Native worker pipe closed"; OSError/ValueError text; all pending get `{"ok":false,"error":failure or "Worker stopped"}` | 38-43, worker_protocol.py:13 |
| `worker_failed` | delivered once when the reader ends and `closing` is False: `{"event":"worker_failed","error":failure}` (error may be None) | 44-45 |
| request() | default timeout 10 s; dead process -> `RuntimeError("Native worker exited: <code>")`; `ok:false` -> `RuntimeError(error or "Native request failed")`; timeout -> `TimeoutError("Native <op> request timed out")`; returns `result` (None for stop) | 57-78 |
| close() | `closing=True`; if alive: request `stop` (3 s, errors ignored) -> wait 3 s -> `terminate()` -> wait 3 s -> `kill()`+wait; join reader/stderr 1 s each; close 3 pipes | 80-99 |

HTTP mapping that depends on these exception types (app_factory.py): `ValueError`/`KeyError` -> 400,
`RuntimeError`/`TimeoutError` -> 503, `StudioError` -> 400/409. So every native `ok:false` is a
503 unless a caller converts it.

## 5. studio_ied.exe (native IED worker, ADR-0069/0073)

Missing binary: `ValueError("Native IED engine is not built yet")` (supervisor.py:37-38).
Config file: `<data>/runtime/<ied_id>.cfg` written UTF-8 from `ied["cfg"]` before spawn, path
passed absolute (`.resolve()`) (supervisor.py:45,55-56,59).

### Requests (py -> cc)

| op | Python fields (ref) | Timeout | Native handling | Result |
|---|---|---|---|---|
| start | `cfg` abs path, `ip`, `port`, `controls` (list of ied controls), `edition`, `goose:{publish,interface}` (worker_settings.py:10-21; supervisor.py:59-60) | 10 s | license; parse cfg; report buffer 1 MiB, no file/log service, max 8 MMS conns, edition 1 else 2; identity "ScadaBench"/"Simulated IED"/"1"; install controls; GOOSE prep; bind; errors "IED already started", "Native model parser rejected configuration", "Unable to construct MMS server", "MMS bind failed: ip:port" (cc worker/server/server.cpp:39-77) | `{running:true, ip, port, goose:{compiled,requested,active,control_blocks[,reason]}, capabilities:[versioned_updates, atomic_control_feedback, control_selection_events, standard_controls, goose_aware, goose_bus (+mms_access)]}` |
| update | `ref`, `value` (supervisor.py:66,72) | 10 s | single-item batch then read (worker/main.cpp:45-47) | scalar value read back |
| update_batch | `updates:[{ref,value}]` (supervisor.py:77) | 10 s | validates all first, 1..256 items else "Update batch must contain 1 to 256 attributes"; type-checked setters; "Value N is out of range", "The device model could not store this integer", "Unsupported scalar type", "Unknown data attribute: <ref>" (worker/updates/values.cpp:99-129, 45-97) | `{ref: value}` read back |
| update_guarded | `updates`, `guard:{ref,revision}` (supervisor.py:88; pulses.py:29-31) | 10 s | revision mismatch -> "Signal changed; stale pulse action cancelled" (values.cpp:110-116) | `{values:{ref:value}, revision}` |
| read | `ref` (supervisor.py:82) | 10 s | lock model, convert (values.cpp:6-33) | scalar: bool, int64, uint32, double, bitstring int (CODEDENUM big-endian), string, UTC ms; unsupported -> null |
| read_versioned | `ref` (supervisor.py:85) | 10 s | values.cpp:35-41 | `{value, revision}` |
| stop | (close) | 3 s | `server.stop()`, reply, exit 0 (worker/main.cpp:57-58) | none |

Python post-start (supervisor.py:61-67): `check_capabilities` requires the 6 caps above else
`ValueError("Native engine is outdated. Use the rebuilt portable application or rebuild the native
worker before Run")` (worker_settings.py:4-7,24-26); endpoint = reply `ip`/`port` (fallback to
request) (:62); `ied["goose_state"] = reply.goose` (:63); then one `update` per signal of kind
Dbpos/Tcmd/Quality/Timestamp with truthy value (:64-66); `ied["state"]="running"`. Any exception
in a batch launch stops only that batch (`stop_some`) and re-raises (:47-52).
Endpoints: missing `ip` assigned `127.50.0.2..254` ("No free loopback addresses")
(networking/loopback.py:5-16); preflight bind test with `SO_EXCLUSIVEADDRUSE` and messages in
networking/endpoints.py:8-39; lab mode allowed addresses from IP Helper (supervisor.py:43).
GOOSE: `goose.publish = bus and ied.goose_publish(default True) and goose.publishes`;
`interface = "udp:<enginePort>:<MAC 12 hex, no dashes, upper>"` (worker_settings.py:10-15;
goose_engine.py:165). Worker refuses non-`udp:` ids with reason "GOOSE is sent only through
the ScadaBench GOOSE engine" (cc worker/goose/publishing.cpp:24-40). Worker sends raw Ethernet
GOOSE frames as UDP datagrams to 127.0.0.1:<port> (native/hal/ethernet_udp.c, ethernet_backend.h:24).
Control entries: `ref,status,cdc(SPC|DPC),model,sbo_timeout_ms,blocked`; `compatibility` key is
rejected; feedback type must match CDC; `.t` sibling auto-stamped (cc worker/controls/controls.cpp:91-120).

### Events (cc -> py) and Python handling

| event | Fields | Source | Python (simulation/coordinator.py:67-90) |
|---|---|---|---|
| connection | connected, peer | server.cpp:16-20 | forwarded |
| report_control | name, kind(int RCB event), parameter, error(int), peer | server.cpp:21-30 | forwarded; `kind` renamed `native_kind` |
| control_requested | ref, value, trigger "Oper", peer | controls.cpp:28-29 | forwarded |
| control_applied | ref, cdc, action(close/open/set_true/set_false), ctl_value, status_ref, value(DPC 1/2 or bool), changed, timestamp(ms), trigger, peer | controls.cpp:57-61 | `external` = MmsAccess.control (peer set and not own command within 20 s, simulation/mms_access.py:7,34-41); cancel pulses; update index value/timestamp and `.t`; add `updates`; `timestamp`->`source_timestamp`; then NativeControlFeedback.enqueue |
| control_selection | ref, state(selected/canceled/timed_out/operated/failed/disconnected/unknown), selected, model, sbo_timeout_ms | controls.cpp:8-18,64-70 | forwarded |
| control_test | ref | controls.cpp:83 | forwarded |
| goose_control | gocb "LN$GO$name", enabled | publishing.cpp:9-15 | forwarded |
| goose_publishing | state starting/active | publishing.cpp:42-49 | forwarded |
| mms_access | clients:[{peer,connected,reads{LD/LN:n},total_reads,writes{obj:n},rcbs[]}] max once/1 s, 32 clients, 64 nodes ("(other)") | server/access_log.cpp:13-24,82-98; access_log.hpp:30-31 | NOT forwarded raw: MmsAccess.update merges and publishes `mms_access{ied_id,clients}` to EventBus only (no run log) (mms_access.py:43-49) |
| native_diagnostic | message | worker_process.py:55 | forwarded |
| worker_failed | error | worker_process.py:45 | cancel pulses; IED `state="failed"`; forwarded |

Forwarding rule (coordinator.py:86-88): `emit(kind, ied_id=..., **payload)` where payload drops
`event` and renames `kind`->`native_kind`, `id`->`native_id`, `timestamp`->`source_timestamp`;
`emit` also appends to the run log (coordinator.py:61-65).

### Shutdown
`Supervisor.stop` closes every worker sequentially (3+3+3 s worst case each), clears endpoints,
sets IED `state="stopped"`, drops `goose_state` (supervisor.py:90-107).
Run-start failure path: `live.stop` -> feedback clear -> scenario stop -> `supervisor.stop` ->
emit `runtime{state:"stopped",message}` -> re-raise (coordinator.py:145-156).

## Parity risks
1. Exception-type to HTTP-status mapping (native `ok:false` = 503, NaN serialisation = 400) must be
   kept; Go code returning generic errors will change status codes the UI relies on.
2. JSON number typing: Python keeps int vs float vs bool from native replies (e.g. `type(current)
   is type(expected)` in gateway/commands.py:84); decoding into Go `interface{}` makes all numbers
   float64 and loses int64 > 2^53 (UTC ms is fine, uint64 `revision` is not guaranteed). Use
   `json.Decoder.UseNumber` or typed structs.
3. Event ordering: events and replies share one reader; Python delivers events synchronously before
   later replies. A Go reader that dispatches events on goroutines changes pulse/revision races.
4. `STUDIO_MODULE_PATH`/`SIG` integrity check is tied to the Python `.pyd`; a Go exe needs a new
   signing scheme or packaged workers silently skip/refuse.
5. Close timing (stop 3 s, wait 3 s, terminate, wait 3 s, kill) and the 10 s default timeout for
   `start` (incl. license + cfg parse) are user-visible; keep them.
6. `worker_failed` must fire only when not closing and must carry the Python error strings
   ("Native worker pipe closed"), which reach the UI via events.
