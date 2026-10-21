# Frontend -> backend callers (part 1 of 2)

Source: `studio/frontend/src` (read-only survey, 2026-10-19). All paths below are relative to
`/api/v1` unless they start with `/assets` or `/`. `FE` = `studio/frontend/src`.
Part 2 (`frontend-callers-2.md`) has the tools API, downloads, static assets and the poll summary.
SSE and the parity-risk list are in `frontend-events.md`.

## 1. Transport layer (every call goes through one of four paths)

| Wrapper | File:line | What it sends | Error handling |
|---|---|---|---|
| `api<T>(path, method='GET', body?)` | FE/api/client.ts:3-26 | `fetch('/api/v1'+path)`, `credentials:'same-origin'`, header `X-Studio-Frontend: window.location.origin` on EVERY call; `Content-Type: application/json` only when body is non-FormData and truthy; FormData sent raw (browser sets multipart boundary). No body and no Content-Type when `body` is omitted (e.g. `api('/runtime/start','POST')`). No timeout, no retry, no AbortController. | `await response.json()` runs FIRST, unconditionally (non-JSON body => SyntaxError surfaces as the error text). Then: 403 + `typeof data.active_frontend==='string'` and its hostname is `127.0.0.1`/`localhost` => `location.replace(active.origin+pathname+hash)`; 402 => `window.dispatchEvent(new Event('studio-license-required'))` + throw `data.error` or "A license is required."; 404 => fixed text "The running backend does not provide this action..."; other !ok => `Error(data.error || 'Request failed (<status>)')` with `.data = <whole JSON body>` attached. 2xx => body returned as T. |
| `upload(path, file)` | FE/api/client.ts:38-42 | multipart field name **`file`**, via `api(path,'POST',form)` | as `api` |
| `download(path, body, filename)` | FE/api/download.ts:2-20 | POST JSON, `credentials:'same-origin'`, `Content-Type: application/json`, **no `X-Studio-Frontend`** header; reads `response.blob()` and saves under the FE-chosen filename | 404 => fixed text "...does not provide this export..."; other !ok => `response.json().error` or `Download failed (<status>)`. |
| license `call(path, method, body?)` | FE/api/license.ts:27-35 | `fetch('/api/v1/license'+path)`, same headers as `api` | `response.json().catch(()=>({error:'Request failed (<s>)'}))`; !ok => `LicenseError(data.error, data.code || 'error')`. Never fires the 402 event, never redirects on 403. |
| Plain `<a href>` / `link.click()` | see part 2 section 6 | browser GET with cookies only; no custom headers | none (browser download / new page) |

Session bootstrap: `connectSession()` FE/api/client.ts:28-36 reads `#session=<token>&window=app` from the URL
fragment, POSTs `{token}` to `/session`, then `history.replaceState(null,'',pathname)` (drops the fragment).
It is called twice: FE/features/license/LicenseBoundary.tsx:17 (before `/license/status`) and
FE/state/useStudio.ts:76 (before `/project`); in React StrictMode dev effects run twice, so `/session` can be
POSTed 2x concurrently with the same token -> backend must accept a token more than once. The session is a
cookie (Flask signed `session` cookie today); FE never reads it, only relies on `credentials:'same-origin'`.

## 2. Project, IEDs, runtime

Legend: Poll = interval in ms (shared = `polls.subscribe` in FE/api/sharedPoll.ts; one timer per path at
the fastest subscriber interval, no overlapping requests, last answer replayed to new subscribers).

