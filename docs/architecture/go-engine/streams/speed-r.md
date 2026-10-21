# Speed lane R: runtime ready for one Go server (S1, S6)

Branch `speed-r` from master fb8f804 ([speed plan](../speed-plan.md), phase 1). Owns
`internal/runtime/**` and `internal/native/**` only. No user-visible change: events, texts,
HTTP answers and per-IED event order are unchanged (evidence below).

## What is exported (S1)

`*runtime.Coordinator` now satisfies **`projectapi.Runtime`** and **`toolsapi.Runtime`**
directly, plus `acquisition.Runtime`, `acquisition.Emitter` and `acquisition.Publisher`
(compile-time checks in `runtime/ports/ports_test.go`).

| Export | Python | Notes |
|---|---|---|
| `CycleLock() sync.Locker` | `coordinator.cycle_lock` | returns `&lifeMu` |
| `Emit(kind, payload)` / `Publish(kind, payload)` | `coordinator.emit` / bus only | Emit also writes the run log |
| `RunningIEDs()`, `RunState()` | `sorted(active)`, `state` | names the consumers use |
| `Stopped(fn)` | `with cycle_lock: require_stopped(); fn()` | StudioError `StoppedRefusal` = "Stop the simulation before editing its structure" (400, code validation) |
| `RunningSequences()` | `tools/activity.py` count | scenario runs in state "running" |
| `Source(ied, ref any)` | `coordinator.sources.get(...)` | copy of the spec; `overridden` set only when the runtime changed it |
| `Signal(ied, ref any)` | `coordinator.signal` | was `(string, string)`; non-text id → "Unknown IED" (not_found), non-text ref → "Unknown signal reference"; view now carries `live_state` when set |
| `Endpoint(ied) (host, port any, err)` | `supervisor.endpoint` | port int64; ValueError "IED is not running" |
| `Record(ied, ref) (SignalRecord, bool)` | `coordinator.index.get(...)` | value, quality_ref, timestamp_ref, live_state ("" = absent) |
| `Values()` → `Has/Get/SetLive/MarkStale` | `ied_indexes`, `live_values` | per-slot `live` state; kept after Stop like Python |
| `SampleRunning`, `ApplySamples` | `ied in sample_writer.locks`, `sample_writer.apply` | quality None |
| `PublishChanged([]PointKey)` | `equipment_states.publish_changed` | |
| `SetIEDState(ied, state)` | `ied["state"] = ...` (real sessions) | notifies the hub |
| `IedStates() map[string]IEDOverlay`, `IedState(id)` | runtime-only `state` / `goose_state` on the IED dict | overlay for GET /project |
| `Subscribe(func(Change)) (cancel)` | (new, S6) | below |

New run index: `Coordinator.index` (= Python `ied_indexes`/`index`) is replaced at Run,
extended by single-IED start, shrunk by single-IED stop, and **kept after Stop** (Python never
cleared it), so `/signals` after Stop shows the last values and live_state (d-live open
question 2). The gateway's `Signal` reads this index, as Python's gateway read
`coordinator.index` in any state.

`internal/runtime/ports` adapts where names or slice types collide (gateway `Signal`/`Change`,
acquisition `RefValue`/`PointKey`): `ports.Gateway(c) service.Runtime`,
`ports.ForAcquisition(c)` (Runtime, Events, Bus, Values, Samples, Equipment, States, Start) and
`ports.StartWorker` (= `acquisition.StartWorker` on `native.Start`).

## Value hub (S6)

`Subscribe(notify)` reports every value, quality, timestamp and live_state write
(`Change{IEDID, Ref}`), every IED state, goose_state, index, stale-marking and values-reset change
(`Change{IEDID, ""}`) and every run-state change (`Change{}`). Hooks sit in the value table
writers (`set`, `setTQ`, `setValueTime`, `setLive`, `markStale`) and in the state setters, so no
writer can bypass them. `runtime.Change` has the fields of `service.Change` (plain conversion).

