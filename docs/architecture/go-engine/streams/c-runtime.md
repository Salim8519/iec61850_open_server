# Stream C: runtime (P1)

Branch `go-c-runtime` (cut from `master` c983a8f, which holds the go-engine plan). Owner of
`studio/engine/internal/native`, `internal/runtime`, `internal/api/runtimeapi` and the native
multiplexed mode in `studio/native/control`. Python stays frozen and the shipped default.

## What exists

| Package | Content |
|---|---|
| `internal/native` | Framing (4-byte LE + JSON ≤ 16 MiB, `pyjson.Decode`, Python texts), `Worker` with WorkerProcess semantics (uuid4 ids, 10 s default, `ok:false`→503, "Native worker exited: N", "Native <op> request timed out", `worker_failed` only when not closing, stderr → `native_diagnostic`, stop/3 s/kill close), unbounded ordered inbox (reader never blocks; events reach the consumer before any later reply), CREATE_NO_WINDOW, one kill-on-close Job Object for every worker; typed clients `Ied`, `Live`, `Goose`, `Control` (+ `CanvasOperate`) |
| `internal/runtime/pyval` | Python `==`, `str()`, truthiness, `int()`/`float()` with their texts |
| `internal/runtime/signal` | `validate_value`, sample companions, `validate_source`, CSV traces (port of CPython's `_csv` reader), generators with CPython `random.Random` (MT19937, `init_by_array`, abs seed) |
| `internal/runtime/plant` | Equipment command/feedback rules, display states, energisation, switching hints, test buttons |
| `internal/runtime` | `Coordinator`: Run/Stop/Pause, single-IED add/remove, parallel worker start and stop, per-IED actor goroutines owning the pipe, value tables by point id (kept across runs like the Python signal dicts), sample writer, automation (0.1 s cadence, same arithmetic, 80-point batches, IEDs in parallel), pulses as timers, scenarios, equipment operate / push buttons / test buttons, native control feedback, energisation only after a mapped state point was written, MMS access, run log on its own goroutine, endpoint assignment/preflight/conflicts |
| `internal/api/runtimeapi` | 13 routes as `[]api.Route`: `/runtime/{action}`, `/runtime/ieds/{id}/{action}`, `/equipment/states`, `/equipment/{node}/operate`, `/equipment/{node}/test`, `/scenarios/runs`, `/scenarios/{key}/start`, `/signals/update`, `/signals/source`, `/signals/pulse`, `/signals/release-override`, `/controls/operate`, `/project/test-panel` |
| `studio/native/control` | `canvas_operate` (`control_mux_v1`): one lane thread and association per endpoint, replies by request id, the probe's control sequence |

## Evidence (2026-10-20, this laptop, workers on 127.63.x.y ports ≥ 21000, temp data dirs)

- `go vet` clean; all six packages pass (`gofmt` clean). Zero `studio_*.exe` left after the runs.
- Protocol tests against the real workers: studio_ied start/update/update_batch/read_versioned/
  update_guarded (stale guard text), timeout and killed-worker texts, studio_live hello/connect/read/
  disconnect, studio_goose hello/open/stats, Job Object membership.
- Canvas operate vs `studio_probe control` on the same IED (`TestCanvasOperateMatchesProbeOnTheWorker`):
  same result bytes (`{"ok":true}`), same worker events (connection, control_requested,
  control_applied, connection), same MMS reads (1, `BayACTRL/XCBR1`) and writes
  (`BayACTRL/XCBR1.Pos`) counted by the worker's access log, for close and open. No packet
  capture tool is installed; the worker-side view is the PDU evidence.
- Events parity (`testdata/record_python.py` vs `TestScriptedRunWithRealWorkers`, compared by
  `testdata/compare_events.py`): run, CB close and DS open on the canvas, manual write, pulse,
  scenario with a failing assert, ES test button, latching button, stop and start one IED, stop,
  close. **Identical** per IED and globally (60/9/26 events) after normalising ids, timestamps,
  peers and correlation ids; generator signals, the worker's own buffered-report timer,
  `automation_late` and `mms_access` are excluded as wall-clock dependent; runs of events Python
  emits from set iteration (initial snapshot signals, equipment states of one write) are compared
  as multisets because Python's own order is random per process.
- HTTP answers: 30 refusal/idle requests (`testdata/stopped_requests.json`) byte-identical to the
  Flask app (`record_flask.py` golden) including 404/415/400 Werkzeug texts, KeyError quoting,
  StudioError bodies.
- CPython vectors: generators 10,000 rows (noise bit-identical incl. 2**100 and negative seeds;
  ramp/step/constant exact; sine differs in the last float64 bit in 147 of 1000 rows because Go's
  `math.Sin` is not the MSVC CRT `sin`, all equal after FLOAT32 rounding); CSV traces 72 rows
  (spec bytes, error texts, played values) identical; energisation + hints 729 position sets
  identical bytes.
- Run log: same record bytes (`run_id` first, compact, ASCII escapes), manifest/runs lines,
  `isoformat()` timestamps, 10 MiB segments, flush after 100 signals or any other event.

### Operate timing (synthetic two-IED template project, 30 CB operations each)

| Engine | done p50 / p95 | symbol p50 / p95 |
|---|---|---|
| Python (studio_probe per click, save of the small project) | 182 / 432 ms | 177 / 426 ms |
| Go, `CommandKeep` 0 (association per command, events identical) | 51 / 81 ms | 49 / 78 ms |
| Go, `CommandKeep` 10 s (association kept) | 3.6 / 19.7 ms | 1.7 / 16.2 ms |

The owner's 1.25 s was with a 28.5 MB project whose full save Go no longer does. With the
association closed after each command (probe-identical) most of the time is the simulated IED's
MMS server accepting a new association (~40 ms, single-threaded server; native item in the plan),
so the P1 budget (symbol ≤ 20 ms, done p95 ≤ 50 ms) is met only with a kept association.

## Decisions taken here (proposed ADR text below)

1. Values live in per-IED tables by point id, outside the project tree, and survive Stop like the
   Python signal dicts did; a table is rebuilt when the IED's signal list changes.
2. One actor goroutine per simulated IED owns its pipe; sample writes, pulses and reads of that IED
   run there in order. The coordinator mutex is never held across I/O. Lifecycle (Run/Stop/Pause)
   and the cadence tick exclude each other (as `cycle_lock` did for them); updates, operate and
   pulses do not wait for the tick.
3. A narrow `commandMu` orders a canvas command before the native feedback of that command (the
   order Python got from `cycle_lock`); nothing else waits on it.
4. Workers start and stop in parallel; on a failed batch the first error in project order is
   returned, as the serial loop did.
5. Canvas commands go through one warm `studio_control` in `control_mux_v1` mode, with
   `CommandKeep` = 0 by default so connection events and the MMS access list stay exactly as
   with `studio_probe`. Raising it is an owner decision (below).
6. The CmdTerm wait in the mux is event-driven with the probe's 2 s limit (no 20 ms polling).

### Proposed ADR (0077): runtime actors and value tables

Context: 0111 rules 1, 3, 6, 9. Decision: items 1-4 above; events keep Python's kinds, fields and
per-IED order (one `signal` event per written point, no batching). Compatibility: none visible;
events of different IEDs may interleave differently; Python's set-order randomness becomes a
stable order.

### Proposed ADR (0078): multiplexed MMS command client

Context: one `studio_probe.exe` per canvas click (0111 §1). Decision: `studio_control` gains
`canvas_operate` with capability `control_mux_v1`: lanes per endpoint (thread + association),
replies by request id, exactly the probe's sequence (ControlObjectClient_create per command, no
originator, interlock/synchro/test off, SBO normal/enhanced by ctlModel, success only with no
error and AddCause unknown) and its result `{"ok": bool}`; `keep_ms` 0 releases the association
before replying. Compatibility: none with `keep_ms` 0. With `keep_ms` > 0 the user would see
ScadaBench as a permanent MMS client row of the IED and fewer `connection` events, in exchange for
~1-4 ms commands instead of ~50 ms.