| # | Method + path | Callers (FE file:line) | Poll | Body sent / fields read | Errors |
|---|---|---|---|---|---|
| 1 | POST `/session` | api/client.ts:33 | - | `{token}`; response ignored | throws; LicenseBoundary swallows (`.catch(()=>{})`), useStudio shows message |
| 2 | GET `/project` | state/useStudio.ts:39 (`refresh`), after every `perform()` (useStudio.ts:129), on SSE open, on `project_changed`/`runtime`/`resync`/`worker_failed`/`live_status` events, after a failed diagram save | event driven | Full `Project` (api/types.ts:54-65): `id,name,session_id,revision,ieds[],diagram{nodes,edges},runtime,network{mode,adapter_guid,adapter_label,goose_reach?},running_ieds?,scenarios[],test_panel[],sources[],startup_recovery?,migration_notice?`. Ied (types.ts:35-47) incl. `test_mode` (read via cast, TestModeCard.tsx:20), `goose_state`, `validation`, `reports[]`, `goose{publishes,subscribes,diagnostics}`, `live`, `lab_ip`, `scl_address`, `controls[]`, `warnings[]`, `signal_count`, `state`, `role`. History identity = `session_id:id:ied ids` (useStudio.ts:32). | message -> top error banner |
| 3 | PUT `/project/diagram` | state/useStudio.ts:55 (debounced 400 ms after edit, useStudio.ts:145; also flushed before save/run/import) | - | `{diagram, revision, tab_id}` (tab_id = per-tab `crypto.randomUUID()`); reads `revision` (number) | any error (incl. 409 conflict) => bump generation, `refresh()` GET /project, rethrow => "Not saved" + `data.error` banner. No status-specific branch. |
| 4 | POST `/project/save` | useStudio.ts:188 | - | no body | banner |
| 5 | POST `/project/open` | useStudio.ts:168 (`upload`, `.subsim`) | - | multipart `file`; response ignored | banner |
| 6 | POST `/scl/preview` | useStudio.ts:171 (`upload`) | - | multipart `file`; reads `available_slots`, `ieds[]{name,description,manufacturer,type,logical_devices,unavailable_reason}` (features/ieds/ImportSelection.tsx:4-8) | banner |
| 7 | POST `/scl/import` | useStudio.ts:177-180 | - | multipart: `file` + `selected_names` = JSON string array | banner |
| 8 | POST `/ieds/template` | useStudio.ts:163 | - | `{name}` | banner |
| 9 | POST `/ieds/removal-preview` | features/devices/DeleteIedsDialog.tsx:24 | - | `{ids}`; reads `ieds[]{id,name},bindings,sources,scenario_steps,scenarios_removed,test_controls,fingerprint` | dialog error text |
| 10 | POST `/ieds/remove` | DeleteIedsDialog.tsx:37 | - | `{ids, fingerprint, confirm:true}` | dialog + banner |
| 11 | PUT `/ieds/{id}/settings` | devices/DeviceConnection.tsx:42 `{live:{ip,port,access,poll_ms}}`; DeviceDetail.tsx:58 `{role}`; DeviceOverview.tsx:47 `{goose_publish}`; network/NoPublishers.tsx:12 `{role:'simulated'}` | - | partial bodies (one key each); response ignored | banner |
| 12 | PUT `/ieds/{id}/test-mode` | devices/TestModeCard.tsx:21 | - | `{devices: Record<ld,number>, simulation: bool}` | banner |
| 13 | GET `/ieds/{id}/signals?...` | DeviceSignals.tsx:40 (150 ms debounce), LiveValueTable.tsx:30 (150), ExportScl.tsx:23 (150), SequenceEditor.tsx:24 (200), SourceEditor.tsx:35 (200) & :43, SignalPicker.tsx:29 & :44, SignalSearchPicker.tsx:42 (150), RelayStatusEditor.tsx:38/:60, StateMappingEditor.tsx:20 | - | Query keys: `q, offset, limit, report, kind (comma list), fc, writable=true, ld, bad_quality=true, in_dataset=true, ref` (components/signalQuery.ts:9-20, devices/deviceFilters.ts:44-53). `ref=` lookups read `items[0]`. Reads `items: Signal[]` (types.ts:27-32: `ref,fc,kind,value,writable,quality,timestamp,description,quality_ref?,timestamp_ref?,enums`), `total`, `devices?` (logical devices list). | inline message / ignored |
| 14 | POST `/ieds/{id}/export-scl` | devices/ExportScl.tsx:94 (`download`) | - | `{overrides: Record<ref,value>}`; response is a file blob saved as `<ied>.edited.scd` | `error` text |
| 15 | POST `/runtime/{start\|stop\|pause}` | useStudio.ts:160 (TopBar.tsx:54-61, App.tsx:82, DiagramPage.tsx:114); DeviceRunSummary.tsx:16 (`start`) | - | no body; `pause` toggles pause/resume | banner |
| 16 | POST `/runtime/ieds/{id}/{start\|stop}` | devices/RunThisIed.tsx:17 | - | no body; id is `encodeURIComponent`-ed | banner |
| 17 | GET `/equipment/states` | useStudio.ts:41 (on SSE open and on `resync`) | event driven | `states[]{node_id,value}`, `energization{nodes:Record<id,EnergyState[]>,edges:Record<id,EnergyState>,hints?}` | banner |
| 18 | POST `/equipment/{node}/operate` | diagram/DiagramPage.tsx:81 | - | `{action}`; reads `action` (string, non-empty: `action[0].toUpperCase()`), `wire_value: boolean\|null` | banner |
| 19 | POST `/equipment/{node}/test` | DiagramPage.tsx:91 | - | `{action, value?, row?}`; response ignored | banner |
| 20 | POST `/signals/update` | test/signalActions.ts:8 | - | `{ied_id, ref, value}` | banner |
| 21 | POST `/signals/pulse` | signalActions.ts:11 | - | `{ied_id, ref, active_value, duration_ms}` | banner |
| 22 | POST `/signals/release-override` | signalActions.ts:13 | - | `{ied_id, ref}` | banner |
| 23 | POST `/signals/source` | signalActions.ts:16 (SourceEditor, CsvPlayback.tsx:55) | - | `{ied_id, ref, spec}`; CSV spec = `{mode:'csv',filename,csv_text,time_column,value_column,interpolation,repeat}` | banner |
| 24 | POST `/traces/preview` | test/CsvPlayback.tsx:26 (`upload`) | - | multipart `file`; reads `columns[]`, `rows[][]`, `row_count` | banner |
| 25 | GET `/project/test-panel` | test/TestPoints.tsx:26 | 1000 while running (else once) | reads `items[]` = TestControl + `signal`(Signal), `automatic`, `overridden` | unhandled rejection (no catch) |
| 26 | PUT `/project/test-panel` | TestPoints.tsx:32, DeviceSignals.tsx:48 | - | **top-level JSON array** of TestControl `{id,label,ied_id,ref,report_ref,duration_ms,active_value}` | banner |
| 27 | PUT `/scenarios` | test/SequenceEditor.tsx:63, :87 | - | **top-level JSON array** of Scenario `{id,name,enabled,steps[{id,at,ied_id,ref,action,value}]}` | banner |
| 28 | POST `/scenarios/{id}/start` | SequenceEditor.tsx:62 | - | no body | banner |
| 29 | GET `/scenarios/runs` | SequenceEditor.tsx:34 | 1000 (always, while mounted) | **top-level array** `[{id,state,completed_steps}]` | inline error |
| 30 | GET `/project/substation-sources` | diagram/SldImport.tsx:31 | - | `sources[]{id,filename,ieds[],substations[],equipment}` | dialog |
| 31 | POST `/project/diagram/from-scl/preview` | SldImport.tsx:38 | - | `{source_id, mode:'beside'\|'replace'}`; reads `diagram, revision` (compared `===` to project.revision, sldPreview.ts:33), `layout:'esld'\|'automatic'`, `removed, drawn[]{path,kind,label,bound}, skipped[]{path,type,reason}, unbound[]{path,kind,reasons}, diagnostics[], already_present[]` | dialog |

