# HTTP and event API

All application routes use `/api/v1`. Administration is loopback-only and requires
the launcher session cookie or bearer token. Development requests also identify the
authoritative frontend with `X-Studio-Frontend`; a stale Vite origin receives 403
with `active_frontend`. JSON errors use `error`, optional `code`, and optional
`details`. Runtime failures return 503; revision conflict returns 409. An unknown id answers 400 with
code `not_found` (Flask parity; not 404). A refused upload is read to the end before its JSON answer,
so large files never end in a connection reset (Go, change 0186).

## Session, project and runtime

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `POST /session` | `{token}` | Establish same-origin session |
| `GET /project` | - | Project snapshot, session, revision, runtime, `running_ieds`, recovery notice |
| `PUT /project/diagram` | `{diagram,revision,tab_id}` | Stopped-only revision-checked save |
| `POST /project/save` | - | Atomic autosave |
| `GET /project/download` | - | Current `.scadabench` ZIP (`substation.scadabench`) |
| `POST /project/open` | multipart `file` | Stopped-only validated restore/migration; a `.subsim` file name is refused (400, code `old_project_file`, ADR-0096) |
| `POST /runtime/start` | - | Start all workers after capability/preflight checks |
| `POST /runtime/stop` | - | Stop workers and flush append-only run log |
| `POST /runtime/pause` | - | Toggle automation pause; MMS remains active |
| `POST /runtime/ieds/{id}/start` | - | Run (simulated) or connect (real) one IED; joins a running run (ADR-0067); returns `{state, running}` |
| `POST /runtime/ieds/{id}/stop` | - | Stop or disconnect one IED; the others keep running; the last one stops the run; an id not in the project: 400 `not_found` "Unknown IED" (Go, change 0186) |

## SCL and IEDs

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `POST /scl/validate` | multipart `file` | Offline schema result |
| `POST /scl/preview` | multipart `file` | Selectable IEDs; imports nothing, so the Go engine also answers while running (change 0186). Go: an IED whose import would fail has `unavailable_reason` "Cannot be imported at <LD/LN.DO.DA>: …" so it can be left out (ADR-0114). An IED with several access points is simulated on the station one and has `access_point_note` naming what is not simulated (ADR-0117) |
| `POST /scl/import` | file + JSON `selected_names` form field | Preserve source; import selected IEDs. Go: up to 100 MB; an error names the IED and the SCL path ("IED X at LD/LN.DO.DA: …"); what the import adapted is listed in the IED's `warnings` (ADR-0114) |
| `POST /ieds/template` | `{name}` | Add laboratory IED template |
| `GET /ieds/{id}/source` | - | Exact retained XML download |
| `GET /ieds/{id}/deadbands` | - | Go only (ADR-0118): `{enabled, points[{ref, object, db, zero_db, range_min?, range_max?, band?, zero_band?, db_writable, zero_db_writable, applies, note}], applied, without_range}`: every measurement with a db; `note` in plain words |
| `PUT /ieds/{id}/deadbands` | `{enabled: bool}` | Go only (ADR-0118): saves "Use SCD deadbands" (`scd_deadbands: true` stored only while on), applies it to a running IED at once, answers as GET |
| `POST /ieds/{id}/export-scl` | `{overrides}` | Validate/re-import edited copy; project unchanged |
| `GET /ieds/{id}/removal-preview` | - | Single-delete impact |
| `DELETE /ieds/{id}` | `{confirm:true}` | Confirmed single deletion with recovery archive |
| `POST /ieds/removal-preview` | `{ids}` | Bulk impact and fingerprint |
| `POST /ieds/remove` | `{ids,fingerprint,confirm:true}` | Fingerprint-checked bulk removal |

