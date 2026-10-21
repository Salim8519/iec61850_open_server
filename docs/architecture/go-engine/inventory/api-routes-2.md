# API routes 2/3: network, real IEDs, GOOSE/SV, SLD, signal flow, health, license, prefs, updates

Same column legend as api-routes-1.md. All paths under `/api/v1`.

## Lab network and per-IED settings (api/network.py, networking/service.py)

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /network | network.py:18 | project network + networking/npcap.npcap_status | - | `{"network":{adapter_guid,adapter_label,goose_reach,goose_share_guid,goose_uplink_guid,mode,nominal_hz},"npcap":{download,installed,message,path}}` | - | none (Npcap = file check `%SystemRoot%\System32\Npcap\wpcap.dll`) | features/settings/NetworkSettings.tsx, SettingsPage.tsx, features/network/SvTraffic.tsx |
| GET /network/adapters | network.py:23 | NetworkService.adapters (cache 10 s, `refresh=1` bypasses) | `?refresh=1` | `{"adapters":[...]}` (networking/adapters.py normalised rows: guid, index, name, description, kind, addresses[{ip,prefix,...}], ...) | 503/500 if IP Helper and PowerShell fallback fail | IP Helper API call / PowerShell | features/network/NetworkSetup.tsx (3000 ms with refresh), features/tools/find/FindTool.tsx |
| PUT /network | network.py:28 | projects/ied_settings.update_network -> domain/network_settings.validate_network | `{mode, adapter_guid?, adapter_label?, nominal_hz?}` (silent JSON) | `{"network":{...}}` | stopped-only (under cycle_lock); validate_network texts ("Network mode must be 'loopback' (this PC only) or 'lab' (network adapter)", GUID, nominal Hz); endpoint conflict StudioError with `details` = all problems | save; changed. A body with only `nominal_hz` keeps mode; a mode change keeps `nominal_hz` and goose_reach keys (ied_settings.py:31-37) | NetworkSettings.tsx |
| POST /network/plan/preview | network.py:35 | NetworkService.preview | `{adapter_guid, strategy="scd"|"adapter", manual?:{ied_id:{ip,prefix}}}` | `{adapter,fingerprint,items[+description],needs_admin,ready,rows[{ied_id,name,proposed,prefix,status,hint,alternative?}]}` | stopped-only; "That network adapter is no longer available. Refresh the adapter list."; "No IED is set to 'Be this IED', so there is nothing to put on the network. Set an IED to 'Be this IED' in Devices first." | ARP probe of candidate addresses (networking/arp_probe.py); stores preview by fingerprint in memory | NetworkSetup.tsx |
| POST /network/plan/apply | network.py:42 | NetworkService.apply | `{fingerprint, confirm:true}` | `{applied,message,results[{key,ok,message,description}]}` | "Review the change list and confirm it before applying"; 409 conflict "This preview has expired. Review the changes again."; 409 conflict "The network changed since you reviewed it. Review the new change list." with `details:{fingerprint}`; "Fix the highlighted address problems before applying"; helper errors (cancelled UAC text, "The network helper finished without a result; nothing was recorded as changed", "The network helper refused the request: ...") | elevated helper (UAC) via `network/requests/apply-<hex>.json`; journal append; sets IED lab_ip + mode lab (save, changed) | NetworkSetup.tsx |
| GET /network/cleanup/preview | network.py:50 | NetworkService.cleanup_preview | - | `{fingerprint,items[+action:"remove",description],needs_admin}` | Windows read failures | reads journal, adapters, firewall rules | features/network/CleanupCard.tsx |
| POST /network/cleanup/apply | network.py:55 | NetworkService.cleanup | `{fingerprint,confirm:true}` | `{message,results}` | "Confirm the clean-up list first"; 409 "The list of ScadaBench changes is different now. Review it again." | elevated helper `remove`; journal; mode -> loopback (save, changed) | CleanupCard.tsx |
| GET /network/gateway-firewall | network.py:63 | networking/gateway_firewall.GatewayFirewall.preview | - | `{blocked,fingerprint,items,needs_admin,open}` | 400 "Windows Firewall could not be checked: <error>" | PowerShell reads; remembers fingerprint | features/tools/gateway/GatewayFirewall.tsx |
| POST /network/gateway-firewall | network.py:69 | GatewayFirewall.apply | `{fingerprint,confirm:true}` | `{applied,message,results}` messages "Done: Settings › Network › Clean up undoes this." / "Some changes failed; see each line for the reason." | "Review the Windows Firewall changes and confirm them first"; 409 "This preview has expired. Review the changes again."; 409 "Windows Firewall changed since you reviewed it. Review the new list." | elevated helper; journal | GatewayFirewall.tsx |
| PUT /ieds/{id}/settings | network.py:75 | projects/ied_settings.update_ied | subset of `{role, live, lab_ip, goose_publish, port}` (non-empty) | `{goose_publish,lab_ip,live,port,role}` | stopped-only; "Only role, live, lab_ip, goose_publish and port can be changed here"; "<IED name>: <validate_ied_settings error>"; "Port must be a whole number from 1 to 65535 (MMS normally uses 102)"; endpoint conflict with `details` | save; changed. `live` is merged into the stored live dict | DeviceConnection.tsx, DeviceDetail.tsx, DeviceOverview.tsx, network/NoPublishers.tsx |

