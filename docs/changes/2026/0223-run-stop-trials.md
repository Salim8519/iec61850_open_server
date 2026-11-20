# 0223 Run and Stop trials R1-R7: spare IED processes, Stop with real sessions together, a busy live client ended at once, Connect without waiting

Date: 2026-11-20. Owner: Run and Stop are slow; "do all of them, once i wake, i should see all
ready and no coding left, just testing".
Plan: [speed and network trials](../../testing/speed-trials-plan-2026-11-20.md), ideas R1-R7.
Results: [trial results](../../testing/speed-trials-results-2026-11-20.md), [Run and Stop details](../../testing/speed-trials-runstop-2026-11-20.md).
Decision: [ADR-0142](../../decisions/ADR-0142-spare-ied-processes-and-connect-without-waiting.md)
(R3, R7; R1 and R5 recorded as not adopted). Layers: Go engine (runtime, acquisition, native
client, gateway service, app), native `studio_ied` (one new request), trial harness.

## Change

- **R3, spare IED processes.** `runtime/spare_ieds.go` (new): while stopped, one `studio_ied` per
  simulated IED is started one after another and greeted with `hello` (licence check); Run's
  `launch` takes one (`supervisor.go`) and starts a process only when none waits; its events go to
  the IED through a small router. Spares are readied after Stop, after a failed Run and after the
  idle start-data read (`start_warm.go` `AfterWarm`, set in `app/startup.go`); a changed
  `studio_ied.exe` (size or time) or a dead spare is never used; `Coordinator.Close` closes them
  before its last Stop. Native: `studio_ied` answers `hello` with `{"ready": true}` after
  `license::require_valid_license()` (`native/worker/main.cpp`); nothing else in the worker changed.
- **R4, Stop with real sessions.** `runtime/stop.go`: the real-IED sessions (`Live.Stop`, which
  gives back the report blocks) close while the simulated IEDs' workers close; the IEDs are marked
  stopped after both, so the events keep their order (`supervisor.go` `closeWorkers` /
  `markWorkersStopped`).
- **R6, a busy live client is ended at once.** `acquisition/session_close.go` (new): a session's
  live client counts its requests in flight; Stop or Disconnect while it is busy (connecting to,
  or reading the data model of, a device that does not answer) and before the session reserves
  any report block kills the client (`native.Worker.Kill`, new) instead of sending "disconnect",
  which waited behind the request (15 s connect, 120 s data model). From the first reservation on,
  "disconnect" is sent as before; a reservation never starts once Stop began.
- **R7, Connect without waiting.** `gateway/service/master_connect.go`, `master_asking.go` (new):
  Connect answers once the link is open and runs the interrogation in the background; the
  converter's point mapping (`converter_points.go`) and the automatic connect at launch wait for
  it; Disconnect during an interrogation ends the worker at once; an interrogation failure caused
  by Disconnect is not shown (`master_ask.go`).
- **R1, R2, R5 dropped** (code built, measured, removed again): see Evidence.
- Trial tools: `runstop_ab.py`, `real_stop_ab.py`, `r7_master_connect.py`, `r7_station.py`,
  `r3_spare_cost.py`;
  `instance.py`, `projects.py`, `watcher.py` read the trial's address range, ports and workers
  from `SCADABENCH_TRIAL_RANGE`, `SCADABENCH_TRIAL_PORTS`, `SCADABENCH_TRIAL_NATIVE`.
- Specs: RUNONE-09, LIVE-15, MASTER-16; API: `POST /runtime/stop`, `POST /tools/master/connect`.

## Evidence

Measured on the owner's PC in alternating sessions against `master` (details and every number:
[Run and Stop details](../../testing/speed-trials-runstop-2026-11-20.md)); other agents loaded
the PC (52-82 % busy). Medians / worst, at least 10 counted rounds per side.

- **R3** Run: small 0.884 / 1.201 s to 0.647 / 0.795 s (-27 %), large 3.116 / 5.869 s to
  1.167 / 2.763 s (-63 %); the Run started 0 processes instead of 8 / 20. Spares: 0.82 MB private
  memory each, 0.000 CPU s in 20 s. Packaged licence check parts moved out of Run: 20 machine-ID
  reads at once 0.223 s, 20 hashes of the 26.5 MB `studio_server.exe` at once 0.131 s.
- **R4** Stop with a real session holding a report block: 0.079 / 0.278 s to 0.047 / 0.127 s; an
  independent `studio_live` saw the block reserved while running and free after every Stop (20/20).
- **R6** Stop 1 s after Run while connecting to a listener that never answers: 4.074 / 4.231 s to
  0.045 / 0.109 s.
- **R7** Connect, 5000 points, stand-in outstation without counter confirmation: 15.117 / 15.181 s
  to 0.073 / 0.125 s, every point shown at 0.078 s; first Disconnect of a session 14.6 s to
  0.016 s; with `studio_gateway` as outstation 0.461 / 0.804 s to 0.260 / 1.172 s; 5000 of 5000
  points every round on both sides.
- **Dropped:** R1 Run small 1.595 to 1.541 s, large 2.257 to 2.482 s (slower); R2 initial-state
  step 0.087 to 0.003 s (3-6 % of Run) but the event order changed every round; R5 click to
  `stopped` 0.096 s on both sides (the release after ADR-0141 costs about 2 ms).
- Wire: `native_wire_compare.py` Release against `trial-r3`: WIRE IDENTICAL (5 MMS connections,
  probe outputs, 4 reports, 14 GOOSE frames). Native protocol tests on `trial-r3`: 16 passed.
- Go: new tests fail before and pass after: `TestStopWhileConnectingKillsTheClient` (15.0 s
  before), `TestConnectAnswersBeforeTheInterrogationEnds`, `TestDisconnectDuringInterrogationIsQuick`
  (10.0 s before), `TestRunTakesSpareProcesses`, `TestSparesNotUsedAfterTheProgramChanged` (new
  API); `TestStopAfterReservingStillDisconnects` keeps the give-back. `go test` passes for
  acquisition, native, gateway, runtime, app and api; `go vet` (Windows) clean for the module,
  `GOOS=linux go vet ./internal/...` clean (the two Windows-only service commands do not build
  for Linux, as on master).
- `verify.py`: every check PASS (file lengths, architecture, documentation, history, specifications, offline interface, GOOSE engine and Go engine tests).

## Limitations

- The trial native folder `studio/native/build/trial-r3` holds the `hello` build; the lead rebuilds
  `Release` (and the installer's copy) from this source. Until then a development engine with the
  old `Release` worker keeps no spares (the old worker refuses `hello`) and starts processes as
  before.
- Spares are readied one after another after Stop: a Run clicked within about 3 s on a very busy
  PC may find only part of them and starts the rest itself (3 of 10 rounds at 82 % busy).
- Packaged-build timing was not measured (no EXE tonight): the licence-check saving is from its
  parts measured on this PC. Real cards, a third-party IED and a third-party RTU were not used
  (Owner checks in the details).
- R6 kills only a client that is busy and has reserved nothing; a Stop while a long read runs
  on a session that holds a report block still waits for it (the block must be given back).
- R7's stand-in outstation is written from the standard in Python (`r7_station.py`); with our own
  lib60870 gateway Connect was already under a second.
- Two Runs (one today, one with spares) were refused with "Native start request timed out" when
  the PC was saturated by other agents; nothing in this change touches that path.