## Signals, controls and equipment

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `GET /ieds/{id}/signals` | `q,fc,kind,writable,report,ref,ld,bad_quality,in_dataset,offset,limit` | Filtered/paged typed signals and the IED's logical `devices` (read-only filters, change 0093) |
| `GET /ieds/{id}/signals` (Go only) | `point_kind`, `helpers=hide`, `kind_counts=1` ([ADR-0095](../decisions/ADR-0095-universal-point-kinds.md)); `annotate=1`, `ln=<LD/LN>`, `ln_counts=1`, `near=<text>` ([ADR-0113](../decisions/ADR-0113-smart-point-picker.md)) | Adds keys only: `kind_counts` {kind: n} without `point_kind`; items get `point_kind`; `ln_counts` {"LD/LN": n} without `ln`; `suggested` (at most 8 items mentioning `near` as a whole word, description first, each with `suggested_by`). Without them the answer is byte-identical |
| `POST /signals/update` | `{ied_id,ref,value,quality?}` | Acknowledged typed update; Go refuses an Enum ord outside its EnumType and non-ASCII VisString text in plain words; a Mod write also writes the Beh it implies in the same batch (ADR-0115) |
| `POST /signals/source` | `{ied_id,ref,spec}` | Configure validated source |
| `POST /signals/pulse` | `{ied_id,ref,active_value?,duration_ms?}` | Guarded Boolean pulse |
| `POST /signals/release-override` | `{ied_id,ref}` | Resume configured automatic source |
| `POST /controls/operate` | `{ied_id,ref,value}` | Low-level independent native probe operation |
| `POST /equipment/{node}/operate` | `{action}` or `{value}` | Typed mapped equipment command/feedback |
| `GET /equipment/states` | - | Authoritative mapped canvas state snapshot |
| `POST /equipment/{node}/test` | `{action, value?, row?}` | Test button on a running simulated IED: writes the bound point via the sample writer, sends no command; real IEDs refused (ADR-0067) |
| `POST /equipment/{node}/test` (point panel) | `{action:"set", row, value}` (status true/false, double 0-3, value number), `{action:"send", row, value}` (command true/false), `{action:"healthy"}` | Point panel (ADR-0102, Go only): set a status, double or value point of a simulated IED, send a command point through Operate, or write every healthy value; answers `{ref,value,type,ied_id,correlation_id}` or the batch summary `{requested,applied,failed,results}`; real IEDs refused |
| `POST /equipment/{node}/select` | `{action:"open"\|"close", options?}` | Go only (ADR-0108): select a CB/DS/ES whose control is `sbo-*`; `{node_id,action,model,selected,expires_ms}` (window = sboTimeout minus margin); direct models refused in words |
| `POST /equipment/{node}/cancel` | any | Go only (ADR-0108): end the node's selection; `{node_id,cancelled,action?}` |
| `POST /equipment/{node}/goose` | equipment: `{action:"open"\|"closed"\|"on"\|"off"\|"value", value?}`; IED component: `{gocb_ref, ref, value}` | Go only (ADR-0112): send GOOSE through the node's `goose_link` or one value of the IED component's own message, written in ONE sample batch (stNum rises once); `{node_id,ied_id,gocb_ref,action,applied,correlation_id}`; refused in words for real, skipped, GOOSE-off or stopped IEDs, a missing link or a wrong value; events `goose_send_requested`, `goose_send_applied` / `goose_send_failed` |
| `GET /diagram/goose-ieds` | - | Go only (ADR-0112): IED components "Add all GOOSE IEDs" would add: `{nodes:[{ied_id,name,position{x,y}}]}` (every GOOSE IED without one, one row above the drawing); nothing saved |
| `POST /equipment/{node}/test` (point panel, ADR-0108) | `{action:"select"\|"cancel", row, value?}`; `send` values also a whole number (INC), number (APC), enum label (ENC), `"higher"`/`"lower"` (BAC, BSC, ISC) | Select before operate of an SBO command row (`send` must repeat the selected value); typed values checked against `min`/`max`/`step` and the keys of `labels` |
| `GET /controls/typed?ied_id=` (ADR-0104 list, served on this branch by an adapter) | none | `{controls, extended_controls}`: the SPC/DPC list and the typed controls read from the stored SCL, each `{ref, status, cdc, ctl_val_kind, ctl_val_ref, value_kind (int\|enum\|float\|step), model (ctlModel 0..4), sbo_timeout_ms, min, max, step, labels {label: number}, trans_ind}`; Go-only |

