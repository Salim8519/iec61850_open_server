# Frontend SSE consumption and parity-sensitive assumptions

`FE` = `studio/frontend/src`. Backend references are context only (`studio/backend/studio_app`).

## 1. SSE connection logic (FE/state/useStudio.ts:73-125)

| Aspect | Frontend behaviour | What the backend must keep |
|---|---|---|
| URL | `new EventSource('/api/v1/events')` (useStudio.ts:78), created only after `connectSession()` + first `GET /project` succeed, and only inside `<App>` (i.e. after the license gate says licensed) | same path, same origin, cookie auth (EventSource sends cookies, never `X-Studio-Frontend`) |
| Query / cursor | FE never sends `?after=`; first connect has no `Last-Event-ID` | backend today treats missing cursor as 0 => **replays the whole bus buffer (capacity 5000)** on every page load (api/events.py:11, runtime/event_bus.py:21-27). FE then keeps the last 500 non-`mms_access` events, so a reloaded tab shows past Activity and re-applies old `signal` values and re-runs `refresh()` for every replayed `runtime`/`project_changed`/... event |
| Reconnect | native EventSource auto-reconnect; no `onerror` handler, no manual backoff, no close-on-error | server sends `retry: 1000` first (events.py:15); browser then reconnects after ~1 s and sends header `Last-Event-ID: <last id: line>` |
| `Last-Event-ID` | sent automatically by the browser because each event carries an SSE `id:` line | `id:` value must equal the JSON `id` and be a monotonically increasing integer; backend parses it with `int()` |
| Event framing | only `stream.onmessage` is used (useStudio.ts:82) | events must be **unnamed** (no `event:` line, or `event: message`); a named event would never reach the FE. Payload is a single `data:` line of JSON parsed with `JSON.parse` |
| Heartbeat | FE has no heartbeat timeout; SSE comment lines are ignored by the browser | backend sends `: heartbeat\n\n` every ~10 s of silence (events.py:19); only needed to keep the socket alive |
| `onopen` | every (re)connect runs `Promise.all([refresh(), hydrateEquipment()])` = `GET /project` + `GET /equipment/states` (useStudio.ts:79-81); errors -> banner | stream must only "open" (200 + `text/event-stream`) when the client is authorised; a non-200 or wrong content type makes EventSource give up permanently (no FE retry) |
| `resync` | treated like `runtime`: `GET /project`, plus `GET /equipment/states` (useStudio.ts:113-120) | emitted when the cursor fell out of the buffer; payload `{id, kind:'resync', timestamp}` only |
| Close | `stream.close()` on unmount (useStudio.ts:124) | - |

## 2. Envelope fields read on every event (FE/api/types.ts:73-77)

| Field | Use |
|---|---|
| `id` (number) | React key in Activity table (EventLog.tsx:15) - duplicates after a replay cause key clashes |
| `kind` (string) | dispatch; displayed as `kind.replaceAll('_',' ')` (EventLog.tsx:17) |
| `timestamp` (epoch **seconds**, float) | `new Date(timestamp*1000).toLocaleTimeString()` (EventLog.tsx:16) |
| `message`, `ref`, `status_ref`, `state`, `value` | generic Activity detail: `message \|\| ref \|\| status_ref \|\| state \|\| ''` + ` -> value` when `value !== undefined` (eventDetails.ts:47-48). `null` value renders "unknown"; a missing key renders nothing |

All events except `mms_access` are appended to the Activity list (last 500, useStudio.ts:85) and passed
to `SignalLayerStore.handle` (useStudio.ts:86). `events.length` is the sidebar badge (App.tsx:56).

## 3. Event kinds with specific handling

