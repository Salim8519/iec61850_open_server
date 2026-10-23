# Frontend -> backend callers (part 2 of 2)

Continues `frontend-callers-1.md` (same legend; paths relative to `/api/v1`; `FE` = `studio/frontend/src`).
Unless noted, errors surface as `error.message` (= backend `error` text) in a local error line;
"swallowed" = `.catch(() => {})`.

## 4. GOOSE, SV, MMS, network, app-level

| # | Method + path | Callers (FE file:line) | Poll (ms) | Body sent / fields read | Errors |
|---|---|---|---|---|---|
| 39 | GET `/goose` | network/GooseTraffic.tsx:19 | 2000 while running; refetch on revision | `running,state,message,npcap{installed,download,message},engine?{running,uplink,share,message,conflicts},watches[]{key,gocb_ref,app_id,mac,go_id,dataset_ref,conf_rev,state,problem,frames,st_num?,sq_num?,age_ms?,origin?,src_mac?,publisher?,subscribers[]},unknown[]{gocb_ref,app_id,mac,dataset}` (api/network.ts:14-30) | inline |
| 40 | GET `/goose/links` | shared poll (api/polls.ts:7): devices/useGooseLinks.ts:9, diagram/signals/useSignalLayers.ts:48 (only when an SLD layer is on) | 5000 while running, else once; re-subscribed on every project revision | `ieds[]{name,ied_id,role,in_project,remote,desc,manufacturer}, blocks[]{publisher,ld_inst,name,gocb_ref,go_id,dataset,dataset_ref,conf_rev,app_id,mac,status('sendable' checked),label,members,member_refs}, links[]{id,gocb_ref,publisher,subscriber,block,status,remote,match,mac?,inputs[],live{state,problem?,st_num?,observed_at?}\|null}, diagnostics[]{level,code,ied,gocb_ref,message,hint}, summary:Record<iedName,{sends_to,gets_from[],not_sent,remote,not_heard,problems}>, running` (api/signalLayers.ts:4-30) | swallowed |
| 41 | GET `/goose/share` | network/GooseReach.tsx:21; network/GooseConflicts.tsx:19 (on each new `goose_conflict` SSE event while running) | 2000 while `share.running`; refetch on revision | `reach,uplink{guid,label,kind,connected,auto,why,reason,warning?},uplink_guid,choices[],risk{level,reasons[]},npcap(bool),running,state,message,conflicts[]{ied_id,gocb_ref,src_mac,sharing,choice}` | swallowed |
| 42 | PUT `/goose/share` | GooseReach.tsx:31 | - | `{reach:'inside'}` / `{reach:'outside',adapter_guid,confirm:true}` / `{reach:'inside',uplink_guid}`; reads GooseShare | banner (perform) |
| 43 | POST `/goose/conflicts` | GooseConflicts.tsx:25 | - | `{ied_id, choice:'stop'\|'continue', confirm: choice==='continue'}` | swallowed |
| 44 | GET `/sv` | network/SvTraffic.tsx:29 | 1000 while running; refetch on revision | `running,state,message,nominal_hz,watches[]{key,publisher,sv_id,app_id,mac,smp_rate,smp_mod,expected_rate,layout,state,problems[],received_rate?,received_lost?,received_smp_synch?(index 0-2),received_values?[]{name,rms,magnitude,angle}(.toFixed => must be numbers),received_raw?,observed_at?},unknown[],not_supervised[]` | swallowed |
| 45 | GET `/mms/access` | devices/useGooseLinks.ts:19 (`useMmsClients`); useSignalLayers.ts:51 (once when readers layer on) | 3000 while running (useMmsClients) | `running, ieds: Record<iedId, MmsClient[]>`; MmsClient `{peer,ip,own,this_pc,reading,reads[]{node,count},more_nodes,total_reads,writes,rcbs[],connected_s}` | swallowed |
| 46 | GET `/network` | settings/SettingsPage.tsx:20 | - | reads only `npcap{installed,download,message}` | swallowed |
| 47 | PUT `/network` | settings/NetworkSettings.tsx:28 `{mode:'loopback'}`; network/SvTraffic.tsx:40 `{nominal_hz:Number}` | - | partial body; response ignored | banner |
| 48 | GET `/network/adapters[?refresh=1]` | network/NetworkSetup.tsx:28; tools/find/FindTool.tsx:25 | 3000 with `?refresh=1` until an adapter is chosen | `adapters[]{guid,name,description,status('Up' compared),index,speed,kind,default_route,addresses[]{ip,prefix,skip_as_source,origin}}` | inline |
| 49 | POST `/network/plan/preview` | NetworkSetup.tsx:44 (500 ms debounce) | - | `{adapter_guid, strategy:'scd', manual: Record<iedId,{ip,prefix}>}`; reads `adapter,rows[]{ied_id,name,scd_ip,proposed,prefix,status,hint,alternative?},items[]{kind,action,owned,description,ip?,name?},fingerprint,ready,needs_admin` | inline |
| 50 | POST `/network/plan/apply` | NetworkSetup.tsx:54 | - | `{fingerprint, confirm:true}`; reads `applied?,message,results[]{ok,message,description}` | inline |
| 51 | GET `/network/cleanup/preview` | network/CleanupCard.tsx:19 | - | `items[] (PlanItem), fingerprint` | null on error |
| 52 | POST `/network/cleanup/apply` | CleanupCard.tsx:35 | - | `{fingerprint, confirm:true}`; reads `message` | inline |
| 53 | GET `/network/gateway-firewall` | tools/gateway/GatewayFirewall.tsx:21 | refetch on port change | `items[],fingerprint,open,blocked` | inline |
| 54 | POST `/network/gateway-firewall` | GatewayFirewall.tsx:27 | - | `{fingerprint, confirm:true}`; reads ApplyResult | inline |
| 55 | GET `/preferences` | app/App.tsx:35; state/useListFilter.ts:6 (memoised once per page); useSignalLayers.ts:37 | - | `tour_seen_version, tour_version, list_filters: Record<list,Record<key,string\|bool>>, sld_goose_layer?, sld_readers_layer?` (0/1) | swallowed |
| 56 | PUT `/preferences` | App.tsx:42 `{tour_seen_version}`; useListFilter.ts:31 `{list_filters:{[list]:values}}` (400 ms debounce, only one list per call: server must MERGE); useSignalLayers.ts:43 `{sld_goose_layer:0\|1, sld_readers_layer:0\|1}`; app/installFit.ts + settings/DisplaySection.tsx `{display_size:0\|1}` (Go only, ADR-0111) | - | partial bodies, server merges; response ignored | swallowed |
| 57 | GET `/activity` | app/useActivity.ts:12 | 2000 once a project is loaded | `gateway{state,protocol,port,clients}, explorer{state,host,polling,watched}, scan{state,probed,total}, master{state,protocol,host,points}, sequences` | swallowed |
| 58 | GET `/app/update` | api/updates.ts:17 (App.tsx:52) | 1 800 000 (30 min) | `current,auto,checked_at(epoch s\|null),checking,error('offline'\|'no_server'\|'no_release'\|'server_error'\|'bad_answer'\|null),available,important,dismissed,latest{version,date,notes,page,min_version}\|null` | swallowed |
| 59 | PUT `/app/update` | updates.ts:24 | - | `{auto}` or `{dismissed: version\|null}`; reads UpdateStatus | swallowed (App) |
| 60 | POST `/app/update/check` | updates.ts:25 | - | no body; reads UpdateStatus | caller |
| 61 | GET `/license/status` | license/LicenseBoundary.tsx:16 (boot + on `studio-license-required`), settings/LicenseSection.tsx:13 | - | `licensed,mode,plan,days_remaining,expires_at(epoch s),reason,license_id,email,enforced,server_url,masked_key,hwid,transfer_available,app_version` | boot: any failure => app shown anyway |
| 62 | POST `/license/trial` | license/LicenseGate.tsx:23 | - | `{}`; reads LicenseStatus | `LicenseError.message` |
| 63 | POST `/license/activate` | license/LicenseActions.tsx:29 | - | `{license_key, move}`; reads LicenseStatus | `code==='bound_elsewhere'` => "move the key here" UI (LicenseActions.tsx:40) |
| 64 | POST `/license/activate-offline` | LicenseActions.tsx:83 | - | `{certificate}` | message |
| 65 | PUT `/license/config` | LicenseActions.tsx:69 | - | `{server_url}` | message |

