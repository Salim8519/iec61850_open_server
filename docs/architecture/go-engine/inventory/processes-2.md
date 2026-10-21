# Process & IPC parity checklist (2/3): studio_live.exe, studio_goose.exe, studio_probe.exe

Framing, envelope, env, cwd, CREATE_NO_WINDOW and WorkerProcess semantics: see processes-1.md
sections 3-4 (all apply unchanged here).

## 1. studio_live.exe (read-only MMS client + GOOSE/SV listener)

Path: `settings.live_executable` = `native_executable.with_name("studio_live.exe")`
(settings.py:15-17; coordinator.py:56). One process per role; each is an independent
WorkerProcess. Native main is single-threaded: `discover` (up to 120 s) blocks every other
request of that process (cc live/main.cpp:44-65).

### Ops (cc live/main.cpp:22-43; stop at :51-58 = goose.stop, sv.stop, release all RCBs, close)

| op | Request fields | Native result / errors | Ref |
|---|---|---|---|
| hello | - | `{capabilities:[live_discover, live_reports, live_poll, goose_listen, goose_inbox, sv_listen, read_only_client, explore, identify]}` | main.cpp:23-27 |
| connect | ip, port(102), timeout_ms(5000), request_timeout_ms(5000) | `{connected:true, ip, port}`; "Already connected"; "Could not connect to ip:port (MMS error N)" | live/mms_session.cpp:17-35 |
| discover | - | `browse_server` result `{ok:true, devices:[{name,nodes:[...]}]}`; "Not connected"; "Discovery failed" | mms_session.cpp:37-43; common/browse.cpp:67-80 |
| read | refs:[{ref,fc}] (max 2000 "At most 2000 references per read") | `{values:[{ref,value} / {ref,structure:{flatref:v}} / {ref,error:int} / {ref,error:"access_error",access_error:int}]}`; stops after NOT_CONNECTED / CONNECTION_LOST | mms_session.cpp:45-76 |
| explore | device? / node? | `{names:[...]}` or `browse_node` `{name,variables,leaves,objects,datasets,dataset_members,buffered_reports,unbuffered_reports}` | live/explore.cpp; browse.cpp:42-56 |
| identify | ip, port(102), timeout_ms(3000) | own temporary association; `{ip,port,identified:false,reason}` or `{ip,port,identified:true,connect_ms,vendor,model,revision,devices[],identity_note?,devices_note?}` | live/identify.cpp:15-47 |
| rcb_status | rcbs:[ref] | `{blocks:[{rcb,readable,buffered,enabled,reserved,owned,dataset,conf_rev,free} / {rcb,readable:false,error,free:false}]}` | live/reports.cpp:37-60 |
| rcb_reserve | rcb | `{reserved:false,status}` when not free, or `{reserved:true,rcb,members}`; sets Resv (URCB), RptEna, GI; "Report block already in use by Studio", "Could not read report dataset X", "The IED refused to enable report block X" | reports.cpp:62-114 |
| disconnect | - | releases all RCBs (RptEna false, Resv false), closes; `true` | main.cpp:42 |
| goose_start | subscriptions:[{key,gocb_ref,app_id,mac "XX-XX-..",go_id,dataset_ref,conf_rev}] | opens loopback UDP inbox; `{listening:true, inbox_port, subscriptions:n}`; "GOOSE listener already running", "Invalid GOOSE MAC address X", inbox errors | live/goose_traffic.cpp:53-96 |
| sv_start | adapter_guid, streams:[{key,mac,app_id,layout(9-2LE/raw),per_cycle,wrap,channels,scale}] max 16 | `{listening:true,streams:n}`; "Npcap is not installed, so Sampled Values cannot be received", "At most 16 ...", "Could not open the network adapter for Sampled Values", "The selected network adapter is not visible to Npcap" | live/sv_traffic.cpp:30-67; sv_traffic.hpp:38; common/npcap.cpp:84 |

### Events

