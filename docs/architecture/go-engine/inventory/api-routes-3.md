# API routes 3/3: Tools place (Explorer, Find IEDs, memory, activity, gateway, SCADA master)

Same legend as api-routes-1.md. Handlers: `api/tools.py`, `api/gateway.py`, `api/gateway_master.py`.
`body()` in these three files = `get_json(silent=True)`, and a non-object -> 400 "Send a JSON object".
None of these routes requires a stopped Run (gateway/master have their own state checks).

## IED Explorer (tools/explorer.py, tools/explorer_auto.py)

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /tools/explorer | tools.py:33 | ExplorerService.warm -> status | - | `{host,interval_ms,message,polling,port,reads,state,watch[{ref,fc,type,+raw/text/status/at...}]}`; idle host/port come from tool-settings.json | - | starts a warm studio_live worker in the background (hello capability `explore`) | features/tools/explorer/ExplorerTool.tsx (polled, see frontend-callers) |
| POST /tools/explorer/connect | tools.py:38 | connect(host, port=102) | `{host, port}` | status (after auto-watch start) | "Type the device address as four numbers, for example 192.168.1.10"; "The port must be a number between 1 and 65535 (IEC 61850 uses 102)" (no bool guard: `true` = 1); 503 "The live client (studio_live.exe) is not built yet; rebuild the native programs"; 503 "Could not open an IEC 61850 association to H:P. Check the address, the cable and that the device allows another client."; 503 outdated | W `connect` (15 s, `timeout_ms` 5000); emit `explorer_connected {host,port}` (bus only); explore key nodes; writes tool-settings.json; starts polling thread | api/tools.ts |
| POST /tools/explorer/disconnect | tools.py:44 | disconnect | none | status | - | W `disconnect` (10 s), worker kept warm | api/tools.ts |
| GET /tools/explorer/devices | tools.py:49 | devices (cached) | - | `{"names":[...]}` | "Connect to a device first" | W `explore` | api/tools.ts |
| GET /tools/explorer/nodes | tools.py:54 | nodes(device) | `?device=` | `{"names":[...]}` | same | W `explore device` | api/tools.ts |
| GET /tools/explorer/node | tools.py:59 | node(ref) -> domain/explorer_tree.node_view | `?ref=LD/LN` | node view `{ref,name,...}` | "A logical node reference looks like LD/LN, for example IED1CTRL/XCBR1" | W `explore node` (60 s) | api/tools.ts |
| POST /tools/explorer/read | tools.py:64 | read(items) | `{items:[{ref,fc}]}` 1..200 | `{"values":{ref:{at,raw,status?,text,...}}}` | "Send items as a list of {ref, fc}"; "Read between 1 and 200 attributes at a time"; per-ref "device refused the read (access error N)" / "read failed (MMS error N)" | W `read` | api/tools.ts |
| PUT /tools/explorer/watch | tools.py:69 | set_watch(items, interval_ms) | `{items, interval_ms?}` | status | "The watch list holds at most 200 attributes"; "Choose a poll interval between 0.5 and 60 seconds" (500..60000 int) | writes tool-settings.json (watch_by_host) | api/tools.ts |
| POST /tools/explorer/polling | tools.py:75 | start_polling / stop_polling | `{on:bool}` | status | "Add attributes to the watch list first"; "Connect to a device first" | thread reads every interval_ms (min 50 ms gap) | api/tools.ts |
| GET /tools/explorer/watch.csv | tools.py:81 | status | - | CSV `reference,fc,value,status,read_at_unix` (CRLF) | - | none | api/tools.ts (link) |
| POST /tools/explorer/operate | tools.py:94 | operate -> tools/control_client.ControlClient (studio_control.exe) | `{ref, value, options?, confirm:true}` | `{observed,result,summary}` | "Send the control reference, for example LD0/XCBR1.Pos"; "Confirm the command first"; "Connect to a device first"; control_options texts ("Originator must be bay, station, remote or maintenance control", "<flag> is on or off"); "A control value is on/off (true/false) or a whole number step"; 503 "The control program (studio_control.exe) is not built yet; rebuild the native programs" / outdated | studio_control `operate` (30 s, or_ident "ScadaBench"); emit `explorer_command` (bus only); reads `<ref>.stVal` after | api/tools.ts |