## 5. Tools API (`/tools/*`)

| # | Method + path | Callers | Poll (ms) | Body / fields read |
|---|---|---|---|---|
| 66 | GET `/tools/memory` | tools/find/FindTool.tsx:25 | - | `explorer{host,port,interval_ms}, find{range,port,choice}` |
| 67 | PUT `/tools/memory/find` | FindTool.tsx:51 | - | `{choice}` (swallowed) |
| 68 | GET `/tools/explorer` | explorer/ExplorerTool.tsx:45, :55; WatchList.tsx:29 | while connected: `polling ? max(500, min(interval_ms,2000)) : 4000` | ExplorerStatus `state,message,host,port,watch[]{ref,fc,type,text,status,at},interval_ms,polling,reads` |
| 69 | POST `/tools/explorer/connect` | ExplorerTool.tsx:37 | - | `{host,port}` -> ExplorerStatus |
| 70 | POST `/tools/explorer/disconnect` | ExplorerTool.tsx:72 | - | `{}` -> ExplorerStatus |
| 71 | GET `/tools/explorer/devices` | explorer/ModelTree.tsx:19 | - | `names[]` |
| 72 | GET `/tools/explorer/nodes?device=` | ModelTree.tsx:27 | - | `names[]` |
| 73 | GET `/tools/explorer/node?ref=` | explorer/NodeDetail.tsx:22 | - | `ref,name,objects[]{name,fcs[],attributes[]{ref,fc,path,type}},leaf_count,datasets[]{name,ref,members},reports[]{name,ref,buffered}` |
| 74 | POST `/tools/explorer/read` | NodeDetail.tsx:26 (max 200 items), WatchList.tsx:28 | - | `{items[{ref,fc}]}` -> `values: Record<key,{raw,text,status,at}>` |
| 75 | PUT `/tools/explorer/watch` | ExplorerTool.tsx:61, WatchList.tsx:25 | - | `{items[{ref,fc,type?}], interval_ms?}` (interval_ms may be omitted => undefined dropped by JSON) -> ExplorerStatus |
| 76 | POST `/tools/explorer/operate` | explorer/CommandPanel.tsx:20 | - | `{ref,value,confirm:true}` -> `summary, observed(ReadValue\|null), result{ok,model?,add_cause_text?,error?}` |
| 77 | POST `/tools/explorer/polling` | WatchList.tsx:41 | - | `{on}` -> ExplorerStatus |
| 78 | GET `/tools/scan` | FindTool.tsx:33, :45 | 300 while `state==='scanning'` | `state,range,port,total,probed,found[]{ip,port,tcp_ms,identified,vendor,model,revision,devices[],reason,connect_ms?},message,seconds` |
| 79 | POST `/tools/scan/preview` | FindTool.tsx:37 (250 ms debounce) | - | `{range,port}` -> `range,count,first,last,port` |
| 80 | POST `/tools/scan/start` / `/tools/scan/stop` | FindTool.tsx:48 / :68 | - | `{range,port}` / `{}` -> ScanStatus |
| 81 | GET `/tools/gateway` | gateway/GatewayTool.tsx:24; converter/ConverterTool.tsx:31 | 1500 | `state,message,protocol,listen,opened,clients[],problems[],run_state,available,traffic_last,real_commands,auto_start,role?` |
| 82 | GET / PUT `/tools/gateway/config` | GET GatewayTool.tsx:26, ConverterTool.tsx:31 (1500 poll); PUT GatewayTool.tsx:36, ConverterTool.tsx:52 | 1500 (converter) | PUT `{protocol,listen,settings,real_commands}`; reads `protocol,listen,settings:Record<proto,Record<k,num\|bool>>,real_commands,auto_start` |
| 83 | GET `/tools/gateway/points?protocol=` | gateway/PointTable.tsx:31, converter/ServedPoints.tsx:11 | 1000 | `points[]{id?,kind,address,ied,ref,scale,description,ied_name?,ied_role?,monitor_only?,value_text?,online?}` |
| 84 | PUT `/tools/gateway/points` | PointTable.tsx:39 | - | `{protocol,points}`; on error reads `error.data.problems[]` (api/gateway.ts:20, PointTable.tsx:41) |
| 85 | POST `/tools/gateway/propose` | PointTable.tsx:42 | - | `{protocol}` -> `points[], notes[]` |
| 86 | POST `/tools/gateway/points.csv` | PointTable.tsx:44 | - | `{protocol, text}` (CSV as JSON string, not multipart) -> `points[], problems[]` |
| 87 | POST `/tools/gateway/start` / `/stop` | GatewayTool.tsx:42-43, ConverterTool.tsx:53 | - | `{}` -> GatewayStatus |
| 88 | POST `/tools/gateway/command` | PointTable.tsx:49 | - | `{point_id,value,confirm:true}` -> `ok, reason` |
| 89 | GET `/tools/gateway/traffic?after=N` | gateway/TrafficLog.tsx:28 | 500 when running else 2000 | `entries[]{seq,at,dir('rx'\|'tx'\|'info'),peer,text}, last, dropped`; FE sets `after=last` only when entries non-empty |
| 90 | GET `/tools/gateway/commands` | gateway/CommandLog.tsx:20 | 1000 | `entries[]{seq,at,label,protocol('manual'=page),peer,request,applied,observed}` (text parsed, see parity list) |
| 91 | POST `/tools/gateway/role` | ScadaTool.tsx:20 | - | `{role}` -> `role` |
| 92 | GET `/tools/master` | master/MasterTool.tsx:25; ConverterTool.tsx:31 | 1500 | `state,message,link_open,protocol,host,port,points,last_interrogation,available,traffic_last,auto_connect,probe{state,table?,start?,end?,requests?,address?,found?[],message?},poll_failures[]{range,text},ranges` |
| 93 | GET / PUT `/tools/master/config` | GET MasterTool.tsx:27, ScadaTool.tsx:16, CommandRoutes.tsx:11, ModbusRanges.tsx:16, ConverterTool.tsx:31 (1500 poll); PUT MasterTool.tsx:37, ConverterTool.tsx:48 | 1500 (converter) | PUT `{protocol,host,settings}`; reads `role,protocol,host,auto_connect,settings,modbus_ranges[],routes[]` |
| 94 | POST `/tools/master/connect` / `/disconnect` / `/interrogate` | MasterTool.tsx:46-47, ConverterTool.tsx:49-50 | - | `{}`; interrogate -> `confirmed,text` |
| 95 | GET `/tools/master/points` | master/MasterPointTable.tsx:27 | 1000 | `points[]{key,group,group_name,type,address,value,value_text,quality_text,good,time_ms,cause,changes,label,ticked,stale,command}` |
| 96 | PUT `/tools/master/selection` / `/label` | MasterPointTable.tsx:36 / :40 | - | `{unticked[]}` / `{key,label}` |
| 97 | POST `/tools/master/labels.csv` | MasterPointTable.tsx:43 | - | `{text}` -> `imported, problems[]` |
| 98 | POST `/tools/master/command` | master/MasterCommandForm.tsx:30 | - | `{kind,address?,value?,select?,code?,confirm:true}` -> `ok,text,terminated?` |
| 99 | PUT `/tools/master/ranges` / `/routes` | ModbusRanges.tsx:31 / CommandRoutes.tsx:22 | - | `{modbus_ranges}` / `{routes}` echoed back |
| 100 | POST `/tools/master/probe` / `/probe/ranges` | master/FindRegisters.tsx:31 / :20 | - | `{table,start,end}` -> ProbeState / `{data_type}` -> `modbus_ranges` |
| 101 | GET `/tools/master/traffic?after=N` | TrafficLog via MasterTool.tsx:55, ConverterTool.tsx:64 | 500 / 2000 | same Page shape as #89 |
| 102 | GET `/tools/master/commands` | master/MasterCommandLog.tsx:18 | 1000 | `entries[]{seq,at,who,kind,address,value,select,state('sent'\|'confirmed'\|'refused'),text}` |