## 3. Live (real IED), health, signal flow

| # | Method + path | Callers | Poll | Fields read | Errors |
|---|---|---|---|---|---|
| 32 | GET `/live` | devices/DeviceDetail.tsx:37 | 2000 while running and role=real | `sessions[]{ied_id,name,state,host,port,endpoint_source,message,access{reports,polling_datasets,notes}\|null}` | swallowed |
| 33 | POST `/live/{id}/read` | devices/LiveValueTable.tsx:39 | - | `{refs[]}`; reads `values: Record<ref,{value, live_state?}>` (`'stale'` checked), `errors: Record<ref,string>` | inline |
| 34 | GET `/live/{id}/comparison` | devices/ComparisonView.tsx:25 | - | `summary{datasets_different, reports_missing, missing_on_device?, extra_on_device?, type_mismatch?}`, `signals[]{ref,fc,category,scd_type,device_type,hint}` (`category!=='match'`), `datasets[]{ref,status,missing[],extra[]}`, `reports[]{ref,status}` | inline |
| 35 | GET `/health` | shared poll (api/polls.ts:6): HealthView.tsx:27, useModes.ts:10, SignalFlowView.tsx:28 | 3000 while running (HealthView/useModes: once when stopped; SignalFlowView only subscribes while running) | `verdict, running, checked_at, ieds[]{id,name,role,verdict,checks[]{area,item,result,message,observed_at,source,stale}}, summary{problem,warning,not_checked,ok}, unknown[]{gocb_ref,app_id,mac,src_mac,vlan_id,go_id,conf_rev,st_num}, unknown_note, simulation_sources?[], modes{ieds:Record<id,{devices,simulation,worst}>,equipment:Record<node,string>}` | HealthView shows error; others ignore |
| 36 | POST `/health/refresh` | HealthView.tsx:36 | - | same shape as `/health` | inline |
| 37 | GET `/signal-flow/sources` | devices/SignalFlowView.tsx:32 | - | `sources[]{id,filename,ieds[]}` | inline |
| 38 | GET `/signal-flow?source=<id>[&ied=<name>]` | SignalFlowView.tsx:42 (`source` NOT url-encoded; `ied` encoded) | - | `source, filename, ieds[]{name,desc,manufacturer,loaded}, blocks[], edges[]{id,type,publisher,block,subscriber,match,signals[]}, diagnostics[]{level,ied,message}, summary{...}` (devices/signalFlow.ts:1-11) | inline |
