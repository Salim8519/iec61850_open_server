# 0125: Speed lane R merged: runtime exports, value hub, native request/timeouts, feedback deadlock fix

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 1, lane R
(S1, S6). Branch `speed-r` (f0d07a6) merged into `master` (b89cc0b). Stream log:
[speed-r](../../architecture/go-engine/streams/speed-r.md). Not wired into the Go server yet.

## Change

- `internal/runtime`: `*Coordinator` satisfies `projectapi.Runtime`, `toolsapi.Runtime`,
  `acquisition.Runtime/Emitter/Publisher` (compile-time checks). New exports: `CycleLock`, `Emit`,
  `Publish`, `RunningIEDs`, `RunState`, `Stopped(fn)` (Python's exact refusal), `RunningSequences`,
  `Source`, `Signal`, `Endpoint`, `Record`, `SetIEDState`, `IedStates`, `PublishChanged`,
  `SampleRunning`, `ApplySamples`, `Values()` with live state per slot; run index like Python's
  `ied_indexes` (kept after Stop). `internal/runtime/ports`: `Gateway`, `ForAcquisition`,
  `StartWorker` adapters where names collide.
- Value hub `Subscribe(func(Change)) cancel`: every value/quality/timestamp/live-state write, IED
  and run-state change; never blocks (bounded coalescing queue, overflow = one resync), no work
  when idle.
- `internal/native`: `Request(op, timeout, fields)` order acquisition expects, `Alive()`,
  `*TimeoutError` with `Timeout() bool` (same text and 503 body).
- Bug fix: control feedback used a 1024-slot channel that could deadlock against a canvas command
  holding `commandMu` when full, and a late event after Close panicked; now an unbounded queue as
  Python's `queue.Queue`.

## Evidence

- `BenchmarkHubNotify`: 0 allocs/op; 2.5 ns with no subscriber, 182-226 ns with 1, 1108-1385 ns
  with 4. `TestHubReportsEveryChange`, coalescing/resync/cancel tests, `TestStoppedRefusesWhileRunning`.
- `TestLockOrderStress` (STUDIO_STRESS=1, 60 s, fake workers): ~190,000 operations, 136,087 events,
  174,544 hub notifications, no stall, nothing left behind.
- With stream C's mux build (read-only): `TestScriptedRunWithRealWorkers` passes; operate done
  p50/p95 53.5/94.4 ms (association closed), 2.8/4.4 ms (kept). Events identical per IED to Python
  recordings; global order differs from one of two Python recordings only where Python itself
  varies between runs (async feedback vs `equipment_state`).
- Integrator on `master`: only owned paths; `go vet ./...` clean; runtime, native, acquisition,
  api, gateway and tools packages pass; `verify.py`: see ledger line.

## Limitations

- After Stop `SampleRunning` is false (Python kept locks; not visible). A list used as IED id or
  ref answers "Unknown IED"/"Unknown signal reference" instead of Python's 500 (UI never sends it).
- Race detector unavailable (cgo off); the stress test stands in. Wiring notes for the integrator
  and lanes P/T are in the stream log.