| event | Fields | Ref |
|---|---|---|
| live_connection | state closed/connecting/connected/closing/unknown | mms_session.cpp:8-13 |
| live_report | rcb, values{flatref:v}, reasons{ref:int}, has_timestamp, source_timestamp | reports.cpp:15-32 |
| goose_rx (change) | key, gocb_ref, origin wire/app, src_mac, st_num, sq_num, conf_rev, dataset, go_id, test, tal_ms, source_timestamp, state receiving/problem, problem (invalid_frame, conf_rev_mismatch, dataset_mismatch, go_id_mismatch, needs_commission, ""), frames, values(tree, only if valid) | goose_traffic.cpp:98-130 |
| goose_rx (timeout) | key, gocb_ref, state/problem "timeout", st_num, sq_num, frames; when now > last_rx + TAL + 50 ms, checked every 250 ms | goose_traffic.cpp:157-170 |
| goose_stats | watches:[{key,frames,sq_num,age_ms}] every 8 ticks (~2 s) | goose_traffic.cpp:171-175 |
| goose_unknown | gocb_ref, app_id, mac, src_mac, origin, dataset, go_id, conf_rev, st_num, test, vlan_id/null (once per gocb, 500 max) | goose_traffic.cpp:132-155 |
| goose_unknown_cap | limit 500 | goose_traffic.cpp:138; goose_traffic.hpp:51 |
| sv_rx | key, state, asdus, rate, sv_id, dataset, conf_rev, smp_rate, smp_mod, smp_synch, lost, restarts, invalid_quality, data_size, values[{name,rms,magnitude,angle}], raw[], simulate; every 500 ms per watch | sv_traffic.cpp:104-122,137-145 |
| sv_unknown / sv_unknown_cap | app_id, mac, src_mac, sv_id, simulate, vlan_id / limit 500 | sv_traffic.cpp:146-158 |

GOOSE inbox wire contract (cc live/goose_inbox.cpp:15-59): UDP bound to 127.0.0.1 ephemeral
port, `SIO_UDP_CONNRESET` off, 4 MiB rcvbuf, select 50 ms; datagram byte 0 = origin
(1 = wire, else app), rest = raw Ethernet frame (>= 2 bytes). Fed to `GooseReceiver_handleMessage`;
the receiver is never started (no adapter opened by studio_live for GOOSE).

### Python consumers

| Role | File:line | Requests (timeout) | Event handling | User-facing failures |
|---|---|---|---|---|
| GOOSE monitor | acquisition/goose_monitor.py:20-41 | `engine.ensure()` first; `goose_start` (30 s); require `inbox_port` else ValueError "The GOOSE listener is outdated (no goose_inbox). Rebuild studio_live.exe."; then engine `deliver(port)` | goose_rx: update watch fields + `received_{conf_rev,go_id,dataset,test}` + `observed_at`, emit `goose_status` (event minus `values`), route to LGOS supervision and real-publisher live values, GooseInputs.apply (:43-60, 80-97); goose_stats updates frames/age_ms/sq_num (:61-68); goose_unknown appended + emit `goose_unknown{gocb_ref,app_id}` (:69-72); cap stored (:73-75); worker_failed -> state failed "The GOOSE listener stopped unexpectedly", emit `goose_listener_failed` (:76-78) | start errors caught in LiveManager: status failed "GOOSE listener could not start: <e>", emit `goose_listener_failed` (manager.py:52-58) |
| SV monitor | acquisition/sv_monitor.py:20-43 | only in lab mode with adapter and Npcap DLL present; `sv_start` (30 s) | sv_rx: `assess` verdict, store `received_*`, emit `sv_status{key,state,problems}` only on change (:45-59); sv_unknown(_cap) stored; worker_failed -> failed "The Sampled Values listener stopped unexpectedly" (:66-67) | caught in manager.py:59-63: "Sampled Values listener could not start: <e>", emit `sv_listener_failed` |
| Real IED session | acquisition/real_session.py:47-157 | `hello` (10 s) require {live_discover,live_reports,live_poll,read_only_client} else "The live client is outdated; rebuild the native programs"; `connect` (15 s, ip/port only); `discover` (120 s); `rcb_status` (30 s) when access auto_reports; `rcb_reserve` (30 s) each; health `read` (30 s) (health_checks.py:13); loop `read` (30 s, refs[:2000]) every `poll_ms`; mode re-read every 10 s; `disconnect` (15 s) on close | live_report -> LiveValues.apply(source "live_report"); live_connection closed or worker_failed while not stopping -> lost (:37-44) | states emitted as `live_status{ied_id,state,message}`, IED state `real_<state>`; retry backoff 2,4,..30 s "<error>. Trying again in N s." (:47-64); "This IED has no address in the SCD..." (:50); lost -> mark_stale + "Connection to the device was lost"; read error entries -> `live_read_error` event; stop joins 20 s, message "Disconnected; every report block ScadaBench used was given back" (:153-157) |
| IED Explorer | tools/explorer.py | warm spawn on page open (:47-57); `hello` require `explore` else "The live client is outdated; rebuild the native programs"; `connect` (15 s, timeout_ms 5000); `explore` (30 s; node 60 s); `read` (30 s, max 200); `hello` before polling; `disconnect` (10 s, on failure close process) | live_connection closed while connected -> state lost "The device closed the connection. Press Connect to try again." (:104-107) | connect failure: "Could not open an IEC 61850 association to h:p. Check the address, the cable and that the device allows another client." (:92-93); emits `explorer_connected`, `explorer_command` |
| Find IEDs scan | tools/ied_scan.py:87-101 | TCP probe in-process (64 threads, 0.5 s) first; then one worker, `identify` (10 s, timeout_ms 3000) per open host; closed after | none (no on_event) | per-host exception -> `{identified:false, reason:str(e)}` |