- `notify` never blocks and never calls a subscriber: a copy-on-write subscriber list (atomic
  load), per subscriber a leaf mutex, a coalescing set (a point still queued is not queued
  twice) and a queue bounded at 4096 distinct changes. Overflow drops the queue and sets the
  resync flag, so the subscriber gets one `Change{}` (re-read everything).
- One goroutine per subscriber delivers in queue order; only the first change after a drain
  wakes it. Idle cost: none (no timers). A panicking subscriber is logged, not fatal; `cancel`
  works from inside `notify`.
- Values unchanged by a write are still reported (one notification per write, like Python's
  one `signal` event per write); the gateway's own sample comparison filters them.

`BenchmarkHubNotify` (Ryzen 9 5900HS, GOMAXPROCS=4, `-benchtime 2000000x -count 3`):

| subscribers | ns/op | allocs/op |
|---|---|---|
| 0 | 2.5-2.6 | 0 |
| 1 | 182-226 | 0 |
| 4 | 1108-1385 | 0 |

The 1- and 4-subscriber numbers are mostly the benchmark thread contending with the delivery
goroutines on 4 cores. At 50 k writes/s with one subscriber that is ~1 % of one core.

## Native client (item 3)

- `Worker.Request(op, timeout, fields)` now has the argument order of `acquisition.Worker` and
  `tools/worker.Process`; the typed clients use the internal `call(op, fields, timeout)`.
  `Worker.Alive()` was added, so `*native.Worker` satisfies both interfaces.
- Timeouts are `*native.TimeoutError`: same text "Native <op> request timed out",
  `Timeout() bool` true (`acquisition.IsTimeout` works), `Unwrap()` → apperr Runtime (503 body
  unchanged through `errors.As`). `native.IsTimeout(err)` helper. The runtime's canvas-timeout
  check now uses `errors.As(*native.TimeoutError)` with `Op == "canvas_operate"`.
- `native.Fifo` (the worker inbox queue) is exported and reused by the runtime.

## Bug fixed on the way

Native control feedback used a 1024-entry channel filled on the worker's dispatcher goroutine.
With a full queue the dispatcher blocked, the worker's replies stalled, and a canvas command
holding `commandMu` while waiting for such a reply deadlocked against the feedback goroutine
waiting for `commandMu`. A late event after `Close` panicked (send on a closed channel). The
queue is now unbounded (`native.Fifo`), as Python's `queue.Queue` was.

## Lock order (item 4)

`lifeMu (CycleLock) -> commandMu -> project mutation -> table.mu -> emitMu/bus.mu`; the hub's
locks are leaves below all of them (notify may run under any lock and never calls out).
`c.mu` stays a short leaf for coordinator fields and is never held across I/O.

`TestLockOrderStress` (`STUDIO_STRESS=1`, `STUDIO_STRESS_SECONDS`, default 60) runs seven lanes
on fake workers (this test binary acting as `studio_ied`/`studio_control`, which reports an MMS
control every 7th batch so native feedback competes with canvas commands): Run/Stop/Pause/
single-IED stop+start, canvas operates (CB, DS, push button), manual writes + pulses + GOOSE-input
samples + resume, `Stopped` and `CycleLock` project edits + `Emit` + `ResetValues` + source
configuration + scenarios, live values + stale + IED state, subscribe/cancel churn with callbacks
reading back into the runtime, and readers. A lane without progress for 10 s dumps every
goroutine and fails. 60 s run: lifecycle 128, operate 18,545, writes 35,966, edits 3,137, live
44,352, subscribers 3,984, readers 42,917 operations; 136,087 events (6,804 feedback applied,
6,534 native feedback observed, 533 removals under the cycle lock), 174,544 hub deliveries, no
stall, 9 goroutines at the end, no fake process or temp folder left behind.

## Evidence