## Not done (open)

- Owner decision: `CommandKeep` > 0 (meets the P1 symbol/done budget, visible MMS-access change).
- 50k updates/s and 20-IED start benchmarks (S-series) not run on this laptop (1 GB free RAM).
- PUT `/scenarios`, PUT `/ieds/{id}/test-mode` (project saves, stream B; runtime part exists:
  `ApplyTestMode`), `/mms/access` (stream D; data from `Coordinator.MmsAccess`).
- The GOOSE engine manager, GOOSE/SV monitors and real-IED sessions (stream D) plug in through
  `runtime.Live`; until then `NoLive` (no GOOSE publishing in the Go preview).
- `runtimeapi/respond.go` writes Flask error bodies itself until stream A's `httpapi` exists.
- The template test project has no GOOSE and no real IEDs; those paths are not exercised here.
- Sine sources: last-bit float64 differences from the MSVC CRT (invisible on FLOAT32 points).

## Needs from other streams

- **A**: a `runtime.Bus` (Publish returning the envelope `{"id","kind","timestamp",**payload}`),
  mounting of `runtimeapi.Area.Routes()`, and its error writer able to render `apperr` the same
  way (or keep the local one).
- **B**: a `runtime.Project` (Current immutable tree with Python's keys/order, NodeData without
  `project_changed`, Source with save + `project_changed`, AssignIP); call
  `Coordinator.ResetValues(id)` when an IED is imported again or removed; merge
  `Coordinator.IedState(id)` (`state`, `goose_state`) into GET `/project` and use
  `Coordinator.Value/Signal` for live values in `/ieds/{id}/signals` and test panel views.
- **D**: a `runtime.Live` (Start, Connect, Disconnect, Stop, BusFor with `udp:PORT:MAC`), using
  `native.Live`/`native.Goose`; live values for real IEDs write through the runtime's tables.
- **F**: `LabAddresses(guid)` from the IP Helper for lab-mode preflight; the parity harness can
  reuse `testdata/record_python.py` and `compare_events.py`.
- **Integrator**: rebuild `studio_control` from this branch (`control/canvas_mux.cpp`,
  `control/canvas_command.cpp`, two sources added to the target in `studio/native/CMakeLists.txt`);
  without `control_mux_v1` the Go canvas operate answers "Bundled MMS client is unavailable".

## How to reproduce

```powershell
# native workers copied (untracked) into studio/native/build/Release; studio_control rebuilt:
cmake -S studio/native -B studio/native/build/mux -G "Visual Studio 16 2019" -A x64 `
  -DIEC_SOURCE=<repo>/.local/native/libiec61850 -DJSON_SOURCE=<repo>/.local/native/json `
  -DNPCAP_SDK=<repo>/.local/native/npcap-sdk -DSTUDIO_GATEWAY=OFF
cmake --build studio/native/build/mux --config Release --target studio_control
cd studio/engine; $env:GOMAXPROCS=4; go vet ./internal/...; go test -p 2 ./internal/native/... ./internal/runtime/... ./internal/api/runtimeapi/...
# events parity: python testdata/record_python.py <Release> 21200 py.json; STUDIO_PARITY_OUT=go.json go test -run Scripted; python testdata/compare_events.py py.json go.json
```