## 6. Browser-navigated downloads and static assets (no custom headers, cookie only)

| Path | FE file:line | Notes |
|---|---|---|
| GET `/api/v1/project/download` | app/TopBar.tsx:12-17 | `<a download="">` click: filename comes from server `Content-Disposition` |
| GET `/api/v1/ieds/{id}/source` | devices/DeviceFiles.tsx:13 | plain link (no `download` attr): server must send `attachment` or the browser navigates |
| GET `/api/v1/live/{id}/comparison.csv` | devices/ComparisonView.tsx:44 | plain link |
| GET `/api/v1/health.csv`, `/api/v1/health/report.html` | devices/HealthView.tsx:40-41 | plain links; the HTML report opens in the same window |
| GET `/api/v1/signal-flow/edges/{id}.csv?source=` | devices/SignalFlowView.tsx:84 | plain link |
| GET `/api/v1/tools/explorer/watch.csv` | api/tools.ts:38, explorer/WatchList.tsx:43 | `download` attr |
| GET `/api/v1/tools/gateway/points.csv?protocol=` | api/gateway.ts:34, gateway/PointTable.tsx:62 | `download` attr |
| GET `/api/v1/tools/master/points.csv` | api/master.ts:39, master/MasterPointTable.tsx:59 | `download` attr |
| GET `/` (index.html), `/assets/index-<hash>.js`, `/assets/index-<hash>.css` | frontend/index.html, vite build (`dist/`) | Vite base `/`; no client router (single page, state in memory; `pathname` only reused in the 403 redirect) |
| GET `/assets/brand/favicon.svg`, `favicon-32.png`, `icon-180.png` | index.html:7-9, app/Sidebar.tsx:29 | from `frontend/public` |
| GET `/api/v1/events` | state/useStudio.ts:78 | `EventSource`; see `frontend-events.md` |