## Real IEDs, GOOSE, SV, MMS access (api/live.py)

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /live | live.py:14 | LiveManager.status | - | `{"goose":{goose snapshot},"sessions":[status with comparison=null, health=null]}`; session status keys `access,comparison,endpoint_source,health,host,ied_id,message,name,polling,port,state` | - | none | features/devices/DeviceDetail.tsx (2000 ms) |
| GET /live/{id}/comparison | live.py:26 | session.status.comparison (domain/model_compare.compare) | - | `{datasets,reports,signals,summary}` | 400 "This IED is not connected as a real device. Set it to 'real' and press Run."; 400 "The comparison is ready after ScadaBench has connected to the device and read its model" | none | features/devices/ComparisonView.tsx |
| GET /live/{id}/comparison.csv | live.py:31 | same | - | CSV header `reference,fc,result,scd_type,device_type` (CRLF) | same; not_found | none | ComparisonView.tsx |
| POST /live/{id}/read | live.py:43 | RealSession.read (WorkerProcess `read`) | `{refs:[1..500]}` | `{"errors":{ref:msg},"values":{ref:{live_state,value}}}` (only refs known to the IED are read) | "Send between 1 and 500 references to read"; "<name> is not connected yet"; not connected text; 503 native/timeout | W read (timeout 30 s); emits `signal` (source live_poll), `live_read_error` | features/devices/LiveValueTable.tsx |
| GET /goose | live.py:58 | running: GooseMonitor.snapshot; stopped: domain/goose_supervision.watch_list | - | running: `{engine,message,npcap,running:true,state,unknown,watches,(unknown_limit)}`; stopped: `{engine,message:"Press Run to start GOOSE",npcap,running:false,state:"stopped",unknown:[],watches[state=stopped,problem="",frames=0]}` | - | none (engine `stats` request when sharing preview) | features/network/GooseTraffic.tsx (2000 ms while running) |
| GET /goose/links | live.py:69 | projects/goose_links.link_map | - | `{blocks,diagnostics,ieds,links,running,summary}` | - | parses stored SCL (cached per source set) | api/polls.ts LINK_MAP (shared poller); features/devices/useGooseLinks.ts (3000 ms) |
| GET /mms/access | live.py:76 | MmsAccess.snapshot | - | `{"ieds":{ied_id:[client rows]},"running":bool}` | - | none | api/signalLayers.ts |
| GET /sv | live.py:81 | running: SvMonitor.snapshot; stopped: sv_watch_list | - | stopped: `{message:"Press Run to start listening for Sampled Values",nominal_hz,not_supervised[{key,publisher}],npcap,running:false,state:"stopped",unknown:[],watches}` | - | none | features/network/SvTraffic.tsx (1000 ms while running) |

