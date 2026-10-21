# HTTP and event API

All application routes use `/api/v1`. Administration is loopback-only and requires
the launcher session cookie or bearer token. Development requests also identify the
authoritative frontend with `X-Studio-Frontend`; a stale Vite origin receives 403
with `active_frontend`. JSON errors use `error`, optional `code`, and optional
`details`. Runtime failures return 503; revision conflict returns 409.

## Session, project and runtime

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `POST /session` | `{token}` | Establish same-origin session |
| `GET /project` | - | Project snapshot, session, revision, runtime, `running_ieds`, recovery notice |
| `PUT /project/diagram` | `{diagram,revision,tab_id}` | Stopped-only revision-checked save |
| `POST /project/save` | - | Atomic autosave |
| `GET /project/download` | - | Current `.subsim` ZIP |
| `POST /project/open` | multipart `file` | Stopped-only validated restore/migration |
| `POST /runtime/start` | - | Start all workers after capability/preflight checks |
| `POST /runtime/stop` | - | Stop workers and flush append-only run log |
| `POST /runtime/pause` | - | Toggle automation pause; MMS remains active |
| `POST /runtime/ieds/{id}/start` | - | Run (simulated) or connect (real) one IED; joins a running run (ADR-0067); returns `{state, running}` |
| `POST /runtime/ieds/{id}/stop` | - | Stop or disconnect one IED; the others keep running; the last one stops the run |

## SCL and IEDs

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `POST /scl/validate` | multipart `file` | Offline schema result |
| `POST /scl/preview` | multipart `file` | Selectable IEDs; imports nothing |
| `POST /scl/import` | file + JSON `selected_names` form field | Preserve source; import selected IEDs |
| `POST /ieds/template` | `{name}` | Add laboratory IED template |
| `GET /ieds/{id}/source` | - | Exact retained XML download |
| `POST /ieds/{id}/export-scl` | `{overrides}` | Validate/re-import edited copy; project unchanged |
| `GET /ieds/{id}/removal-preview` | - | Single-delete impact |
| `DELETE /ieds/{id}` | `{confirm:true}` | Confirmed single deletion with recovery archive |
| `POST /ieds/removal-preview` | `{ids}` | Bulk impact and fingerprint |
| `POST /ieds/remove` | `{ids,fingerprint,confirm:true}` | Fingerprint-checked bulk removal |

## Signals, controls and equipment

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `GET /ieds/{id}/signals` | `q,fc,kind,writable,report,ref,ld,bad_quality,in_dataset,offset,limit` | Filtered/paged typed signals and the IED's logical `devices` (read-only filters, change 0093) |
| `POST /signals/update` | `{ied_id,ref,value,quality?}` | Acknowledged typed update |
| `POST /signals/source` | `{ied_id,ref,spec}` | Configure validated source |
| `POST /signals/pulse` | `{ied_id,ref,active_value?,duration_ms?}` | Guarded Boolean pulse |
| `POST /signals/release-override` | `{ied_id,ref}` | Resume configured automatic source |
| `POST /controls/operate` | `{ied_id,ref,value}` | Low-level independent native probe operation |
| `POST /equipment/{node}/operate` | `{action}` or `{value}` | Typed mapped equipment command/feedback |
| `GET /equipment/states` | - | Authoritative mapped canvas state snapshot |
| `POST /equipment/{node}/test` | `{action, value?, row?}` | Test button on a running simulated IED: writes the bound point via the sample writer, sends no command; real IEDs refused (ADR-0067) |

Equipment actions are Open/Close for CB/DS/ES, Operate/Reset for Trip Relay and
Press for a Push Button. The server rejects mixed `action` and `value`.

## Network, GOOSE and real IEDs