Equipment actions are Open/Close for CB/DS/ES, Operate/Reset for Trip Relay and
Press for a Push Button. The server rejects mixed `action` and `value`.
A tap-changer transformer (ADR-0109, Go only) takes `raise`, `lower`, `auto` and `manual`: raise/lower
route to a BSC/ISC step control (typed command path, ADR-0104) or an SPC pair (canvas command path), then
write the new tap as control feedback for a simulated tap position; it answers
`{node_id, action, wire_value, tap, feedback, correlation_id}`; real IEDs are refused. Events
`tap_command_requested|applied|failed` and `avr_decision` (`event`: `out_of_band`, `step`, `back_in_band`,
`at_max_tap`, `at_min_tap`, `no_position`, `manual`, `blocked`, `mode_auto`, `mode_manual`) carry
`node_id`, `ied_id`, `correlation_id` and `origin` (`studio_api` or `avr`). `GET /equipment/states` and
`equipment_state` add `<node>:tap`, `<node>:mode`, `<node>:lv` and `<node>:avr`
(`{auto, regulator, state, remaining_s, deviation_pct, lv_kv}`).

### Typed commands (Go engine only, ADR-0104)

Simulated IEDs of the running project only (the same rule as `/controls/operate`). `options` is
`{test?, interlock_check?, synchro_check?, orcat? (0-8; 0 or absent sends 3 remote), orident? (<= 64)}`.
`value` is a boolean (SPC/DPC), a whole number (INC/ISC), a label or ord (ENC), a number (APC),
`"higher"`/`"lower"`/`"stop"` (BSC/BAC; ISC higher/lower moves one position from the present one).
Device refusals answer 409 with the device's reason in plain English; bad input 400.

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `POST /controls/operate` | `{ied_id,ref,value,options?}` | `{ied_id,ref,value:bool}` keeps the old path byte for byte; `options` or a non-boolean `value` goes typed: `{ref, model, selected, operated, canceled, terminated, add_cause, add_cause_text, expires_ms, ok}`. After an expired Select: 409 "Select expired after N s. Press Select again." Without a Select, SBO models select and operate in one step |
| `POST /controls/select` | `{ied_id,ref,value?,options?}` | `{ok, model (direct-normal\|sbo-normal\|direct-enhanced\|sbo-enhanced), expires_ms}`; direct models answer `expires_ms: 0`; SBO enhanced needs `value` |
| `POST /controls/cancel` | `{ied_id,ref}` | `{ok}`; nothing selected is not an error |
| `POST /controls/checks` | `{ied_id,ref,blocked?,interlock_blocked?,synchro_blocked?}` | Simulated blocks of one typed control (AddCause 9, 10, 11 when the command asks for that check) |
| `GET /controls/typed?ied_id=` | - | `{controls, extended_controls}`; each extended entry is `{ref, status, cdc, ctl_val_kind, ctl_val_ref, value_kind, model, sbo_timeout_ms, min, max, step, labels, trans_ind}` |
| `POST /tools/explorer/command` | `{action: select\|operate\|cancel, ref, kind: bool\|int\|enum\|float\|step, value, options, confirm}` | IED Explorer typed step on the connected device; `confirm: true` for select/operate; `{result, summary}` with the device's answer |

Worker IPC: the start request may carry `extended_controls` (the entries above); the worker lists
`typed_controls` and accepts `control_checks`. `studio_control` lists `typed_commands` and serves
`typed_select`, `typed_operate`, `typed_cancel`. Typed control events add `cdc`, `or_cat`,
`or_ident`, `ctl_num` (and `typed: true` on `control_applied`).
Press for a Push Button. The server rejects mixed `action` and `value`. Operate takes an
optional `options` `{test, interlock_check, synchro_check, orcat 0..8, orident ≤ 64}` (Go only,
ADR-0108); without it and without a selection the request and native call are unchanged. A
held selection must match the action; an expired one answers "… ran out after 27 s, so nothing
was sent". New events: `equipment_selected`, `equipment_select_failed`, `equipment_select_expired`,
`equipment_select_cancelled`, `point_panel_selected`, `point_panel_select_cancelled`.

