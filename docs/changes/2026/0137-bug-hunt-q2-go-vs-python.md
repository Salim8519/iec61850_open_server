# 0137: Bug hunt Q2: every function on the composed Go server side by side with Python

Date: 2026-10-20. Speed plan, lane Q2. Branch `speed-q2`. Stream log:
[speed-q2](../../architecture/go-engine/streams/speed-q2.md) (function table, timings, push plan).

## Change

- Go `app/ied_state_view.go`, `app/wire_tools.go`, `app/project_view.go`: the gateway sees the
  runtime IED state (overlay on copies); before, every gateway point had no value, quality IV+NT
  and every IEC 104 command was refused on the Go engine while the IEDs ran.
- Go `push/stream.go`, `push/area.go`: after a quiet period the stream waits `Settle` (1 ms) once,
  so the events of one request (test press: `signal`, `equipment_state`) leave in one write
  instead of the second waiting out the 16 ms gather.
- Go `store/recovery` + `cmd/studio_server` (parity build only): copy fault seam for
  `startup_recovery.shutil.copyfile`, so `test_recovery_that_stopped_to_protect_autosave_refuses_work`
  runs under `STUDIO_ENGINE=go`.
- Go events parity scenario (`record_python.py`, runtime `parityScript`, composed test): wait
  0.3 s after each of the two operates; `TestComposedEventsMatchPython` failed 7 of 12 runs on
  master with N3 natives because cb1's reported position raced ds1's operate in both engines.
- Frontend `copy/master.ts`, `RoleSwitch.tsx`: greyed-out role buttons say "Stop or disconnect to
  switch roles." in their tooltip.

## Evidence

- New tests: `TestGatewayProjectsCarryRuntimeState`, `TestComposedGatewaySeesRunningIEDs` (real
  workers; fails before the fix), `TestFirstBurstAfterQuietLeavesTogether` (fails before),
  `TestCopySeamFailureProtectsAutosave` (parity tag), `bugHuntQ2.test.ts`.
- Browser, both engines: normalised page text identical on every page listed in the log; Go
  gateway 60/60 points online, IEC 104 GI and commands match Python.
- Test press to symbol state on the push stream (server side, N3 natives): p50 3 ms on Go,
  5 ms on Python `/events`; before the fix half the presses took 16-20 ms on Go.
- F4 11-step push plan passed on Go (Vite proxy path not tested).
- Events parity test 12/12 (N3) and 6/6 (pre-N3) after the scenario fix (7 of 12 failed before).
- `STUDIO_ENGINE=go` pytest 456 passed, 104 skipped, 0 failed (was 1 failed); in process
  unit + integration 453/8; frontend 228/228; other suites in the log's Verification section.

## Limitations

- Go ran with `STUDIO_XSD_SEED` verdicts (no schema helper in Go yet).
- B3 could not be re-timed in the browser after the final build (pane throttled to 1 frame/s).
- Remaining same-on-both issues are listed in the log; none changed here.
