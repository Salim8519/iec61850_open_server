# Process & IPC parity checklist (3/3): studio_control, studio_gateway, studio_master, network helper, PowerShell, OS contracts

Framing/envelope/env/WorkerProcess semantics: processes-1.md sections 3-4.

## 1. studio_control.exe (warm control client, ADR-0057)

Owner: `ControlClient(native_executable.with_name("studio_control.exe"))` (app_factory.py:52),
shared by IED Explorer commands and gateway SCADA commands. One process, no event callback
(diagnostics and `worker_failed` are dropped). Respawned when `process.poll()` is not None.
All calls serialised by `self.lock`, so a 30 s command blocks the next one (tools/control_client.py:40-56).

| op | Request | Result / errors | Ref |
|---|---|---|---|
| hello | - | `{capabilities:[control_operate, control_models_all]}`; py requires `control_operate` else close + RuntimeError "The control program is outdated; rebuild the native programs" | cc control/main.cpp:14; py control_client.py:45-47 |
| operate | ip, port, ref, value (bool or int32), or_ident "ScadaBench", or_cat (bay 1/station 2/remote 3/maintenance 7, default remote), interlock_check (default true), synchro_check (false), test (false); optional termination_ms (5000, py never sends) | `{ref, model(status-only/direct-with-normal-security/sbo-with-normal-security/direct-with-enhanced-security/sbo-with-enhanced-security/unknown), selected, operated, terminated(null/bool), add_cause, add_cause_text, ok, error?, client_error?}`; refusals are `ok:false` results, not errors | cc control/control_session.cpp:71-133; control_texts.cpp:4-25; py :13-31, :51-56 (timeout 30 s) |
| stop | - | close all associations, reply, exit | cc control/main.cpp:25-28 |

Native details: associations cached per `ip:port`, reused while CONNECTED, connect timeout
5000 ms, request timeout 10000 ms (control_session.cpp:34-52); hard errors (thrown -> `ok:false`
envelope -> RuntimeError): "Could not connect to ip:port (MMS error N)", "<ref> is not a
controllable object on this device", "This control expects a value type ScadaBench cannot send yet
(only on/off and integer steps)". Python validation errors (ValueError, 400): "Originator must be
bay, station, remote or maintenance control", "<key> is on or off", "A control value is on/off
(true/false) or a whole number step". Missing exe: RuntimeError "The control program
(studio_control.exe) is not built yet; rebuild the native programs". `summary()` sentence:
"The device <selected and operated|operated> it (<model>)." else `error` or "The device refused
the command." (control_client.py:65-70).

## 2. studio_gateway.exe (station gateway simulator: IEC 104 / Modbus TCP / DNP3 server)

Owner: gateway/manager.py. Missing: RuntimeError "The gateway program (studio_gateway.exe) is not
built yet; rebuild the native programs" (:78-79). Required caps {gateway_iec104, gateway_modbus,
gateway_dnp3, commands_via_backend} else "The gateway program is outdated; rebuild the native
programs" (:11, 96-97). Spawn failure closes the process and re-raises (:99-101).

| op | Request (py ref) | Timeout | Native (cc gateway/) | Result |
|---|---|---|---|---|
| hello | - (:96) | 10 s | main.cpp:23-25 | capabilities |
| start | protocol iec104/modbus/dnp3, settings (validated + `bind`=listen_address), points:[{id,kind,address,scale,monitor_only}] (:91-98) | 15 s | license; "The gateway is already running", "Unknown gateway protocol"; point table: kinds single/double/float/integer/command_single/command_double, "Unknown point kind X", "Two points use address N (table)" (point_table.cpp:6-33); listen errors "Could not listen on ip:port..." | iec104 `{protocol,bind,port,common_address,...}` (server_104.cpp:26-55; t1,t2,t3,k,w,queue,max_connections), modbus `{protocol,bind,port,unit_id}` (modbus_server.cpp:44-64), dnp3 `{protocol,bind,port,outstation_address,...}` (dnp3_outstation.cpp:114-154; unsolicited) |
| update | values:[{id,value bool/number,quality int,time_ms,source_ok}] in chunks of 500, only changed samples, every 0.1 s (:10,131-154) | 10 s | unknown ids skipped; null value throws (get<double>) -> whole call fails (point_table.cpp:35-53) | `{changed:n}` |
| command_result | token, ok, reason (gateway/commands.py:32) | 10 s, errors ignored | wakes the protocol thread (command_bridge.cpp:19-26) | true |
| halt | - (:173) | 10 s | stop server, keep process | true |
| stop | (close) | 3 s | stop server, exit | - |