## Network, GOOSE and real IEDs

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `GET /network` | - | Network mode and Npcap status |
| `PUT /network` | `{mode,adapter_guid?,adapter_label?}` | Stopped-only; endpoint conflicts refused |
| `GET /network/adapters` | `refresh=1` | Windows adapters with IPv4 addresses (cached 10 s) |
| `POST /network/plan/preview` | `{adapter_guid,strategy,manual?}` | Address rows with fix hints (a row that cannot be used adds `alternative: {ip, prefix}`, change 0098), change items, fingerprint; Go also plans SNMP devices (`ied_id` `snmp:<id>`, `kind`, `saved`, item `snmp_firewall`, change 0148) |
| `GET /network/plan/progress` | | Go only (ADR-0093): the running address check: `run`, `running`, `steps[{key,label,state,ms}]`, `asked`, `answered`, `rows[{id,name,address,state}]` with state `checking`/`free`/`in_use`/`not_checked` |
| `GET /network/lab` | | Go only (ADR-0120): the ScadaBench Lab card: `installed`, `running` (service), `card`, `card_present`, `card_name`, `ready`, `active` (project uses the card), `ieds_running`, `real_cards[{name,kind,connected,networks}]`, `rows[{ied_id,name,scl_ip,ip,yours,on_card,status,hint,card?,alternative?}]` with status `ok`/`same_range`/`real_device`/`checking`/`duplicate`/`other_card`/`invalid`/`missing`, `checking`, `problem?`, `card_addresses` |
| `POST /network/lab/check` | `addresses{ied_id: ip}` | The same answer for addresses typed before Use; Go only |
| `POST /network/lab/install` | | Create (or repair) the card and the `ScadaBenchLabNet` service through one UAC prompt; journaled in `network/lab_journal.jsonl`; Go only |
| `POST /network/lab/use` | `addresses{ied_id: ip}` | Stopped only: every simulated IED's address on the card, `lab_ip` stored, lab mode on the card; no prompt; refused with the first blocking row's words; Go only |
| `PUT /network/lab/address` | `ied_id`, `ip` | Any time once in use: one IED moves (a running IED is stopped and started again); Go only |
| `POST /network/lab/ping` | `ip` | One ICMP echo: `ok`, `ms`, `ttl`; Go only |
| `POST /network/lab/remove` | | Stopped only: project back to this PC only, then one UAC prompt removes service, program and card; Go only |
| `POST /network/plan/apply` | `{fingerprint,confirm:true}` | Re-plan; 409 if changed; elevated helper; enters lab mode |
| `GET /network/cleanup/preview` | - | Outstanding owned items and fingerprint |
| `POST /network/cleanup/apply` | `{fingerprint,confirm:true}` | Remove owned items; back to loopback mode |
| `GET /network/gateway-firewall` | - | Gateway firewall check: items, fingerprint, `open`, `blocked`, `needs_admin` (no elevation) |
| `POST /network/gateway-firewall` | `{fingerprint,confirm:true}` | Allow rule + switch off Block rules for the gateway worker; 409 if Windows changed (ADR-0062) |
| `PUT /ieds/{id}/settings` | `{role?,live?,lab_ip?,goose_publish?,port?}` | Stopped-only; SCL never rewritten |
| `GET /goose` | - | Configured/observed GOOSE blocks (`origin` app/wire, `src_mac`), problems, unknown GOOSE, `engine` (ADR-0073) |
| `GET /goose/share` | - | Where GOOSE goes: `reach`, picked `uplink`, Advanced `choices`, `risk` {level, reasons}, `npcap`, `state`, `message`, `conflicts`; Go engine also `goes_out`, `alone` (ADR-0119) |
| `PUT /goose/share` | `{reach, uplink_guid?, adapter_guid?, confirm?}` | Allowed while running. Go engine (ADR-0119): `reach` `auto`/`local`; `confirm:true` with the `adapter_guid` shown records the real-device confirmation for that adapter. Python: `outside` needs `confirm:true`, Npcap and the `adapter_guid` shown (GOOSE-19) |
| `POST /goose/conflicts` | `{ied_id, choice: stop\|continue, confirm?}` | Answer a conflict for this Run; `continue` needs `confirm:true` |
| `GET /goose/links` | - | GOOSE link map (ADR-0065): `ieds`, `blocks` (status, `member_refs`), `links` (status, inputs, `live`), `diagnostics`, `summary` |
| `GET /ieds/{id}/goose/messages` | - | Go only (ADR-0112): `{ied_id,name,role,running,can_send,why_not,messages[{gocb_ref,name,dataset,go_id,app_id,conf_rev,subscribers,complete,members[{position,text,fc,refs[{ref,kind,fc,value_type (onoff\|position\|number\|null),writable,sendable,description,value}]}]}],receives[{gocb_ref,publisher,block,status}]}`; members FC-aware from the stored dataset |
| `GET /goose/members` | - | Go only (ADR-0112): `{ieds:{ied_id:{gocb_ref:[exact leaf refs]}}}` for expected pulses and the panel "GOOSE" tag |
| `GET /mms/access` | - | MMS clients per simulated IED (ADR-0066): address, nodes read, report blocks, commands |
| `GET /live` | - | Real-IED sessions (state, endpoint source, access notes) |
| `GET /live/{id}/comparison` | - | SCD vs device categories, dataset and report diffs |
| `GET /live/{id}/comparison.csv` | - | Same comparison as CSV |
| `POST /live/{id}/read` | `{refs}` (max 500) | On-demand read; device errors reported, not values |