## Where GOOSE goes (api/goose_share.py, acquisition/goose_share.py)

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /goose/share | goose_share.py:12 | GooseShare.preview | - | `{choices,conflicts,message,npcap,reach,risk:{level,reasons},running,state,uplink:{auto,connected,guid,kind,label,reason?,why},uplink_guid}` | - | adapter listing; engine `stats` | features/network/GooseReach.tsx (2000 ms), GooseConflicts.tsx |
| PUT /goose/share | goose_share.py:17 | GooseShare.apply (allowed while running) | `{reach, uplink_guid?, adapter_guid?, confirm?}` | preview shape | 400 "Read the warning and confirm before sharing GOOSE with the network"; npcap message; uplink `reason`; "The network adapter changed. Review the warning again before sharing GOOSE."; validate_network errors | save; changed; engine `uplink` request when running | GooseReach.tsx |
| POST /goose/conflicts | goose_share.py:22 | GooseShare.choose -> engine `gate` | `{ied_id, choice:"stop"|"continue", confirm?}` | engine gate result | "Choose 'stop' or 'continue'"; "Confirm that ScadaBench may keep sending this IED's GOOSE next to the real device"; "Press Run first: GOOSE is not running"; engine errors wrapped as StudioError (400) | studio_goose `gate` | GooseConflicts.tsx |

## SLD import and signal flow

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /project/substation-sources | sld.py:9 | projects/sld_import.substation_sources | - | `{"sources":[...]}` | - | parses stored SCL | features/diagram/SldImport.tsx |
| POST /project/diagram/from-scl/preview | sld.py:14 | sld_import.preview | `{source_id, mode}` (exactly, strings) | `{already_present,diagnostics,diagram,drawn,layout,removed,revision,skipped,unbound}` | "Preview requires source_id and mode"; sld_import StudioErrors; not_found | none (apply uses PUT /project/diagram) | SldImport.tsx |
| GET /signal-flow/sources | signal_flow.py:19 | projects/signal_flow.flow_sources | - | `{"sources":[{filename,id,ieds}]}` (id = sha256(source)[:16]) | - | none | features/devices/SignalFlowView.tsx |
| GET /signal-flow | signal_flow.py:24 | signal_flow.graph | `?source=&ied=` (empty source = first) | `{blocks,diagnostics,edges,filename,ieds,source,summary}` | "Import an SCL file first"; 400 not_found "This SCL file is not in the project" | parses SCL | SignalFlowView.tsx |
| GET /signal-flow/edges/{edge}.csv | signal_flow.py:29 | signal_flow.edge_csv | `?source=` | CSV (LF line endings) `type,publisher,block,subscriber,source,input,description` | 400 not_found "Unknown signal-flow link" | none | SignalFlowView.tsx (link) |

## Station health

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /health | health.py:15 | acquisition/health_snapshot.station_health -> domain/station_health.build_health, mode_badges | - | `{app_version,checked_at(ISO, seconds, +00:00),ieds,modes:{equipment,ieds},running,simulation_sources,summary,unknown,unknown_note,verdict}` | - | none (no MMS traffic) | api/polls.ts HEALTH (shared poller), features/devices/HealthView.tsx |
| POST /health/refresh | health.py:20 | health_snapshot.refresh -> RealSession.refresh_health | none | health keys + `failures:{ied_id:msg}` (no app_version) | - | W `read`, `rcb_status` on each live session | HealthView.tsx |
| GET /health.csv | health.py:26 | domain/health_export.csv_text | - | CSV columns `ied,role,area,item,result,message,observed_at,source,stale` | - | none | HealthView.tsx (link) |
| GET /health/report.html | health.py:32 | health_export.html_report(name, "<session_id>-<revision>") | - | HTML page | - | none | HealthView.tsx (link) |