## Activity, tool memory, Find IEDs (tools/activity.py, tool_memory.py, ied_scan.py)

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /activity | tools.py:102 | tools/activity.activity | - | `{explorer:{host,polling,state,watched},gateway:{clients,port,protocol,state},master:{host,points,protocol,state},scan:{probed,state,total},sequences:n}` | - | never starts anything | api/activity.ts via app/useActivity.ts (2000 ms) |
| GET /tools/memory | tools.py:110 | ToolMemory.get | - | `{explorer:{host,interval_ms,port,watch_by_host},find:{choice,port,range}}` | - | none | api/tools.ts |
| PUT /tools/memory/find | tools.py:115 | ToolMemory.update("find", choice) | `{choice:str<=200}` | the `find` section | "Send the chosen network as text" | writes tool-settings.json | api/tools.ts |
| POST /tools/scan/preview | tools.py:123 | ScanService.preview -> domain/scan_range.plan_range | `{range, port=102}` | `{count,first,last,port,range}` | "Type a network such as 192.168.1.0/24, one address, or a range such as 192.168.1.20-40"; "Scan at most 256 addresses (a /24 network) so the scan stays quick and polite"; "The range must go from the lower to the higher address"; "'<text>' is not a valid address, range or network"; "Multicast and 0.0.0.0 addresses cannot be scanned"; port text | none | api/tools.ts |
| POST /tools/scan/start | tools.py:129 | ScanService.start | `{range, port}` | scan status | above + 503 live client missing; "A scan is already running; stop it first" | writes tool-settings.json `find.range/port`; 64 parallel TCP connects (0.5 s); then studio_live `identify` per open host (10 s, timeout_ms 3000) | api/tools.ts |
| POST /tools/scan/stop | tools.py:135 | ScanService.stop | none | status | - | joins thread (15 s) | api/tools.ts |
| GET /tools/scan | tools.py:140 | ScanService.status | - | `{found[{connect_ms?,devices,identified,ip,model,port,reason,revision,tcp_ms,vendor}],message,port,probed,range,seconds,started,state,total}`; idle shows remembered range/port | - | none | features/tools/find/FindTool.tsx (300 ms) |

Scan messages: "Looking for IEDs on N address(es)", "Asking N device(s) who they are",
"Stopped. N device(s) found so far.", "Done. N address(es) answer on port P, M identified as IEC 61850.",
"Done. No device answered on this port.", "The scan stopped: <error>"; entry `reason` "Identifying…" (U+2026).