New SSE kinds: `goose_publishing`, `goose_control`, `goose_status`, `goose_unknown`,
`goose_listener_failed`, `live_status`, `live_stale`, `live_read_error`. Signal events
from real devices carry `source` `live_report`, `live_poll` or `live_goose`.
ADR-0066 adds `goose_input` (`applied` refs, `problems`, `st_num`), `goose_input_refused`
(`reason`) and `mms_access` (client list, live view only, never in the run log); worker
`connection`, `report_control` and control events carry `peer` (`ip:port`) and
`control_applied` carries `external`. Signal events from GOOSE inputs have source `goose_input`.
ADR-0073 adds `goose_conflict` (`ied_id`, `gocb_ref`, `src_mac`, `sharing`), `goose_share` (`state`
off/checking/sharing), `goose_uplink` (`state`, `message`) and `goose_storm` (`ied_id`, `limit_per_second`).
ADR-0112 adds `goose_send_requested` (`node_id`, `ied_id`, `gocb_ref`, `action`, `refs`, `values`,
`correlation_id`, `origin`), then `goose_send_applied` (+ `applied`) or `goose_send_failed` (+ `message`);
the written values are `signal` events with source `goose_send`.

## Scenarios, tests and events

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `PUT /scenarios` | scenario array | Stopped-only validated definitions |
| `GET /scenarios/runs` | - | Active run snapshots |
| `POST /scenarios/{id}/start` | - | Start saved scenario while running/paused |
| `GET /project/test-panel` | - | Saved tests enriched with live signal/source state |
| `PUT /project/test-panel` | test array | Stopped-only validated panel save |
| `POST /traces/preview` | CSV multipart `file` | Five-row preview, columns and row count |
| `GET /events` | `after` or `Last-Event-ID` | SSE replay, heartbeat and `resync` marker |
| `GET /license/status` | - | `licensed, mode, plan, days_remaining, expires_at, reason, hwid, server_url, masked_key`; Go engine only, for a virtual machine licensed by its host PC: `mode: "vm-host"` and the masked `host_key` (ADR-0099) |
| `POST /license/trial` | - | Start the one trial via the license server; saves `license.cert` |
| `POST /license/activate` | `license_key` | Activate a lifetime key online; `move: true` answers 400 `moves_disabled` (licenses never move, ADR-0097) |
| `POST /license/activate-offline` | `certificate` text | Verify and save a pasted/loaded `license.cert` |
| `GET /license/host-key` | - | Go engine only (ADR-0099): can this PC share its lifetime license with its virtual machines: `available, reason` (plain words when not), `key` (masked), `enabled` (the Run value), `running` (the helper), `helper_found`, `port` (47862) |
| `PUT /license/host-key` | `enabled` | Switch "Let my virtual machines use this license" on (needs a shareable license, starts the helper) or off (always allowed, the helper ends itself); Go engine only |
| `GET /license/host-key/firewall` | - | The one Windows Firewall rule the helper needs, as a preview (`items`, `fingerprint`, `open`, `needs_admin`); Go engine only |
| `POST /license/host-key/firewall` | `fingerprint`, `confirm: true` | Add that rule through the network helper after the reviewed preview (journaled, removed by Clean up); Go engine only |
| `GET /license/activation-code` | - | `code` = `<machine ID>.<host key>` for the offline activation page; 400 `development` in development mode; Go engine only |
| `GET /app/update` | - | `current, auto, checked_at, checking, error, available, important, dismissed, latest` (ADR-0071; stored answer, no internet call) |
| `POST /app/update/check` | - | Ask the license server now (a repeat within 60 s reuses the answer); failures are an `error` code, never a 5xx |
| `PUT /app/update` | `{auto?: bool, dismissed?: version or null}` | Once-a-day switch; hide the note for one version (not for an important update) |
| `GET /preferences` | - | `tour_seen_version`, `tour_version`, `list_filters`, `sld_goose_layer`, `sld_readers_layer` |
| `PUT /preferences` | `{tour_seen_version?, list_filters?: {list: {key: text or bool}}, sld_goose_layer?, sld_readers_layer?}` | Store UI preferences in the data folder; a list's filters replace only that list (change 0093); layers 0 or 1 (change 0094) |
| `PUT /license/config` | `server_url` | Refused with 400 `bad_url` in a packaged build, which always uses its built-in server (ADR-0097); test-only seam otherwise |
| `GET /tools/explorer`, `POST /tools/explorer/connect` | `host`, `port` | IED Explorer status (warms the worker); connect, then autonomous key-point watch |
| `GET /tools/explorer/devices`, `/nodes?device=`, `/node?ref=` | - | Lazy data model, one level per call |
| `POST /tools/explorer/read`, `PUT /tools/explorer/watch`, `POST /tools/explorer/polling` | `items`, `interval_ms`, `on` | Read, set the remembered watch list, start/stop polling; `GET /tools/explorer/watch.csv` |
| `POST /tools/explorer/operate` | `ref`, `value`, `confirm: true`, `options` | Operate a control via studio_control (ADR-0057); refused without `confirm` |
| `GET /tools/memory`, `PUT /tools/memory/find` | `choice` | Remembered tool settings |
| `GET /activity` | - | What runs outside the IED Run: gateway, master, Explorer, scan, sequences (running dots) |
| `POST /tools/gateway/command` | `point_id`, `value` (bool; Go engine also `"higher"`/`"lower"` for a step, a number for a set point, ADR-0107), `confirm: true` | Page command, run and logged like a SCADA command (ADR-0059) |
| `POST /tools/scan/preview`, `/start`, `/stop`; `GET /tools/scan` | `range`, `port` | Find IEDs (<= 256 addresses) |
| `GET/PUT /tools/gateway/config` | `protocol`, `listen`, `settings`, `real_commands` | Gateway settings (stopped only) |
| `GET/PUT /tools/gateway/points?protocol=`, `POST /tools/gateway/propose` | `points` | Point list with live values; smart proposal |
| `GET/POST /tools/gateway/points.csv` | CSV text (POST) | CSV point list with line diagnostics |
| `GET /tools/gateway/types`; `POST /tools/gateway/candidates`, `/points/from-ied`, `/points/rows`; `GET/POST /tools/gateway/points.xlsx` | see [ADR-0105](../decisions/ADR-0105-gateway-add-points.md) | Go only: type catalog, fitting IED signals, planned/checked rows `{rows, points, ok}` (nothing saved), Excel template and its upload; `points/new` adds `count`/`from`/`name` (a range), `propose` adds `mode`/`current`/`ieds` (smart preview with `summary`) |
| `POST /tools/gateway/start`, `/stop`; `GET /tools/gateway`, `/traffic?after=`, `/commands` | - | Start (points; never Run), status, decoded traffic, command log |
| `POST /tools/gateway/role` | `role` (`gateway`, `master`, `converter`) | Act as station gateway, SCADA master or protocol converter (ADR-0061); refused while running |
| `GET/PUT /tools/master/config`; `GET /tools/master` | `protocol`, `host`, `settings` | SCADA master settings (disconnected only) and status (link, points, probe, poll failures) |
| `POST /tools/master/connect`, `/disconnect`, `/interrogate` | - | Connect (then discover all), disconnect (forgets auto-connect), ask for all values / read now |
| `GET /tools/master/points`; `PUT /tools/master/selection`, `/label` | `unticked`, `key`, `label` | Discovered points with worded quality; untick and name points |
| `GET /tools/master/points.csv`; `POST /tools/master/labels.csv` | CSV text (POST) | Export points; import names |
| `POST /tools/master/command` | `kind`, `address`, `value`, `select`, `code`, `confirm: true` | Command to the device after the REAL-device warning |
| `PUT /tools/master/ranges`, `/routes`; `POST /tools/master/probe`, `/probe/ranges` | `modbus_ranges`, `routes`, `table`, `start`, `end`, `data_type` | Modbus ranges, converter command routes, Find registers (<= 10,000 addresses) and adding what it found |
| `GET /tools/master/traffic?after=`, `/commands` | - | Decoded device traffic, master command log |
| `GET/PUT /tools/master/list`; `PUT /tools/master/list/check-ied`; `GET /tools/master/list/compare` | `points`, `on` | Go only (ADR-0110): the master's own address-only point list, Check against IED switch, expected vs received (Received, Missing, Wrong type, Waiting, Command, Not in your list) |
| `POST /tools/master/list/new`, `/rows`, `/copy-gateway`, `/from-received`; `GET/POST /tools/master/list.xlsx`; `GET /tools/master/list.csv` | a point or `type, count, from, name`; `text`; `keys`; `data` (base64) | Go only: plan points (one, range, pasted/CSV rows, Excel template, from the gateway list, from what arrived) without saving; template and CSV export |

