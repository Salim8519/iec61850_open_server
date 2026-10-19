# Stream E: tools (P4)

Branch `go-e-tools`. This stream ports the frozen Python tools to Go under `studio/engine`. It covers the IED Explorer, Find IEDs (scan), tool memory, the test panel, CSV traces, the station gateway, the SCADA master and the protocol converter.

The port must behave byte-identically. The only deliberate change is the speed of the gateway value push and of command confirmation. The packages are:

- `internal/tools/...`
- `internal/gateway` (domain rules)
- `internal/gateway/service` (runtime)
- `internal/api/toolsapi` (routes)

## What exists

| Package | Python counterpart | Contents |
|---|---|---|
| `tools/pytext`, `tools/pycsv` | CPython semantics | `str`/`repr`/`format(.6g)`, `int()`/`float()` parsing with exact error texts, `==` across int/float/bool, the 3.12 `_csv` reader state machine and the `QUOTE_MINIMAL` writer |
| `tools/worker` | `native/worker_client.py` | Framed worker client: 4-byte LE length + JSON, version 1 envelope, ordered events, `worker_failed` once, Python's timeout/`RuntimeError` texts; `CREATE_NO_WINDOW` |
| `tools/numlog` | traffic/command logs | Numbered logs with `after`/`limit` (traffic 1000, commands 200) |
| `tools/control` | `control_client.py` | `studio_control` operate client and options |
| `tools/memory` | `tool_settings.py` | `tool-settings.json` byte-identical (`json.dumps(indent=2)`, CRLF), atomic `.tmp` + rename |
| `tools/scan` | `scan_service.py`, `scan_plan.py` | Range rules (Python `ipaddress`, at most 256 hosts), preview/start/stop/status, results in address order |
| `tools/explorer` | `explorer/*.py` | Tree/value/quality/time text, watch limits (16 nodes, 40 points), connect/browse/read/poll/operate through `studio_live` and `studio_control`, activity |
| `tools/trace` | `traces.py` | CSV traces (5 MB, 100000 rows), `utf-8-sig` decoding with Python's `UnicodeDecodeError` texts, value validation |
| `tools/testpanel` | `test_panel.py` | Panel and mapping validation, report members |
| `gateway` | `domain/gateway_*.py`, master/converter rules | Proposal, 5000-point limit, settings, CSV import/export texts, master ranges/routes/labels/probe, converter mapping |
| `gateway/service` | `gateway/manager.py`, `commands.py`, `converter.py`, `master_manager.py`, `modbus_probe.py` | Gateway, master and converter processes; command flow; Modbus probe |
| `api/toolsapi` | `api/tools.py`, `gateway.py`, `gateway_master.py`, `test_panel.py` (panel), `traces.py` | `[]api.Route` with Python's paths, sorted-key Flask JSON, error texts, CSV download headers |

## Decisions

1. **Gateway value push is change-driven.**
   - Python diffs all points ten times a second. Go subscribes to the value hub (`Runtime.Subscribe`) instead.
   - A change marks only the points whose value, quality or timestamp reference changed. IED-state and run-state changes mark that IED's points, or all of them.
   - Batches still leave at most once per 100 ms, so SCADA sees the same spontaneous events. Updates are still sent 500 per request.
   - A 10 s full sweep is a safety net for edits the hub does not report. Quality and timestamp dependencies are rebuilt only at start and on run-state changes.
2. **Command confirmation waits on the hub.**
   - It keeps Python's 3 s deadline and checks the state at once.
   - After that it re-checks on every hub change of that IED, instead of polling every 100 ms.
   - The texts are unchanged: "is now closed (ON)", "did not confirm", and so on.
3. **Small interfaces instead of imports** from other streams. See "Needs from other streams" below.
4. **A stand-in for the worker framing.**
   - `tools/worker` is a stand-in for stream C's native framing. Its `Process` interface (`Request`, `Alive`, `Close`) and `Spawner` are the swap point.
   - Its behaviour matches `worker_client.py`: event order, `worker_failed` once, reply errors become `RuntimeError`, timeouts become `TimeoutError` (Runtime, 503).
5. **Local copies of Python text rules.** `pytext`/`pycsv` are local copies until stream A's shared pycompat exists. They have their own oracle tests and can be replaced one function at a time.

## Evidence (all on 2026-10-20)

**Test conditions**
- Settings: `GOMAXPROCS=4`, `go test -p 2 -trimpath`, `STUDIO_PYTHON=studio/.venv/Scripts/python.exe`.
- `go vet` is clean on all stream E packages, including under `GOOS=linux` for the service package.
- Every package passes.

**Oracle parity: Go output vs the frozen Python functions** (`Dumps` bytes compared, synthetic seeded inputs)

| Area | Cases |
|---|---|
| pytext | int/float parsing, repr, `.6g`, equality |
| pycsv | reader/writer, including malformed quoting and the field limit |
| memory | 400 cases (clean + indent bytes) |
| scan plan | 6000 ranges |
| traces | about 10000 cases (rows, values, UTF-8 errors) |
| explorer domain | node views, value/quality/time text, watch merge |
| test panel | entries and mappings |
| gateway domain | about 51 s of cases: propose, validate, CSV export/import, master rules, converter mapping, probe ranges |

**Route parity** (`api/toolsapi/routes_parity_test.go`)
- About 190 requests in three scenarios: gateway, master and tools. The same requests go to the frozen Flask app and to the Go routes.
- Status, body bytes, `Content-Type` and `Content-Disposition` must match exactly.