Events: `gw_traffic{dir,peer,text,skipped_before?}` rate-limited 200/s (traffic.cpp:15-36,
traffic.hpp:6) -> NumberedLog (1000 lines) with " (N lines skipped)" suffix (:117-119);
`gw_client{state open/closed,peer}` -> `clients` (:120-124); `gw_command{token,point_id,value,
protocol,peer}` -> CommandExecutor queue (:125-126); `worker_failed` -> state failed "The gateway
program stopped unexpectedly. Press Start again." (:127-129).

Command round trip (must keep timing): protocol thread emits `gw_command`, blocks up to 15000 ms
(command_bridge.cpp:9-14, command_bridge.hpp:19) -> "no answer from ScadaBench within the command
timeout"; backend executor (single thread, FIFO) runs refusal checks, then
`ControlClient.operate(..., originator "remote" ("station" for manual page commands))` (30 s),
sends `command_result`, observes the status for 3 s in another thread (gateway/commands.py:14,
27-89). Refusal texts at commands.py:106-125. Emits `gateway_command`, `gateway_started`,
`gateway_stopped` events. `stop()`: if `halt` raises, `close()` still runs but the exception
propagates and state is not reset (:162-179). `auto_start` persisted on start/stop (:110-111,164-165).

## 3. studio_master.exe (ScadaBench as SCADA master, ADR-0061)

Owner: gateway/master_manager.py. Missing: RuntimeError "The master program (studio_master.exe)
is not built yet; rebuild the native programs" (:57-58). Cap per protocol master_iec104/
master_dnp3/master_modbus else RuntimeError "ScadaBench cannot act as <IEC 104|DNP3|Modbus TCP>
master in this build yet" (:18-19, 66-67). Native loop single-threaded: `probe` (up to 1500 s)
blocks every other request; Python skips polling while probe runs (:137-138).

| op | Request | Py timeout | Native result / errors |
|---|---|---|---|
| hello | - | 10 s | caps (master/main.cpp:24) |
| connect | protocol, settings {..validated, host} (:68-69,156-157) | `t0`(default 10)+10 s | license; "Already connected; disconnect first", "This protocol is not available as master yet"; 104 `{host,port,common_address}` (client_104.cpp:34-60: port 2404, common_address, time_tagged_commands, k 12, w 8, t0 10, t1 15, t2 10, t3 20, originator_address); dnp3 `{host,port,outstation_address}` (client_dnp3.cpp:68-97: master_address 1, outstation_address 10, unsolicited, event_seconds 5); modbus `{host,port,unit_id}` (client_modbus.cpp:19-26: timeout_ms 3000) |
| interrogate | - (:110) | 60 s | 104 `{confirmed,counters,...}`; dnp3 `{confirmed:true,text:"sent (integrity poll)"}`; modbus throws "Modbus has no interrogation..." |
| poll | ranges (:120) | 30 s | modbus `{values:n, failed:[...]}` |
| probe | table, start, end (modbus_probe.py:41) | 1500 s | `{found, requests}`; "Find registers checks 1 to 10000 addresses from 0 to 65535" |
| command | kind, address, value, select?, code?, word_order? (api/gateway_master.py:132-158) | 40 s | `{ok, steps:[{step,ok,text}], terminated?}` |
| disconnect | - (:155) | 10 s | true |
| stop | - (:219) | 10 s, then close() | disconnect + exit |

Events (master/value_events.cpp:4-14; client_modbus.cpp:119): `master_values{values}` -> merge into
table under `table_lock`; `master_link{state connecting/open/closed, reason}` -> open sets
link_open, closed (reason != "Disconnected by you") -> state "lost", message "<reason>. Trying
again every 10 s." (:168-173); `master_probe{address,requests}` -> probe progress;
`master_interrogation{state finished/failed: ...}` -> traffic "All values received" / "Asking for
all values <state>"; `gw_traffic` -> traffic log; `worker_failed` -> "The master program stopped
unexpectedly. Press Connect again." (:163-186). Keeper thread: every 1 s (modbus: poll_ms),
reconnect after 10 s when lost via `disconnect`+`connect` on the same process, GI/integrity every
`gi_minutes`/`integrity_minutes` (:125-161). Command RuntimeError -> logged refused, returned
`{ok:false,steps:[],text}` (:202-205).

## 4. Elevated network helper (lab network + gateway firewall)