## Station gateway simulator (gateway/manager.py, domain/gateway_*.py)

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /tools/gateway | gateway.py:28 | GatewayManager.status | - | `{auto_start,available,clients[sorted peers],listen,message,opened({bind,port,...}|null),problems,protocol,real_commands,role,run_state,state,traffic_last}` | - | none | api/gateway.ts; features/tools/gateway/GatewayTool.tsx (1500 ms) |
| GET /tools/gateway/config | gateway.py:33 | config | - | `{auto_start,listen,protocol,real_commands,settings}` | - | none | api/gateway.ts |
| PUT /tools/gateway/config | gateway.py:40 | domain/gateway_rules.listen_address/validate_settings; ProjectService.gateway | `{protocol, listen="this_pc", settings, real_commands?}` | same as GET config | "Stop the gateway before changing its settings"; "Choose IEC 104 (iec104), Modbus TCP (modbus) or DNP3 (dnp3)"; listen/settings texts in gateway_rules.py; "Commands to real IEDs are on or off" | save; changed(source "gateway") | api/gateway.ts |
| GET /tools/gateway/points | gateway.py:59 | GatewayManager.table | `?protocol=` | `{"points":[point + ied_name,ied_role,monitor_only,(value_text,online)]}`; converter role -> mapping rows | protocol text | none | features/tools/gateway/PointTable.tsx (1000 ms) |
| PUT /tools/gateway/points | gateway.py:64 | gateway_rules.validate_points | `{protocol, points}` | `{"points": table}` | "Stop the gateway before changing its point list"; 400 `{"error":"Some points need attention before they can be saved","problems":[...]}`; "A gateway holds at most 5000 points" | save; changed("gateway") | api/gateway.ts |
| POST /tools/gateway/propose | gateway.py:86 | domain/gateway_points.propose | `{protocol}` | `{notes,points[+ied_name,monitor_only]}` | protocol text | none | api/gateway.ts |
| GET /tools/gateway/points.csv | gateway.py:92 | domain/gateway_csv.export_csv | `?protocol=` | CSV columns `type,address,ied,reference,scale,description` | protocol text | none | api/gateway.ts (link) |
| POST /tools/gateway/points.csv | gateway.py:100 | gateway_csv.import_csv + validate_points | `{protocol, text}` | `{points,problems}` (nothing saved) | protocol text | none | api/gateway.ts |
| POST /tools/gateway/start | gateway.py:110 | GatewayManager.start | none | status | "The gateway is already running; stop it first"; 503 "The gateway program (studio_gateway.exe) is not built yet; rebuild the native programs"; "The tool is set to SCADA master; switch it to Station gateway first"; "The point list is empty. Press 'Propose from IEDs' or import a CSV point list first."; converter texts; 503 outdated | auto-proposes points when empty (save); spawns studio_gateway (`hello`, `start` 15 s); sync thread every 0.1 s (`update` batches of 500); sets `auto_start:true` (save); emit `gateway_started {protocol,port,points}` | api/gateway.ts |
| POST /tools/gateway/stop | gateway.py:115 | GatewayManager.stop(remember=True) | none | status | - | `auto_start:false` (save); `halt` (10 s); emit `gateway_stopped` | api/gateway.ts |
| POST /tools/gateway/command | gateway.py:120 | GatewayManager.send -> CommandExecutor.run | `{point_id:str, value:bool, confirm:true}` | `{"ok":bool,"reason":str}` (never raises) | "Confirm the command first"; "Send point_id and value (true = ON / close, false = OFF / open)" | studio_control `operate` (originator station); command log; emit `gateway_command` | api/gateway.ts |
| GET /tools/gateway/traffic | gateway.py:131 | NumberedLog(1000).after(after) | `?after=` int | `{dropped,entries[{at,dir,peer,seq,text}] (last 300),last}` | int() text | none | features/tools/gateway/TrafficLog.tsx (500 ms running / 2000 ms) |
| GET /tools/gateway/commands | gateway.py:136 | NumberedLog(200).after(after, limit=200) | `?after=` | `{dropped,entries[{applied,at,label,observed,peer,point_id,protocol,request,seq}],last}` | int() text | none | CommandLog.tsx (1000 ms) |