| Kind | Fields read | Effect (FE file:line) |
|---|---|---|
| `signal` | `ied_id`, `ref`, `status_ref?`, `value`, `updates?: Record<ref,value>` | `values["<ied_id>:<status_ref\|\|ref>"] = value` plus one entry per `updates` key (useStudio.ts:87-92); SLD: if `ied_id` is a GOOSE publisher in the link map, finds links whose block `member_refs` match `ref` (exact or `ref` starts with `member + '.'`) and pulses them, label from last path segment minus `.stVal`, `.Pos.stVal` => position wording (signalLayerStore.ts:77-86, signalPulses.ts:36-38) |
| `control_applied` | `ied_id`, `ref`, `status_ref?`, `value`, `updates?`, `ctl_value`, `action`, `external` (=== true), `peer` ("ip:port") | same value update as `signal`; Activity text `ref / ctlVal=<ctl_value> / <action>` (eventDetails.ts:38-40); if `external===true`: SLD control badge "`<peer ip>` operated `<last ref segment>` on `<IED>`" (signalLayerStore.ts:89-94) |
| `equipment_state` | `node_id`, `value` | `equipmentValues.set(node_id, value)` (useStudio.ts:93-95); compared with `Object.is` |
| `energization` | `nodes: Record<id,EnergyState[]>`, `edges: Record<id,EnergyState>`, `hints?` | `energization.set({nodes,edges,hints})` (useStudio.ts:96-99); Activity text counts edge states in object-iteration order: "wires now: 3 live, 2 dead" (`replace('_',' ')` first underscore only) (eventDetails.ts:41-46) |
| `project_changed` | `source`, `revision` | if `source === this tab's tab_id` => only `project.revision = Number(revision)`; else discard pending save, reset undo identity, `GET /project` (useStudio.ts:100-112). Activity: "authoritative project revision N" |
| `runtime` | `state` | if `state === 'stopped'` clear live values, equipment, energization, SLD layers; always `GET /project` (useStudio.ts:113-117) |
| `worker_failed`, `live_status` | (none specific) | `GET /project` (useStudio.ts:113) |
| `resync` | - | `GET /project` + `GET /equipment/states` |
| `mms_access` | `ied_id`, `clients` (array of MmsClient) | NOT added to Activity; SLD readers layer `clients[ied_id] = clients` (signalLayerStore.ts:87-88) |
| `goose_status` | `gocb_ref` (string), `state`, `problem?`, `st_num?` (number only) | SLD GOOSE state per gocb_ref, sets `listening`, pulses observed links when `st_num` changes (signalLayerStore.ts:69-76) |
| `goose_conflict` | (count only) | count of these in the kept 500 changes => `GET /goose/share` and show conflict dialog (GooseConflicts.tsx:14-21) |
| `control_selection` | `ref`, `state` (`selected\|operated\|canceled\|timed_out\|failed\|disconnected`), `sbo_timeout_ms` | Activity text with "N-second timeout" when divisible by 1000 else "N ms timeout" (eventDetails.ts:19-33) |
| `equipment_feedback_observed`, `equipment_feedback_applied` | `control_ref`, `wire_value`, `action`, `target_count`, `changed_count` | Activity text "... / feedback changed (c/t)" (eventDetails.ts:7-13, 34-37) |

Generic Activity-only kinds (emitted by backend, shown via section 2 rule, no FE logic): e.g.
`automation_failed, automation_late, equipment_button_applied, equipment_feedback_failed, equipment_test,
explorer_command, explorer_connected, gateway_command, gateway_started, gateway_stopped, goose_input,
goose_input_refused, goose_listener_failed, goose_share, goose_storm, goose_supervision_failed, goose_unknown,
goose_uplink, ied_runtime, ieds_removed, interlock_hint_observed, live_read_error, live_stale,
master_connected, master_disconnected, pulse_cancelled, pulse_started, scenario_*, source, source_failed,
source_resumed, sv_listener_failed, sv_status`, plus native-worker kinds forwarded by
`coordinator.on_worker_event` (`event.get("event")`). For these, the visible text depends on which of
`message/ref/status_ref/state/value` keys exist, so key presence and the `message` wording are user-visible.

## 4. Parity-sensitive frontend assumptions

Highest risk first. "Go risk" = what a naive Go port would change.