| Step | Contract | Ref |
|---|---|---|
| Request file | `<data>/network/requests/<action>-<uuid hex>.json`, bytes = `json.dumps({"action","fingerprint","items"})` (default separators, ensure_ascii); items minus `description` | networking/service.py:149-155 |
| Launch | `ShellExecuteExW` verb `runas`, `SEE_MASK_NOCLOSEPROCESS`, `SW_HIDE`, no working dir; params via `subprocess.list2cmdline` | elevation.py:36-47 |
| Command | frozen: `sys.executable --network-helper <request> <sha256 hex> <program>`; dev: `sys.executable <backend>/studio_app/networking/helper/run.py <request> <sha256> <program>` (run.py adds backend to sys.path because elevation resets env) | elevation.py:27-33; entrypoint.py:21-23; helper/run.py |
| program arg | `native_executable` (studio_ied.exe path); gateway rules derive `studio_gateway.exe` beside it | service.py:31-32; helper/validation.py:58-61 |
| Wait | `WaitForSingleObject` 120000 ms else RuntimeError "The network helper did not finish within two minutes"; exit code read but ignored | elevation.py:52-59 |
| Cancel | error 1223 -> ElevationCancelled "Windows administrator permission was not given, so nothing was changed" -> StudioError; other -> "Windows could not start the network helper (error N)" | elevation.py:44-47; service.py:158-159 |
| Validation | sha256(file bytes) == arg else "Request file changed after ScadaBench prepared it"; action apply/remove; 1..64 items; each `owned`; kinds alias (GUID regex, index > 0, prefix 8..30, IPv4 not loopback/multicast/link-local/unspecified), firewall (name `SubstationStudio-MMS-<12hex>`, group "Substation Studio", ports 1..65535, addresses, program == worker), gateway_firewall (`ScadaBench-Gateway-<12hex>`, group "ScadaBench", program == studio_gateway.exe), unblock (Windows rule-name regex, program == gateway) | helper/validation.py:9-100 |
| Execution | per item: `powershell.exe -NoProfile -NonInteractive -Command "$ErrorActionPreference='Stop'; <cmd> | Out-Null"`, CREATE_NO_WINDOW, 60 s; message = last line of stderr or stdout, cut to 500 | helper/runner.py:12-27 |
| Commands | New/Remove-NetIPAddress (ActiveStore, SkipAsSource), New/Remove-NetFirewallRule, Disable/Enable-NetFirewallRule with live re-check scripts; PowerShell strings single-quoted with `''` escaping | change_plan.py:11-126 |
| Result | `<request stem>.result.json` = `{"refused":bool,"reason"?, "results":[{key,ok,message}]}`; exit 0 / 2 refused | runner.py:30-40 |
| Backend after | no result file -> "The network helper finished without a result; nothing was recorded as changed"; refused -> "The network helper refused the request: <reason>"; journal `applied/removed/failed` per item to `<data>/network/journal.jsonl` (json sort_keys) | service.py:160-172; journal.py:20-29 |

Naming/fingerprints that must be byte-identical (existing machines already have these rules):
MMS rule `SubstationStudio-MMS-` + sha256(`json.dumps([sorted(addresses), ports, str(program)],
sort_keys=True)`)[:12]; plan fingerprint sha256(`json.dumps({"adapter","items"}, sort_keys=True)`)
(change_plan.py:27-36); gateway rule `ScadaBench-Gateway-` + sha256(`str(program).lower()`)[:12],
gateway fingerprint (:39-52); cleanup fingerprint sha256(json.dumps(items, sort_keys=True))
(service.py:198-201); item keys `alias:<guid>:<ip>`, `unblock:<name>`, `firewall:<name>`
(journal.py:7-12). Python default `json.dumps` uses `", "`/`": "` separators and `\uXXXX` escapes.

## 5. Non-elevated PowerShell and Win32 queries