## Role switch and SCADA master / converter (gateway/master_manager.py, domain/master_rules.py)

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| POST /tools/gateway/role | gateway_master.py:31 | master_rules.validate_role | `{role:"gateway"|"master"|"converter"}` | `{"role":...}` | role text; "Stop the gateway and disconnect the master before changing this" | save; changed("gateway") | api/master.ts |
| GET /tools/master | gateway_master.py:40 | MasterManager.status | - | `{auto_connect,available,host,last_interrogation,link_open,message,points,poll_failures,port,probe,protocol,ranges,state,traffic_last}` | - | none | MasterTool.tsx / ConverterTool.tsx (1500 ms) |
| GET /tools/master/config | gateway_master.py:45 | config | - | `{auto_connect,host,modbus_ranges,protocol,role,routes,settings}` | - | none | api/master.ts |
| PUT /tools/master/config | gateway_master.py:52 | validate_host/settings/ranges; save | `{protocol?, host?, settings?, modbus_ranges?}` | config | "Disconnect before changing the connection settings"; master_rules texts ("Choose IEC 104 (iec104), DNP3 (dnp3) or Modbus TCP (modbus)", host, ranges) | save; changed("gateway") | api/master.ts |
| POST /tools/master/connect | gateway_master.py:66 | connect | none | status | "Already connected; press Disconnect first"; "Switch the tool to SCADA master first"; 503 master exe missing; "Type the address of the gateway, RTU or outstation first"; 503 "ScadaBench cannot act as <NAME> master in this build yet"; native connect errors | spawns studio_master (`connect`, timeout t0+10 s); `auto_connect:true` (save); emit `master_connected {protocol,host,port}`; GI/integrity/poll; keeper thread | api/master.ts |
| POST /tools/master/disconnect | gateway_master.py:71 | disconnect(remember=True) | none | status | - | `auto_connect:false`; `stop` (10 s); emit `master_disconnected` | api/master.ts |
| POST /tools/master/interrogate | gateway_master.py:76 | interrogate | none | GI answer `{confirmed,text,...}` or Modbus `{confirmed:true,text:"read N values"}` | "Not connected to the device; press Connect first"; 503 timeouts (60 s / 30 s) | W `interrogate`/`poll` | api/master.ts |
| GET /tools/master/points | gateway_master.py:81 | points -> domain/master_points.rows | - | `{"points":[rows]}` | - | none | MasterPointTable.tsx (1000 ms), converter/ServedPoints.tsx |
| PUT /tools/master/selection | gateway_master.py:86 | save unticked | `{unticked:[str<=40]}` | `{"unticked":[first 20000]}` | "Send the unticked point keys as a list" | save; changed | api/master.ts |
| PUT /tools/master/label | gateway_master.py:99 | save labels | `{key<=40, label}` | `{key,label}` (label trimmed, <=80; empty removes) | "Send the point key and its name" | save; changed | api/master.ts |
| GET /tools/master/points.csv | gateway_master.py:115 | master_points.export_csv | - | CSV | - | none | api/master.ts (link) |
| POST /tools/master/labels.csv | gateway_master.py:122 | master_points.import_labels | `{text}` | `{imported,problems}` | - | save; changed | api/master.ts |
| POST /tools/master/command | gateway_master.py:131 | MasterManager.command | `{kind,address,value,select,word_order?,code?,confirm:true}` | `{ok,steps,text,...}` (RuntimeError -> `{ok:false,steps:[],text}` 200) | "Confirm the command first"; "This command type is not available for P: choose one of a, b"; "The address must be a whole number" (bool guarded; 0..16777215 iec104 else 0..65535); "The value must be a number"; "A CROB is latch, pulse or trip_close"; not connected | W `command` (40 s); master command log | api/master.ts |
| PUT /tools/master/ranges | gateway_master.py:161 | validate_ranges | `{modbus_ranges}` | `{"modbus_ranges":[...]}` | "Send at most 20 Modbus ranges" and per-range texts | save; changed | api/master.ts |
| PUT /tools/master/routes | gateway_master.py:170 | validate_routes | `{routes}` | `{"routes":[...]}` | "Send at most 500 command routes as a list" ... | save; changed | api/master.ts |
| POST /tools/master/probe | gateway_master.py:178 | validate_probe; ModbusProbe.start | `{table,start,end}` | probe snapshot `{address,end,found,message,requests,start,state,table}` | "Find registers is already running"; master_rules probe texts (<=10000 addresses) | W `probe` (timeout 1500 s) in background | api/master.ts |
| POST /tools/master/probe/ranges | gateway_master.py:185 | ranges_from_probe + validate_ranges | `{data_type="uint16"}` | `{"modbus_ranges":[...]}` | range texts | save; changed | api/master.ts |
| GET /tools/master/traffic | gateway_master.py:196 | NumberedLog(1000) | `?after=` | `{dropped,entries,last}` | int() text | none | features/tools/gateway (traffic view) |
| GET /tools/master/commands | gateway_master.py:201 | NumberedLog(200) | `?after=` | `{dropped,entries[{address,at,kind,select,seq,state,text,value,who}],last}` | int() text | none | MasterCommandLog.tsx (1000 ms) |

## Background (no route) behaviour tied to these tools
- App start: thread `gateway-resume` waits for startup recovery, then `master.autoconnect()`
  then `gateway.autostart()` (app_factory.py:74,114-119); failures set state `failed` with
  "Automatic connect failed: ..." / "Automatic start failed: ...".
- App close: entrypoint.py:189 only calls `runtime.close()` (IED Run); nothing calls
  `stop(remember=False)`/`disconnect(remember=False)`. Gateway, master, Explorer and control
  workers end when the backend exits (their stdin pipe closes; see processes-*.md).
  `auto_start`/`auto_connect` therefore stay true and resume on the next launch.
- Master keeper: re-asks every `gi_minutes`/`integrity_minutes`, Modbus every `poll_ms`;
  lost link retried every 10 s; texts "<reason>. Trying again every 10 s.", "Connected again",
  "Still trying to reach the device: ...".
- Gateway commands observe the status point for 3 s: "<ref> is now ON/OFF/open (OFF)/closed (ON)...",
  "<ref> did not change within 3 s".

## Parity risks
- NumberedLog `at` is `time.time()` float; `seq` restarts per process; `after` semantics + last-300 cap.
- Explorer/scan timings (300 ms UI poll, 0.5 s connect timeout, 64 parallel) are user-visible.
- Many messages embed Python formatting (`{seconds:g}`, `round(x,1)`), e.g. "3 s", `tcp_ms` 1 decimal.
- Gateway/master child processes outliving the backend on abrupt exit (Windows job objects not used).