- `go vet ./...` clean; `go test -p 2 -trimpath ./...` green (one run hit the known Windows
  temp-cleanup flake in `internal/updates`, not in this lane; it passed on rerun). Runtime test
  temp folders now use `testutil.TempDir`.
- New tests: `TestHubReportsEveryChange` (run, manual write with q/t, live value, stale, IED
  state, pause, stop, index kept after Stop, values reset, endpoint), `TestStoppedRefusesWhileRunning`,
  `TestHubCoalescesAndResyncsWithoutBlocking`, `TestHubCancelInsideNotify`,
  `TestTimeoutErrorIsRuntimeAndTimeout`, `TestAdaptersAnswerLikeTheCoordinator`. They need no
  native build.
- Real workers from `studio/native/build/Release`: native protocol tests pass; operate tests skip
  (no `control_mux_v1` there, expected).
- With stream C's mux build (`.claude/worktrees/agent-ae92a6e23d0a27d72/studio/native/build/Release`,
  read only): `TestScriptedRunWithRealWorkers` and `TestOperateTiming` pass (keep 0: done p50
  53.5 / p95 94.4 ms; keep 10 s: done 2.8 / 4.4 ms, symbol 1.4 / 1.6 ms). Events parity against
  `record_python.py` + `compare_events.py`: per IED identical in every run (26 / 9 events). The
  global stream was identical (60 events) against one Python recording. It differed at event 15
  against another, where Python itself put `equipment_state ds1` before the asynchronous
  `equipment_feedback_observed cb1`. That is Python's own run-to-run variance of the feedback
  thread; three Go runs gave the same order.

## What other lanes must do

- **Integrator (I):**
  - projectapi `Deps.Runtime = coordinator`. toolsapi `Deps.Runtime = coordinator`.
  - Gateway/master `service.Runtime = ports.Gateway(coordinator)`.
  - Acquisition: `a := ports.ForAcquisition(coordinator)`, then set `Deps.Runtime/Events/Bus/Values/Samples/Equipment/States/Start` from `a`.
  - Pass `runtime.Options.Live` an adapter over `acquisition.Manager` (Start takes `[]string` here, a set there).
  - GET /project: merge `coordinator.IedStates()` into each IED. Set `state` when non-empty (existing key keeps its position, else appended). Set `goose_state` while `HasGoose`, else remove it.
  - Call `coordinator.ResetValues(id)` on IED re-import or removal.
  - Hold `CycleLock()` before any project mutation in the same call, never the other way round.
  - Note: acquisition and runtime both have MMS-access and test-mode code. Pick one owner per feature (acquisition's `MmsAccess`/`TestModes` vs runtime's `mmsAccess`/`applyTestModes`).
- **Lane P:** `runtime.Project.AssignIP` is still required by `runtime.Project`. `projectapi.Runtime` is unchanged and satisfied as is.
- **Lane T:**
  - `*native.Worker` satisfies `tools/worker.Process` (Request, Alive, Close). Spawning through `native.Start` puts tools children in the kill-on-close Job Object.
  - Gateway/explorer code that checks `*worker.Timeout` with `errors.As` must switch to an `interface{ Timeout() bool }` check (`native.IsTimeout`).
  - Event delivery differs slightly: native delivers events on its own dispatcher goroutine in pipe order, with `worker_failed` once and stderr as `native_diagnostic`.
  - toolsapi's fake `Signal(string,string)` on `fakeRuntime` is test-only and needs no change.
- **Stream D/acquisition:** `ports.StartWorker` returns `*native.Worker`; timeouts satisfy `IsTimeout`.

## Open

- Python's `sample_writer.locks` survived Stop; `SampleRunning` is false after Stop. It is only consulted while live sessions run, which Stop ends first, so nothing visible changes.
- `Signal` with an unhashable id/ref (list) answered Python's 500 TypeError only while running; Go answers "Unknown IED"/"Unknown signal reference". The UI never sends this.
- A race-detector run is not possible here (no cgo toolchain); the stress test plus watchdog stands in.