| Method/path | Input | Result / invariant |
| --- | --- | --- |
| `GET /network` | - | Network mode and Npcap status |
| `PUT /network` | `{mode,adapter_guid?,adapter_label?}` | Stopped-only; endpoint conflicts refused |
| `GET /network/adapters` | `refresh=1` | Windows adapters with IPv4 addresses (cached 10 s) |
| `POST /network/plan/preview` | `{adapter_guid,strategy,manual?}` | Address rows with fix hints (a row that cannot be used adds `alternative: {ip, prefix}`, change 0098), change items, fingerprint; Go also plans SNMP devices (`ied_id` `snmp:<id>`, `kind`, `saved`, item `snmp_firewall`, change 0148) |
| `GET /network/plan/progress` | | Go only (ADR-0093): the running address check: `run`, `running`, `steps[{key,label,state,ms}]`, `asked`, `answered`, `rows[{id,name,address,state}]` with state `checking`/`free`/`in_use`/`not_checked` |
| `POST /network/plan/apply` | `{fingerprint,confirm:true}` | Re-plan; 409 if changed; elevated helper; enters lab mode |
| `GET /network/cleanup/preview` | - | Outstanding owned items and fingerprint |
| `POST /network/cleanup/apply` | `{fingerprint,confirm:true}` | Remove owned items; back to loopback mode |
| `GET /network/gateway-firewall` | - | Gateway firewall check: items, fingerprint, `open`, `blocked`, `needs_admin` (no elevation) |
| `POST /network/gateway-firewall` | `{fingerprint,confirm:true}` | Allow rule + switch off Block rules for the gateway worker; 409 if Windows changed (ADR-0062) |
| `PUT /ieds/{id}/settings` | `{role?,live?,lab_ip?,goose_publish?,port?}` | Stopped-only; SCL never rewritten |
| `GET /goose` | - | Configured/observed GOOSE blocks (`origin` app/wire, `src_mac`), problems, unknown GOOSE, `engine` (ADR-0073) |
| `GET /goose/share` | - | Where GOOSE goes: `reach`, picked `uplink`, Advanced `choices`, `risk` {level, reasons}, `npcap`, `state`, `message`, `conflicts` |
| `PUT /goose/share` | `{reach, uplink_guid?, adapter_guid?, confirm?}` | Allowed while running. `outside` needs `confirm:true`, Npcap and the `adapter_guid` shown (GOOSE-19) |
| `POST /goose/conflicts` | `{ied_id, choice: stop\|continue, confirm?}` | Answer a conflict for this Run; `continue` needs `confirm:true` |
| `GET /goose/links` | - | GOOSE link map (ADR-0065): `ieds`, `blocks` (status, `member_refs`), `links` (status, inputs, `live`), `diagnostics`, `summary` |
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
| `GET /license/status` | - | `licensed, mode, plan, days_remaining, expires_at, reason, hwid, server_url, masked_key` |
| `POST /license/trial` | - | Start the one trial via the license server; saves `license.cert` |
| `POST /license/activate` | `license_key`, `move` | Activate (or move) a lifetime key online |
| `POST /license/activate-offline` | `certificate` text | Verify and save a pasted/loaded `license.cert` |
| `GET /app/update` | - | `current, auto, checked_at, checking, error, available, important, dismissed, latest` (ADR-0071; stored answer, no internet call) |
| `POST /app/update/check` | - | Ask the license server now (a repeat within 60 s reuses the answer); failures are an `error` code, never a 5xx |
| `PUT /app/update` | `{auto?: bool, dismissed?: version or null}` | Once-a-day switch; hide the note for one version (not for an important update) |
| `GET /preferences` | - | `tour_seen_version`, `tour_version`, `list_filters`, `sld_goose_layer`, `sld_readers_layer` |
| `PUT /preferences` | `{tour_seen_version?, list_filters?: {list: {key: text or bool}}, sld_goose_layer?, sld_readers_layer?}` | Store UI preferences in the data folder; a list's filters replace only that list (change 0093); layers 0 or 1 (change 0094) |
| `PUT /license/config` | `server_url` (https) | Change the license server address |
| `GET /tools/explorer`, `POST /tools/explorer/connect` | `host`, `port` | IED Explorer status (warms the worker); connect, then autonomous key-point watch |
| `GET /tools/explorer/devices`, `/nodes?device=`, `/node?ref=` | - | Lazy data model, one level per call |
| `POST /tools/explorer/read`, `PUT /tools/explorer/watch`, `POST /tools/explorer/polling` | `items`, `interval_ms`, `on` | Read, set the remembered watch list, start/stop polling; `GET /tools/explorer/watch.csv` |
| `POST /tools/explorer/operate` | `ref`, `value`, `confirm: true`, `options` | Operate a control via studio_control (ADR-0057); refused without `confirm` |
| `GET /tools/memory`, `PUT /tools/memory/find` | `choice` | Remembered tool settings |
| `GET /activity` | - | What runs outside the IED Run: gateway, master, Explorer, scan, sequences (running dots) |
| `POST /tools/gateway/command` | `point_id`, `value` (bool), `confirm: true` | Page command, run and logged like a SCADA command (ADR-0059) |
| `POST /tools/scan/preview`, `/start`, `/stop`; `GET /tools/scan` | `range`, `port` | Find IEDs (<= 256 addresses) |
| `GET/PUT /tools/gateway/config` | `protocol`, `listen`, `settings`, `real_commands` | Gateway settings (stopped only) |
| `GET/PUT /tools/gateway/points?protocol=`, `POST /tools/gateway/propose` | `points` | Point list with live values; smart proposal |
| `GET/POST /tools/gateway/points.csv` | CSV text (POST) | CSV point list with line diagnostics |
| `POST /tools/gateway/start`, `/stop`; `GET /tools/gateway`, `/traffic?after=`, `/commands` | - | Start (points; never Run), status, decoded traffic, command log |
| `POST /tools/gateway/role` | `role` (`gateway`, `master`, `converter`) | Act as station gateway, SCADA master or protocol converter (ADR-0061); refused while running |
| `GET/PUT /tools/master/config`; `GET /tools/master` | `protocol`, `host`, `settings` | SCADA master settings (disconnected only) and status (link, points, probe, poll failures) |
| `POST /tools/master/connect`, `/disconnect`, `/interrogate` | - | Connect (then discover all), disconnect (forgets auto-connect), ask for all values / read now |
| `GET /tools/master/points`; `PUT /tools/master/selection`, `/label` | `unticked`, `key`, `label` | Discovered points with worded quality; untick and name points |
| `GET /tools/master/points.csv`; `POST /tools/master/labels.csv` | CSV text (POST) | Export points; import names |
| `POST /tools/master/command` | `kind`, `address`, `value`, `select`, `code`, `confirm: true` | Command to the device after the REAL-device warning |
| `PUT /tools/master/ranges`, `/routes`; `POST /tools/master/probe`, `/probe/ranges` | `modbus_ranges`, `routes`, `table`, `start`, `end`, `data_type` | Modbus ranges, converter command routes, Find registers (<= 10,000 addresses) and adding what it found |
| `GET /tools/master/traffic?after=`, `/commands` | - | Decoded device traffic, master command log |

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
`POST /api/licenses/issue|note|revoke|reset-hwid|resend`, `POST /api/subscribers/add|remove`, `POST /api/news/test`;
with a fresh `X-Confirm-Code`: `POST /api/licenses/delete`, `/api/disk/clean`, `/api/news/send`.
Public: `GET/POST /unsubscribe?t=` (POST is RFC 8058 one-click); `POST /api/v1/checkout/create-order` accepts `news`.

Private native IPC is not HTTP: version-1 length-prefixed JSON on worker stdin/stdout.
`studio/backend/studio_app/runtime/worker_protocol.py` owns framing; native worker
capabilities are validated before an IED is marked running.
