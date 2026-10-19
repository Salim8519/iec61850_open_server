# Stream D: live acquisition (P2)

Part of the [Go engine workstreams](../workstreams.md). Branch `go-d-live`, cut from `go-engine`
at c983a8f. Owns `studio/engine/internal/acquisition` and `studio/engine/internal/api/liveapi`.

## What is ported

| Go package | Python source | Notes |
|---|---|---|
| `acquisition/pyv` | (helpers) | Python dict/list semantics over `pyjson` trees: truthiness, `==` (1 == 1.0 == True), `repr`/`str`, `{:04X}`, deep copy, ISO seconds, `str.casefold` (Unicode 15.0, checked on every code point) |
| `acquisition/livedom` | domain/live_value, model_compare, access_policy, mms_access | Python `int()` on text and values (`ParseIntText`, `PyInt`) |
| `acquisition/modes` | domain/health_reads, test_mode, mode_badges | |
| `acquisition/health` | domain/goose_consistency, health_model_checks, station_health, health_export | CPython 3.12 `fromisoformat` for the stale check |
| `acquisition/goosedom` | domain/goose_supervision, goose_reach, goose_inputs, goose_link_map, goose_link_live; networking/adapters `normalize` | sha256 link ids as Python |
| `acquisition/svdom` | domain/sv_supervision | int vs float results kept |
| `acquisition/signalindex` | api/signals `list_signals`, scl/report_catalogue `report_members` | index built once per immutable IED model; Werkzeug `request.args` parsing |
| `acquisition` | acquisition/* (manager, real_session, live_values, health_checks, health_snapshot, goose_engine, goose_monitor, goose_inputs, goose_share, sv_monitor), simulation/mms_access, simulation/test_mode_apply | services behind small interfaces (`ports.go`) |
| `api/liveapi` | api/live, api/health, api/goose_share, api/signals (GET list), api/test_mode | 17 routes as `[]api.Route` |

Routes: `GET /live`, `GET /live/{key}/comparison`, `GET /live/{key}/comparison.csv`,
`POST /live/{key}/read`, `GET /goose`, `GET /goose/links`, `GET /mms/access`, `GET /sv`,
`GET|PUT /goose/share`, `POST /goose/conflicts`, `GET /health`, `POST /health/refresh`,
`GET /health.csv`, `GET /health/report.html`, `GET /ieds/{ied_id}/signals`,
`PUT /ieds/{key}/test-mode` (all under `/api/v1`).

## Go rules applied

- Immutable model, no copies: health reads a per-model cached view (all IED keys except
  `source`/`cfg`, signals = Mod/Beh/Sim only), built once per model object; Python deep-copied
  every IED on each poll (change 0110 reduced it). Signal search, ref lookups and mode views are
  indexes keyed by the model object pointer and rebuilt only when the model object changes.
- No lock across I/O: sessions, monitors and the engine hold their mutex only for state; native
  requests run outside it. The engine keeps Python's two locks (`calls` for requests, `mu` for
  state updated by events) so events are never blocked by a waiting request.
- Persistent session per real IED (one studio_live process per role, as Python; studio_live is
  single-threaded, so roles are not shared). The session status is published as copies.
- Discovery cache keyed by device identity: not added. A session discovers once per connection
  (as Python); the comparison stays on the session. A cache across reconnects would change what
  the user sees after a device reconfiguration, so it needs an owner decision (open question 1).
- Outward events are identical to Python: one `signal` event per received point (or its owner),
  even when the value did not change. `Values.SetLive` is where stream C's value hub can detect
  real changes for internal consumers (emit on change only internally).

## Evidence

All with `GOMAXPROCS=4`, `CGO_ENABLED=0`, `go test -p 2`.

- `go vet ./internal/acquisition/... ./internal/api/...`: clean. `go test`: all packages pass.
  `python studio/tools/verify.py`: PASS (file lengths, architecture, docs, history, specs, Go).
- Golden files recorded from the Python functions (generators in `acquisition/testdata/gen`,
  outputs in `testdata/generated`, public SCDs `scd/model_substation.scd`,
  `scd/open_substation.scd` and synthetic inputs only), zero differences:
  goosedom + svdom 1,078 cases; livedom 2,088 (value 1,889, compare 11, access 111, MMS 6,
  `int()` 71); modes + health 423 (compared in `json.dumps` insertion order, stricter than
  Flask); signal search 546 recorded Flask HTTP answers; casefold every code point.
- Route parity (`liveapi/parity_test.go`, generator `gen_routes.py`): the real Flask app on the
  model substation, stopped and running. Running uses fake native workers whose conversation is
  recorded (22 requests) and replayed to Go, a fixed clock, FEED1 as a real device, an SV stream,
  and injected GOOSE/SV/engine/MMS events. 48 HTTP answers (status, Content-Type,
  Content-Disposition, body bytes) and the events of every step are identical (the Run start
  events compared as a set, since session and listener events interleave by thread).
- Real IED with an independent client (`acquisition/real_ied_test.go`, opt-in
  `STUDIO_REAL_IED=1`, `STUDIO_PYTHON=<venv python>`): a stand-in `studio_ied` device on
  127.64.0.31:20131 (`testdata/rig/stand_in.py`), Go session through the real `studio_live.exe`
  with a test-only framed client. Discover + compare (135 points match), free report block
  `EventsRCB` reserved (the independent studio_live client sees it enabled/reserved, not free),
  the rest polled, a value written in the device (42.5) arrives and equals the independent read,
  health verdict ok, refresh without failures, and after Stop the independent client sees the
  block free again. Start to live: Go 123 ms; Python `RealSession` on the same rig 138-340 ms.

### Timings (same running state, median per request; Python via the Flask test client, Go via `httptest`)

| Route | Python | Go |
|---|---|---|
| /live | 436 µs | 118 µs |
| /goose | 445 µs | 89 µs |
| /goose/links | 977 µs | 42 µs |
| /mms/access | 402 µs | 16 µs |
| /sv | 521 µs | 32 µs |
| /health | 1905 µs | 214 µs (gate ≤ 5 ms) |
| /health.csv | 1696 µs | 192 µs |
| /health/report.html | 1876 µs | 300 µs |
| /goose/share | 695 µs | 52 µs |
| /live/{id}/comparison | 2112 µs | 854 µs |
| /ieds/{id}/signals (limit 100) | 1463 µs | 599 µs |
| signal search, 90,000 points | 79.6 ms (view) / 107.6 ms (HTTP) | 4.3 ms (5.7 ms with encoding) |

Reproduce: `STUDIO_BENCH=1 python gen_routes.py` and
`go test -run XXX -bench Routes ./internal/api/liveapi`; `go test -bench Search ./internal/acquisition/signalindex`.

## Deviations (none visible with importer-built projects)

1. Watched points: Python collects them in a set, so their order (and which 500 survive)
   follows per-process string hashing; Go uses the first-seen order (diagram mappings, relay
   statuses, test panel). It shows in `/live` `polling` and the poll request order only.
2. Connecting an IED that already has a session stops the replaced session (Python overwrote
   the dict entry and left its thread running).
3. A lost connection ends the poll wait at once (Python finished the poll interval first).
4. Python exceptions on malformed internal data (missing keys, wrong types in the model or
   native events) are not reproduced; Go reads them as None and degrades. The model comes from
   the importer and native events have fixed schemas, so users cannot reach these paths.
   Request-body crash paths ARE reproduced (500 for a truthy non-object body, unhashable refs).
5. A non-ValueError from the sample writer during GOOSE input routing: Python's exception left
   the event consumer; Go skips only that route.
6. `DescribeMMS(4.0)` returns "integer" like Python's hash lookup.

## Integration points (code against `acquisition/ports.go`)

- Stream A: mount `liveapi.Service.Routes()`; `Emitter` = coordinator.emit (bus + run log),
  `Publisher` = bus only (`mms_access`); `liveapi/respond.go` mirrors the Flask error bodies and
  can be replaced by httpapi's shared writer; `Version`; `pycompat.Casefold` may replace
  `pyv.Casefold` (same golden).
- Stream B: `Projects` (immutable `Current()`, `FindIED` with `Coded("Unknown IED","not_found")`,
  `RunID` = session-revision, `SetTestMode`, `SaveReach` running `validate_network`);
  `liveapi.Declarations` (= `read_goose_sources` cached per source set, layout in
  `goosedom.Declarations`); `SubscriptionMembers` (`_with_members` for pre-0094 projects);
  the IED runtime `state` field through `IEDStates`.
- Stream C: `StartWorker` (framed client: `Request` errors are `apperr.Runtime`, timeouts
  implement `Timeout() bool` with "Native <op> request timed out", events delivered in pipe
  order, synthetic `worker_failed`); `Values` (the runtime table; it must keep the last values
  and `live_state` after Stop, because Python's project signals keep them and `/signals` shows
  them); `Samples` (`Running` = in `sample_writer.locks`, ValueError kind for refusals);
  `Equipment.PublishChanged`; coordinator calls `Manager.Start/Connect/Disconnect/Stop`,
  `MmsAccess.Reset/Forget/MarkOwn/Control/Update`, `TestModes.ApplyAll`, `GooseEngine.BusFor`.
- Stream F: `Network` (adapters as normalized dicts, `npcap_status`); parity harness may reuse
  `gen_routes.py`/`parity_test.go`.

## Open questions

1. Discovery cache across reconnects keyed by device identity (Go rule): what identifies a
   device (address + LD names + model hash?) and when is the cache invalid? Not implemented.
2. `/signals` for a stopped IED after a Run: depends on stream C keeping the last values.

## Proposed ADR text (for the integrator, number 0077+)

"Live acquisition in Go. Real-IED sessions, GOOSE/SV monitors, MMS access, health, test mode
and the signal index run in `internal/acquisition` behind interfaces to the project actor,
runtime values, sample writer, event bus and native framing. REST bodies and events stay
byte-identical to Python (golden and replayed-scenario tests); health reads the immutable
model through cached per-model views; signal search uses a per-model index. Compatibility:
no API, event, file or protocol change."
