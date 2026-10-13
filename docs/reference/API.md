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
| `GET /project` | - | Project snapshot, session, revision, runtime, recovery notice |
| `PUT /project/diagram` | `{diagram,revision,tab_id}` | Stopped-only revision-checked save |
| `POST /project/save` | - | Atomic autosave |
| `GET /project/download` | - | Current `.subsim` ZIP |
| `POST /project/open` | multipart `file` | Stopped-only validated restore/migration |
| `POST /runtime/start` | - | Start all workers after capability/preflight checks |
| `POST /runtime/stop` | - | Stop workers and flush append-only run log |
| `POST /runtime/pause` | - | Toggle automation pause; MMS remains active |

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
| `GET /ieds/{id}/signals` | `q,fc,kind,writable,report,ref,offset,limit` | Filtered/paged typed signals |
| `POST /signals/update` | `{ied_id,ref,value,quality?}` | Acknowledged typed update |
| `POST /signals/source` | `{ied_id,ref,spec}` | Configure validated source |
| `POST /signals/pulse` | `{ied_id,ref,active_value?,duration_ms?}` | Guarded Boolean pulse |
| `POST /signals/release-override` | `{ied_id,ref}` | Resume configured automatic source |
| `POST /controls/operate` | `{ied_id,ref,value}` | Low-level independent native probe operation |
| `POST /equipment/{node}/operate` | `{action}` or `{value}` | Typed mapped equipment command/feedback |
| `GET /equipment/states` | - | Authoritative mapped canvas state snapshot |

Equipment actions are Open/Close for CB/DS/ES, Operate/Reset for Trip Relay and
Press for a Push Button. The server rejects mixed `action` and `value`.

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

Private native IPC is not HTTP: version-1 length-prefixed JSON on worker stdin/stdout.
`studio/backend/studio_app/runtime/worker_protocol.py` owns framing; native worker
capabilities are validated before an IED is marked running.