| # | Assumption | Where (FE file:line) | Go risk |
|---|---|---|---|
| 1 | **Every `/api/v1` response, including errors, 404s and 401/403/402/409/503, must be a JSON body.** `api()` calls `response.json()` before looking at the status; a plain-text or empty body (e.g. Go `http.Error`, default mux 404/405, `204 No Content`) throws a SyntaxError whose text is shown instead of the intended message, and the 403/402/404 branches never run. | api/client.ts:11 | default Go 404/405 handlers, panics recovered as text, empty 200s for "no response needed" POSTs |
| 2 | Error contract: message in **`error`** (string). Extra keys read: `active_frontend` (string on 403, null/absent in packaged), `code` (license calls only: `bound_elsewhere`, `license_required`), `problems[]` (gateway points save, via `error.data`). Status codes with FE branches: **402** (fires `studio-license-required`, re-gates app), **403 + active_frontend** (tab redirect), **404** (fixed "restart ScadaBench" text, so a missing route must be 404 and a missing *entity* should NOT be 404 or the user gets the wrong text). 409/503/401 have no branch: only `error` text is shown. | client.ts:12-24; license.ts:34; gateway.ts:20; LicenseActions.tsx:40 | returning 404 for "IED not found"; different key name (`message`) |
| 3 | SSE events must be unnamed `data:` JSON with matching `id:` lines; `retry:` honoured; missing cursor => full replay of the buffer (FE relies on this to repopulate Activity after reload). | useStudio.ts:78-86; events.py:11-22 | using `event: <kind>` lines; no `id:`; starting from "now" |
| 4 | Exact backend text parsed for status: gateway command `applied` starts with `"operated"` or contains `"is now"` => done; starts with `"refused"` => refused; `observed` contains `"did not change"` => warning. | gateway/CommandLog.tsx:11-13, :38; gateway/logFilters.ts:39-43 | rewording messages |
| 5 | ISO time strings sliced by position: `observed_at.slice(11,19)` must yield `HH:MM:SS`, i.e. `YYYY-MM-DDTHH:MM:SS...` (Python `isoformat()`); Go RFC3339Nano is compatible only if the date part is 10 chars and `T` separated. `checked_at` from health is displayed raw. | devices/HealthView.tsx:66; devices/signalFlow.ts:28 | Go `time.Time` default JSON is RFC3339 (OK) but a space separator or Unix number breaks it |
| 6 | Epoch **seconds** (float) for: event `timestamp`, traffic/command `at`, explorer `at`, update `checked_at`, license `expires_at` (all `*1000`). Master point `time_ms` is ms. | EventLog.tsx:16; TrafficLog.tsx:52; CommandLog.tsx:33; MasterCommandLog.tsx:31; ValueText.tsx:15; UpdateSection.tsx:12; LicenseSection.tsx:22 | emitting ms or ISO strings |
| 7 | JSON key ORDER is user-visible where FE iterates backend objects: `Signal.enums` option order (Python insertion order = enum ord), `modes.ieds[id].devices` (test-mode LD list order), energization `edges` (Activity "wires now" wording order), `mms_access` `ieds` map, `event.updates`. Go `map[string]T` marshals keys **sorted**, so these orders change unless ordered structures are used. | SignalValueEditor.tsx:16-21; ExportScl.tsx:75; TestModeCard.tsx:13; eventDetails.ts:43-44; SignalOverlay.tsx:42 | `map[string]` encoding |
| 8 | `list_filters` change detection compares `JSON.stringify(stored prefs[list])` with the FE-built object (spec key order). If the backend returns keys in a different order the comparison never matches and every filter change re-PUTs `/preferences` (extra disk writes, not visible). | state/useListFilter.ts:29-31 | sorted map keys |
| 9 | `PUT /preferences` and `PUT /ieds/{id}/settings`, `PUT /network`, `PUT /goose/share`, `PUT /app/update` send **partial** bodies; server must merge, and `list_filters` sends ONE list key that must be merged into the stored map (not replace other lists). | useListFilter.ts:31; preferences.ts:11-16; DeviceDetail.tsx:58 etc. | struct decode that zero-fills missing fields |
| 10 | Top-level JSON **arrays**: request bodies of `PUT /project/test-panel` and `PUT /scenarios`; response of `GET /scenarios/runs`. | TestPoints.tsx:32; DeviceSignals.tsx:48; SequenceEditor.tsx:34,63,87 | wrapping in an object |
| 11 | null vs missing: FE checks `=== null` for `wire_value` (null => no "sent" suffix; missing => "sent 0"), `result.revision` compared with `===` to project revision, `value !== undefined` in Activity, `entry.value !== null` (master log), `item[key] === null \|\| undefined` (health unknown), `previewIsCurrent` strict equality. Optional keys typed `?` (e.g. `running_ieds`, `simulation_sources`, `devices`, `hints`, `goose_state`) default when absent; `running_ieds` absent => all non-off IEDs counted. `omitempty` vs `null` therefore changes UI text. | DiagramPage.tsx:83; eventDetails.ts:48; MasterCommandLog.tsx:34; HealthView.tsx:75; TopBar.tsx:28 | `omitempty`, nil slices encoded as `null` (FE calls `.length`, `.map`, `.filter` on arrays such as `ieds`, `checks`, `items`, `entries`, `points`, `conflicts`: `null` crashes the view) |
| 12 | Numbers: JS cannot tell `1` from `1.0`, so int/float encoding is safe for JSON numbers; but `NaN`/`Infinity` (Python `json.dumps` default allows them) make `JSON.parse` fail for the whole response/event. `rms/magnitude/angle` must be numbers (`.toFixed`). `received_smp_synch` used as array index 0-2. `sbo_timeout_ms % 1000` wording. Backend-preformatted strings (`value_text`, `quality_text`, `text`, `summary`, `message`, `hint`, `reason`) are shown verbatim, so Go float formatting inside those strings (e.g. `%v` of 0.1, `1e-07` vs `1e-7`) is visible. | SvTraffic.tsx:63-64; eventDetails.ts:21-23; PointTable.tsx:98 | float-to-string helpers |
| 13 | Enum vocabularies compared literally: `runtime` (`stopped\|running\|paused`), IED `role` (`simulated\|real\|off`), `state` strings mapped in `copy/status.ts` (e.g. prefix `real_`), health `verdict/result` (`problem\|warning\|not_checked\|ok`), compare `category` (`match`...), adapter `status === 'Up'`, link `status` (`declared\|remote\|...`), block `status === 'sendable'`, `live_state === 'stale'`, traffic `dir` (`rx\|tx\|info`), command `protocol === 'manual'`, update `error` codes, license `mode/plan`. | many; e.g. copy/status.ts:85, setupWords.ts:14, signalPulses.ts:30-38 | renamed constants |
| 14 | `project_changed.source` must echo the `tab_id` sent with `PUT /project/diagram`, and `revision` must be numeric (FE does `Number(...)`); otherwise every own save triggers a full reload and drops the pending undo identity. | useStudio.ts:55-57, 100-103 | not echoing tab_id |
| 15 | Downloads: `/project/download` relies on server `Content-Disposition` filename (`download=""`); plain links (`/ieds/{id}/source`, CSVs, `/health/report.html`) need `attachment` (or the right MIME for the HTML report) because they navigate the current window otherwise; the app-window exit prompt is bypassed only for real downloads. | TopBar.tsx:12-17; DeviceFiles.tsx:13; HealthView.tsx:40-41 | missing Content-Disposition |
| 16 | `X-Studio-Frontend` is sent by `api()`/license calls but NOT by `download()`, `<a href>` links or EventSource; authorisation for those must work with cookie + `Origin` only (download POST sends `Origin`). | client.ts:7; download.ts:3-5 | requiring the header everywhere |
| 17 | `/session` may be POSTed twice with the same token (LicenseBoundary + StrictMode); `/license/*` and `/preferences` must answer while the project is still opening (FE calls them before `/project`). Long-held requests are fine: FE has no timeouts (backend holds up to 300 s then 503 `project_opening`). | LicenseBoundary.tsx:17; App.tsx:35; startup_gate.py:10 | one-shot tokens, gating license behind startup |
| 18 | Fixed English FE fallback texts depend on status only: 404 => "The running backend does not provide this action. Save your project and restart ScadaBench..."; 402 without `error` => "A license is required."; generic `Request failed (<status>)`. | client.ts:21-24; download.ts:8-10 | - |
| 19 | Query parameters FE relies on: booleans as the literal string `true` (`writable`, `bad_quality`, `in_dataset`), `refresh=1`, comma-joined `kind`, `after=<int>`, `signal-flow?source=` unencoded id. | signalQuery.ts:15-18; deviceFilters.ts:50-51; NetworkSetup.tsx:28 | strict bool parsing |