SNMP devices (ADR-0092), Go engine only, under `/tools/snmp` (the Python engine answers 404):

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `GET /tools/snmp`, `/config`; `POST /role` | `role` (`simulate`, `read`) | Status (devices, reader, receiver) and the saved `snmp` section |
| `POST /devices`, `/import`; `PUT/DELETE /devices/{id}` | `profile` (`switch`, `gps`, `empty`); `kind` (`walk`, `csv`), `name`, `text`; `device` | Add from a profile or a file (line diagnostics), change, delete |
| `POST /start`, `/stop` | - | Start/stop all simulated devices; a port clash is explained with a free port |
| `GET/PUT /devices/{id}/points`, `PUT /devices/{id}/value`, `GET /devices/{id}/points.csv` | `points`; `oid`, `value`, `hex` | Values with names; live change while running; CSV export |
| `GET /devices/{id}/actions`, `POST /devices/{id}/trap` | `action` | Events (link down, sync lost...) that change values and send traps/informs |
| `GET /requests` | - | Request log of the simulated devices |
| `POST /read/connect`, `/disconnect`, `/walk`, `/set`, `/save`; `PUT /read/poll`, `/watch`; `GET /read/values` | `target`; `root`; `point`, `confirm`; `name`; `every_ms` (>= 200); `oids` | Read a device: walk all, re-read, write, save as simulated device |
| `POST /traps/listen`, `/traps/stop`; `GET /traps` | `ip`, `port`, `users` | Trap/inform receiver (v1/v2c/v3) and received traps with names |
| `GET/POST /mibs`, `DELETE /mibs/{module}`, `POST /mibs/restore`, `GET /mibs/resolve?name=` | `name`, `text` (MIB file) | MIB library: built-in and added modules, remove, bring back, name to OID |

Packaged builds answer 402 `license_required` on every other `/api/v1` call until licensed
(ADR-0046); development never does. License endpoints return `{error, code}` on failure.

License server owner page (ADR-0072, [spec](../specs/owner-page.md)): under the configured
`<owner path>`, reachable only with the gateway Worker's `X-Admin-Gateway` secret (otherwise 404):
`GET /api/session`, `POST /api/login|logout|logout-all`, `GET /api/overview|licenses|purchases|activity|issues|disk|subscribers|news/status`,
`POST /api/licenses/issue|note|revoke|resend`, `POST /api/subscribers/add|remove`, `POST /api/news/test`;
with a fresh `X-Confirm-Code`: `POST /api/licenses/delete`, `/api/disk/clean`, `/api/news/send`.
Public: `GET/POST /unsubscribe?t=` (POST is RFC 8058 one-click); `POST /api/v1/checkout/create-order` accepts `news`.

Private native IPC is not HTTP: version-1 length-prefixed JSON on worker stdin/stdout.
`studio/backend/studio_app/runtime/worker_protocol.py` owns framing; native worker
capabilities are validated before an IED is marked running.