LiveManager (acquisition/manager.py): missing exe -> `ValueError("The live client
(studio_live.exe) is not built yet; rebuild the native programs")` when any real IED or any
GOOSE/SV publisher/subscriber exists (:47-50, 67-68). Stop order: goose monitor, engine, SV,
sessions (:79-85). Watched points capped at 500 (:29).

## 2. studio_goose.exe (Go GOOSE engine, ADR-0073)

Spawn: `GooseEngine.ensure()` once per Run (acquisition/goose_engine.py:45-58), triggered by
`Supervisor.add -> goose_bus -> bus_for` (coordinator.py:44-45) or GooseMonitor.start. Missing:
`ValueError("The GOOSE engine (studio_goose.exe) is not built yet; run studio/tools/build_goose_engine.py")`;
caps missing `goose_engine_v1`: close + `ValueError("The GOOSE engine is outdated. Rebuild it with
studio/tools/build_goose_engine.py before Run")` (:10-11, 49-51). Python serialises calls with
`calls` RLock and never holds `lock` while waiting (deadlock note :26-29).

| op | Request | Result | Go ref | Py ref (timeout 10 s) |
|---|---|---|---|---|
| hello | - | `{version:"1", capabilities:["goose_engine_v1"]}` | engine/ops.go:10-19 | goose_engine.py:53 |
| open | ieds:[{id, gocb_refs[]}] (first call `[]`) | `{port, sources:{ied_id:"02-PP-PP-NN-NN-NN"}}`; bus opened on first call; "every IED needs an id" | ops.go:20-28; engine.go:59-85 | :58, :67 (bus_for builds `udp:PORT:MAC12`) |
| deliver | port | `true`; "open the engine before setting the inbox", "inbox port is out of range" | ops.go:29-37; engine.go:88-96; bus/bus.go:38-44 | :75 |
| uplink | adapter_guid ("" = none), share bool | `{adapter_guid, uplink:closed/opening/open/lost/error/needs_npcap/send_failed, message, share:off/checking/sharing}` | ops.go:38-47; engine/share.go:23-59,115-125 | :83 (merged into `link`) |
| gate | ied_id, choice stop/continue | `{ied_id, gate:open/held/blocked/forced}` or error | ops.go:48-58; guard/guard.go:15-21,124-136 | :92; StudioError wrap in goose_share.py:90-93 |
| stats | - | `{bus_frames,bus_rejected,delivered_app,no_inbox,wire_frames,delivered_wire,sent,refused{reason:n},gates,sources,link}` | engine/stats.go:30-50 | :100 (`delivered_wire` used by goose_share.py:28) |
| stop | - | Close (stop tick, uplink off, bus close) then reply, exit 0; EOF also closes | ipc/server.go:14-40; engine.go:168-177 | close() |

Events (Go -> py) and handling (goose_engine.py:102-125):

| event | Fields | Go ref | Python |
|---|---|---|---|
| goose_conflict | ied_id, gocb_ref, src_mac, sharing | engine.go:142-146; share.go:50-53 | store conflict (keep previous `choice`), emit `goose_conflict` |
| goose_share | state checking/off/sharing | share.go:49,56,103 | link.share=state, emit `goose_share` |
| goose_uplink | adapter_guid, state, message | share.go:68-80 | link.uplink/message, emit `goose_uplink{state,message}` |
| goose_storm | ied_id, limit_per_second (1000), max 1 per IED per 5 s | engine.go:156-165 | emit `goose_storm` |
| worker_failed (py) | error | - | link = closed + uplink "failed", message "The GOOSE engine stopped unexpectedly"; emit `goose_listener_failed` |
| native_diagnostic (py) | Go `log` lines on stderr ("event lost: ..", errors) | main.go:15,21 | ignored |