Dev only: `vite.config.ts` proxies `/api` to `http://127.0.0.1:8765` (strictPort). In dev the page origin
differs from the API origin; `X-Studio-Frontend` is how the backend recognises the active dev tab.
External (not backend): `https://scadabench.com/tutorials/` (window.open), `status.server_url + '/buy'` and
`'/activate'` links (api/license.ts:45-52), `latest.page` (UpdateNote.tsx:12), Npcap URL.

## 7. Polling summary (24 polled paths)

| Interval | Paths (condition) |
|---|---|
| 300 | `/tools/scan` (scanning) |
| 500 / 2000 | `/tools/gateway/traffic`, `/tools/master/traffic` (500 if link running) |
| 500-2000 / 4000 | `/tools/explorer` (polling ? clamp(interval_ms) : 4000) |
| 1000 | `/sv`, `/project/test-panel` (running); `/scenarios/runs`, `/tools/gateway/points`, `/tools/gateway/commands`, `/tools/master/points`, `/tools/master/commands` (while mounted) |
| 1500 | `/tools/gateway`, `/tools/master`; converter also `/tools/gateway/config`, `/tools/master/config` |
| 2000 | `/activity` (always after project load), `/live` (real IED, running), `/goose` (running), `/goose/share` (share.running) |
| 3000 | `/health` (shared, running), `/mms/access` (running), `/network/adapters?refresh=1` (no adapter chosen) |
| 5000 | `/goose/links` (shared, running) |
| 1 800 000 | `/app/update` |
Only `/health` and `/goose/links` use the non-overlapping shared poller; all other polls are plain
`setInterval` that can overlap if the backend is slower than the interval (requests pile up).
Debounced (not polled): diagram save 400, list-filter save 400, plan preview 500, scan preview 250,
signal searches 150/200.
