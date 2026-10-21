# 0117: Go stream C merged: native clients, runtime coordinator, multiplexed canvas operate

Date: 2026-10-20. Workstream C of [ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)
(phase P1). Branch `go-c-runtime` (a54dd17..603bc03, 8 commits) merged into `master` (afa8149). Stream
log: [c-runtime](../../architecture/go-engine/streams/c-runtime.md) (proposed ADR text for 0077, 0078).
Python stays the engine; the Go runtime is not wired into the Go server yet.

## Change

- `internal/native`: framed pipe client and typed clients for studio_ied, studio_live, studio_goose,
  studio_control; every worker in one kill-on-close Job Object.
- `internal/runtime` (+ `pyval`, `signal`, `plant`): per-IED actors, sample writer, sources,
  CPython-compatible generators, pulses (deadline timers), scenarios, equipment control and native
  feedback, energization and switching hints, run log on its own goroutine.
- `internal/api/runtimeapi`: runtime routes as `api.Route` with Flask golden answers.
- Native: `studio_control` canvas mode `control_mux_v1` (`control/canvas_mux.*`, `main.cpp`,
  `CMakeLists.txt` adds the two sources). The binary in `studio/native/build/Release` is NOT rebuilt
  (the owner's app uses it); Go canvas operate needs the rebuilt `studio_control`.

## Evidence

- Real workers on 127.63.x.y, ports ≥ 21000, temp data: canvas operate vs `studio_probe control`:
  same result bytes, same worker events in order, same MMS reads/writes in the worker access log
  (no capture tool installed; worker-side view).
- Events parity on a scripted scenario (run, CB close, DS open, manual write, pulse, failing assert,
  test button, latching button, stop/restart one IED, stop): identical per IED and globally
  (60/9/26 events) after normalising ids, timestamps, peers, correlation ids; wall-clock-driven
  events excluded; set-ordered Python emissions compared as multisets.
- 30 idle/refusal HTTP answers byte-identical to Flask. Generators: noise bit-identical over 10,000
  rows; sine differs in the last float64 bit in 147/1000 rows (Go `math.Sin` vs MSVC `sin`), equal
  after FLOAT32 rounding. CSV traces 72 rows, energization 729 position sets: identical.
- Operate, synthetic two-IED project, 30 CB operations each (done p50/p95, symbol p50/p95):
  Python 182/432 ms, 177/426 ms; Go closing the association after each command 51/81 ms, 49/78 ms;
  Go keeping the association 10 s 3.6/19.7 ms, 1.7/16.2 ms.
- Integrator: branch based on c983a8f; only owned paths plus `studio/native/CMakeLists.txt`;
  `go vet` clean; runtime, plant, pyval, signal, runtimeapi pass; `internal/native` passes when its
  test binary runs outside Go's temp build folder (the antivirus refused `native.test.exe` there).

- Integrator fix: the three tests that operate real workers (`TestOperateTiming`,
  `TestScriptedRunWithRealWorkers`, `TestRunOperateAndStopOverHTTP`) now skip when the native folder's
  `studio_control` lacks `control_mux_v1` (the main checkout's binary predates it), and the timing
  test always closes its coordinator; `verify.py` on `master` passes 7/7.

## Open decision (owner)

Keeping the canvas command association open meets the P1 budget (symbol ≤ 20 ms, done ≤ 50 ms) but
ScadaBench then stays listed as an MMS client of the IED and fewer `connection` events appear. The
default closes it after every command (identical to Python today).

## Limitations

- Not done: PUT `/scenarios` and PUT test-mode saves (stream B); live/GOOSE/real-IED plug-in through
  `runtime.Live` (stream D code now on master; wiring pending), so the Go preview publishes no GOOSE yet.
- 50k updates/s and 20-IED benchmarks not run. Template project has no GOOSE or real IEDs.
- Deliberate differences: Stop is not blocked by an in-flight operate (no global lock); pulses restore
  at their deadline instead of on a 20 ms poll.