Bus wire contract: Go binds UDP 127.0.0.1:0, 4 MiB buffers (bus/bus.go:24-32); worker datagram =
raw Ethernet frame; forwarded to live inbox as `[origin byte 0=app|1=wire] + frame`
(bus.go:11-15,47-57). Source MAC per IED: `02:<2 random bytes per run>:<24-bit index>`
(engine.go:59-80). Sharing starts only 3 s after the uplink is open (share.go:11, 83-106).
Rate cap 1000 frames/s/IED outside (engine.go:19). Share decision and hold logic is in
Python (`goose_share.py:67-76`: listen = uplink guid only if Npcap DLL installed; share =
reach "outside" and no hold). `apply` requires `confirm:true`, Npcap, uplink guid and the
same adapter the user saw (messages at goose_share.py:9-10, 52-61).

## 3. studio_probe.exe (one-shot independent MMS client)

| Aspect | Value | Ref |
|---|---|---|
| argv | `IP PORT browse` / `read REF [FC=ST]` / `control REF true|false` / `reports REF` / `sbo-* REF [true] [n] [x]` | cc probe/main.cpp:24-81 |
| Used by backend | `control <ref> true|false` (runtime/mms_client.py:19-21) from POST /controls/operate (api/signals.py:59-66) and equipment buttons (simulation/equipment_control.py:68); `read <ref> MX` in `--self-test` (launcher/self_test.py:30-36) | |
| Flags | `capture_output=True, text=True` (locale decode), `CREATE_NO_WINDOW` on nt, timeout 15 s (control) / 10 s (self-test) | mms_client.py:19-21; self_test.py:31-34 |
| stdout | single JSON line `output.dump()`; NOT protected from libiec61850 printf (no `protect_frame_output`) | probe/main.cpp:84 |
| exit codes | 0 ok, 1 `ok:false` or connect fail (`{"ok":false,"error":int}`), 2 usage, 4 license failure (text on stderr) | main.cpp:24-40,85 |
| control semantics | Boolean ctlVal only; SBO normal -> select, SBO enhanced -> selectWithValue; operate; enhanced waits <= 100x20 ms for CommandTermination; `{"ok":bool}` | main.cpp:54-74 |

Python preconditions (mms_client.py:9-17): value must be bool ("Control value must be boolean"),
ref in `ied["controls"]` ("Unknown control object"), IED running ("IED must be running to
operate equipment") -> ValueError (400); missing exe `RuntimeError("Bundled MMS client is
unavailable")`; non-zero exit `RuntimeError("MMS control failed or command termination was not
received from host:port")` (503); timeout -> `subprocess.TimeoutExpired` (unhandled type ->
generic 500 unless caught); result = `json.loads(stdout)`. Before calling, the API marks the
command as ScadaBench's own for 20 s (`mms_access.mark_own`) so the resulting `control_applied`
is not flagged external (api/signals.py:62; simulation/mms_access.py:7,30-41). Endpoint =
`supervisor.endpoint(ied_id)` (worker-confirmed ip/port).

## Parity risks
1. studio_live single-threaded loop: a real-IED `discover` (120 s) or `read` blocks that process;
   Go must keep one process per role (not share one) or behaviour/timeouts change.
2. Timeouts are per call and user-visible: connect 15, discover 120, rcb 30, read 30, explore
   30/60, identify 10, goose_start/sv_start 30, engine ops 10, probe 15 s.
3. GOOSE startup order must be preserved: engine `ensure` -> live `goose_start` -> engine
   `deliver(inbox_port)` -> `uplink`; `open` returns MACs that become the worker `udp:` interface.
4. studio_probe parsing: `text=True` decodes stdout with the ANSI code page, and stray HAL printf
   on stdout would break `json.loads`; `TimeoutExpired` currently escapes as an unmapped
   exception (HTTP 500). Decide whether Go reproduces or fixes, and record it.
5. Event payload trimming (`goose_status` without `values`, `goose_unknown` reduced to 2 fields,
   `sv_status` only on verdict change) feeds the EventBus/run log; field sets must match exactly.
6. Engine `worker_failed` maps to `goose_listener_failed` (not `goose_engine_failed`); the UI and
   logs depend on that kind name.