| Use | Command / API | Timeout | Error to user | Ref |
|---|---|---|---|---|
| Adapter list (primary) | IP Helper `GetAdaptersAddresses(AF_INET, 0x8E)` + `GetUnicastIpAddressTable`; hide loopback(24)/tunnel(131)/"wi-fi direct virtual adapter"/"wan miniport"/"kernel debug"; status 1 Up, 2/7 Disconnected, else Down; speed `{:g} Gbps/Mbps/Kbps`; MAC `XX-XX` | - | falls back on OSError only | ip_helper.py:12-15,68-185; adapters.py:80-86 |
| Adapter list (fallback) | `powershell.exe -NoProfile -NonInteractive -Command <SCRIPT>` (Get-NetAdapter/Get-NetIPAddress/Get-NetRoute, ConvertTo-Json -Depth 5 -Compress) | 20 s | "Windows could not list network adapters: <last stderr line>" | adapters.py:10-35 |
| Normalise | guid upper with braces; kind from description/name keywords; sort by kind order, not Up, default_route, name | - | - | adapters.py:38-77 |
| MMS rules present | `Get-NetFirewallRule -Name 'SubstationStudio-MMS-*' ... Name` | 20 s | as above | adapters.py:89-95 |
| Gateway rules | `Get-NetFirewallRule -Name 'ScadaBench-Gateway-*'` | 20 s | "Windows Firewall could not be read: <line>" -> StudioError "Windows Firewall could not be checked: <e>" | firewall_rules.py:22-25; gateway_firewall.py:23-26 |
| Blocking rules | enabled inbound Block rules whose program -eq gateway path, TCP/Any, JSON | 60 s | same | firewall_rules.py:28-38 |
| Cleanup state | one script: gateway rule names + which given rules are disabled, JSON | 60 s | same | firewall_rules.py:41-52 |
| Address in use | `SendARP` per candidate, 16 threads | - | - | arp_probe.py:9-26 |
| Lab addresses | `addresses_on(guid)` via GetAdaptersAddresses | - | - | ip_helper.py:177-185; supervisor.py:43 |
| Npcap present | file `%SystemRoot%\System32\Npcap\wpcap.dll` exists (no process) | - | message texts | npcap.py:9-18 |

All PowerShell calls: `capture_output=True, text=True` (locale ANSI decode), `CREATE_NO_WINDOW`
on nt; non-zero exit -> RuntimeError. Adapter list cached 10 s (service.py:39-44).

## 6. Launcher OS contracts (packaged app)

| Contract | Detail | Ref |
|---|---|---|
| App window | first of Edge (x86, x64), Chrome (x64, x86, LOCALAPPDATA); argv `--app=<url>`, `--user-data-dir=<data>/browser-profile[-2|-3]` (else temp `ScadaBench-browser-*`), `--no-first-run --no-default-browser-check --window-size=1440,900`; `CREATE_NEW_PROCESS_GROUP`, stdout/stderr DEVNULL; fallback `webbrowser.open` | launcher/app_window.py:16-79 |
| Window close = app exit | wait process; if exited < 15 s, wait 5 s for profile `lockfile` hand-off, then poll 0.5 s until lock gone; then `interrupt_main` | app_window.py:24-26,82-112; entrypoint.py:128-131 |
| URL | `http://127.0.0.1:<port>/#session=<token>` + `&window=app` when a browser was found; port 8765 else ephemeral (dev: hard fail) | entrypoint.py:130,159-176 |
| launcher.json | `<data>/launcher.json` `{url, window_pid?}` | entrypoint.py:98-100,133 |
| Single instance | lock byte 0 of `<data>/studio.lock` (msvcrt LK_NBLCK); symlink refused; messages | launcher/directory_lease.py:10-37 |
| Show request | named auto-reset event `Local\ScadaBench-Show-` + sha256(normcase(abspath(data)))[:24]; second launch focuses `window_pid` or `SetEvent` + `AllowSetForegroundWindow(-1)` | launcher/single_instance.py:17-90; entrypoint.py:103-120 |
| Self-test | `--self-test` prints JSON from `check()`; uses studio_probe `read` and loopback `127.54.0.254` | entrypoint.py:52-55; self_test.py:39-117 |
| Dev | `--dev` refused when frozen; werkzeug reloader watches the native exe; token from `STUDIO_DEV_TOKEN`, Vite from `STUDIO_FRONTEND_URL` | entrypoint.py:31-42,57-60 |
| Update check | `SCADABENCH_UPDATE_CHECK=off` disables | updates/service.py:146 |

## Parity risks
1. Elevated helper is Python code today (`--network-helper` in the same exe). The Go exe must
   accept the same argv, validation, PowerShell text and result-file schema, or old request files
   and UAC prompts break.
2. Rule names and fingerprints hash Python `json.dumps` output (spaces after `,`/`:`, sort_keys,
   ASCII escapes) and `str(Path)`; a Go encoder differs byte-for-byte and would orphan existing
   firewall rules and journal entries.
3. Gateway SCADA command latency: native waits 15 s for `command_result`, Python's control call
   can take 30 s; serialised single executor + single control worker. Keep the same queueing.
4. Single-threaded native loops (master `probe` 1500 s, live `discover` 120 s, control lock)
   define which UI actions block; a Go caller must not add concurrency that the worker serialises
   anyway or change the timeouts.
5. PowerShell/probe output is decoded with Python's locale (`text=True`); Go `exec` returns raw
   bytes, so non-ASCII adapter/rule names may differ unless decoded the same way.
6. Launcher contracts (named event name, lock file byte, launcher.json, browser profile names,
   portable worker folder hash) must be identical so old and new builds coexist on one data folder.