**Wire tests** (real native workers on the stream's own loopback/ports)

| Test | Setup | What it covers |
|---|---|---|
| Explorer | `studio_ied` (ExploreTest template) on 127.65.0.32:20132 | connect, browse, read, watch/poll through `studio_live`, operate through `studio_control`, and a scan of the stream's own range |
| Gateway IEC 104 | `studio_gateway` on 127.0.0.1:20404 with `studio_test_iec104_master` | general interrogation, select/execute → COT 7/7/10, spontaneous update after a hub change, the "is now closed (ON)" command log |
| Master runtime | `test_master_runtime.py` ported through HTTP, device on port 20411 | role switch, config validation, connect, labels/selection, refused and confirmed commands, disconnect |

**Gateway CPU at 5000 points** (`gateway/service/bench_test.go`, `STUDIO_GATEWAY_BENCH=1`, 20 s windows)

| Changes/s | Python `GatewayManager` | Go gateway (exact cycles) | Go (tick-based process times) |
|---|---|---|---|
| 0 | 12.6 to 19.0 % of one core | 0.03 % | 0.00 % |
| 1000 | 14.7 to 19.1 % | 0.49 to 0.64 % | 0.62 % (1.17 to 1.33 % in earlier runs) |

How the benchmark works:
- Both sides run the same 5000 points, half floats and half singles, against the real `studio_gateway`. The worker's own CPU is excluded on both sides.
- The Go figure subtracts a harness baseline: the fake hub and the value changer running without a gateway.
- The Python figure includes its changer thread. Most of the Python cost is the 10 Hz diff over all points.
- On Windows, `GetProcessTimes` counts in 15.6 ms ticks, which is too coarse at about 10 ms/s. The gate value therefore uses `QueryProcessCycleTime`, calibrated on a spinning thread. Both figures are logged.
- A CPU profile with blocked syscalls removed shows Go-level work of about 0.5 % of samples. The rest is pipe I/O and scheduler wake-ups.
- The P4 gate (≤ 1 %) holds by exact cycle count. Tick-based figures straddle 1 % at 1000 changes/s and are noisy.

## Known differences from Python

| Difference | Python | Go |
|---|---|---|
| Value push timing | every point diffed each 100 ms | changed points only, still at most one batch per 100 ms. A change the hub does not report is picked up by the 10 s sweep, where Python took 100 ms. Stream C's hub must therefore report every value, quality, timestamp, IED-state and run-state change. |
| Command confirmation | sees a state up to 100 ms late | sees it on the hub notification; same deadline and texts |
| Test binaries | n/a | must be built with `go test -trimpath`, because the antivirus blocks some untrimmed test executables ("Access is denied") |

`POST /signals/pulse` and `/signals/release-override` are not in this stream; they belong to stream C's signal routes.

## Needs from other streams

**Stream C (runtime, value hub, native framing)**

`gateway/service.Runtime`:
- `RunState()`.
- `Signal(iedID, ref) (Signal, bool)` with value, quality ref, timestamp ref and live state.
- `Endpoint(iedID)`, where an IED that is not running returns the ValueError "IED is not running".
- `Emit(kind, payload)`.
- `Subscribe(func(Change)) cancel`. A `Change{IEDID, Ref}` is either:
  - one value, quality or timestamp reference;
  - `Ref == ""`: an IED's state or all of its values;
  - `IEDID == ""`: the run state, meaning everything.
  `notify` must be cheap and must never block the hub.

`toolsapi.Runtime`:
- `Signal(iedID, ref) (*pyjson.Object, error)`.
- `Source(iedID, ref)`.
- `Stopped(fn)`: refuses with "Stop the simulation before editing its structure" unless the simulation is stopped, then runs `fn` under the cycle lock.
- `RunningSequences()`.

Framing: swap `tools/worker` for `internal/native` behind `worker.Process`/`worker.Spawner`, or adapt it. The tests rely on events arriving in order on the reader goroutine.

**Stream B (project service)**

`service.Projects`:
- `IEDs()` (immutable snapshot).
- `Gateway()`.
- `SaveGateway(config) error`: store, save and `changed("gateway")`; keep the old section if the save fails.

`toolsapi.Projects` adds `TestPanel()` and `SaveTestPanel(entries) error`.

**Stream A (composition, HTTP)**

- Build `toolsapi.Deps`:
  - `explorer.NewService`
  - `scan.NewService`
  - `memory.New`
  - `service.NewManager`
  - `service.NewMaster`
  - converter wiring
- Mount `Area.Routes()`.
- At startup, call `Master.Autoconnect()` first and then `Manager.Autostart()`, as `app_factory.py` lines 118-119 do.
- Give the gateway firewall service (ADR-0062, networking) the manager's executable path.
- `toolsapi/respond.go` maps errors itself, following `app_factory.py`. Swap it for the shared error writer when it exists. The body format is identical.

## Proposed ADR text

> **Change-driven gateway value push.** The Go gateway service updates `studio_gateway` from value-hub change notifications instead of diffing all points ten times a second, and confirms commands on hub notifications instead of 100 ms polling.
>
> - *Need:* the P4 budget of ≤ 1 % CPU at 5000 points; Python used 13 to 19 % of a core.
> - *Compatibility:* points, batches (≤ 500, ≤ one per 100 ms), spontaneous events, command texts and deadlines are unchanged.
> - *Requirement:* the hub must report every value, quality, timestamp, IED-state and run-state change.
> - *Safety net:* a 10 s full sweep covers anything missed.

## Not done

- Integration with the real stream C hub and framing, the stream B project service and the stream A composition. The interfaces are above; everything is tested with fakes and real native workers.
- The gateway firewall and network routes are not ported here (networking stream).
- No ledger, `INDEX.md` or `docs/changes` entries: those files are shared, and the coordinator owns them.