## License (api/license.py, licensing/service.py) - never gated by 402 or startup wait

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /license/status | license.py:32 | LicenseService.status (refresh=True) | - | `{app_version,days_remaining,email,enforced,expires_at,hwid(null in dev),license_id,licensed,masked_key("SUBS-****-****-****-XXXX"|null),mode,plan,reason,server_url("" in dev),transfer_available}`; dev: `licensed:true, mode:"development", plan:"dev"` | - | evaluates cert, integrity hashes, anchor advance | features/license (api/license.ts) |
| POST /license/trial | license.py:37 | start_trial -> LicenseClient (HTTPS) | none | status keys (no app_version) | `_run` mapping: LicenseServerError code in {offline,no_server,server_error} -> 503, {bound_elsewhere,transfer_limit,trial_used} -> 409, else 400, body `{code,error}`; LicenseActionError/CertificateError -> 400 `{code,error}` (code default bad_certificate); dev -> 400 code `development` "Development mode does not use licenses." | network call; writes license.cert, license_state.json, .time_anchor | api/license.ts |
| POST /license/activate | license.py:42 | activate(key, move) | `{license_key, move}` | status keys | "A license key looks like SUBS-XXXX-XXXX-XXXX-XXXX. Copy it from your purchase email." (bad_key); server texts; "<status reason>" not_valid_here | same | api/license.ts |
| POST /license/activate-offline | license.py:48 | activate_offline | `{certificate}` text | status keys | bad_certificate texts (licensing/certificate.py) | same | api/license.ts |
| PUT /license/config | license.py:53 | set_server_url | `{server_url}` | status keys | "The server address must start with https://" (bad_url); `http://127.0.0.1`/`http://localhost` allowed | writes client_config.json | api/license.ts |

## Preferences and update check

| Route | H | Svc | Req | Resp | Err | SE | FE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET /preferences | preferences.py:7 | PreferenceStore.get | - | `{list_filters,sld_goose_layer,sld_readers_layer,tour_seen_version,tour_version:1}` | - (corrupt file = defaults) | none | api/preferences.ts |
| PUT /preferences | preferences.py:12 | PreferenceStore.update | `{tour_seen_version?, sld_goose_layer?, sld_readers_layer?, list_filters?}` | same as GET | "Preferences must be a JSON object"; "Unknown preference: a, b"; "Preference X must be a whole number"; "Preference X is out of range" (sld_* 0..1, tour 0..1000); list_filters texts (projects/list_filters.py) | writes preferences.json (indent 2) | api/preferences.ts |
| GET /app/update | updates.py:11 | UpdateService.status | - | `{auto,available,checked_at,checking,current,dismissed,error,important,latest}` | - | none | api/updates.ts (every 30 min) |
| POST /app/update/check | updates.py:16 | UpdateService.check(manual) | none | status | never 5xx; error codes stored: `no_server`, `no_release`, `server_error`, `offline`, `bad_answer`; repeat within 60 s returns stored status | HTTPS GET `<server>/api/v1/release` (6 s timeout, ETag); writes update_check.json | api/updates.ts |
| PUT /app/update | updates.py:21 | UpdateService.configure | `{auto?:bool, dismissed?:version|null}` | status | 400 `{code:"invalid", error}`: "Only auto and dismissed can be changed", "auto must be true or false", "dismissed must be a version" | writes update_check.json | api/updates.ts |

## Parity risks
- Lab-network fingerprints and firewall rule names are sha256 of Python `json.dumps(sort_keys=True)`
  text (separators `", "`/`": "`): rule names are persisted in Windows and the journal.
- Adapter listing (IP Helper/PowerShell) and its sort order feed fingerprints and the UI.
- License error-status mapping and messages from the license server pass through verbatim.
- `/goose/links` and `/signal-flow` ids are sha256 prefixes of `"|".join(parts)` strings.
- `checked_at` and ISO strings use Python `isoformat(timespec="seconds")` with `+00:00`.
