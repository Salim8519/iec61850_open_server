# 0135: Speed lane I2 merged: Go vs Python gaps closed (Stop, first command, lab lookup, goldens, seams)

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md); gaps from
[0132](0132-speed-lane-i-composed-go-server.md). Branch `speed-i2` (706b78d..da93737, 8 commits)
merged into `master` (3a1998a). Stream log: [speed-i2](../../architecture/go-engine/streams/speed-i2.md).

## Gaps

- **Live values not saved (0132 gap 4): no gap.** A Python test with real workers shows Python
  keeps a runtime value for the session only; its archive stores IED settings and the SCL source,
  so a restart shows the initial value. Go matches at every stage
  (`TestRuntimeValuesAcrossRestartMatchPython`).
- **Stop (gap 2):** GOOSE monitor, GOOSE engine, SV listener and live sessions close in parallel;
  "stopped" `live_status` events keep connection order. Demo station, 12 rounds: Go p50 82 ->
  70-75 ms, stops over 300 ms 4/12 -> 1/12 (Python p50 106-132 ms). Remaining tail: studio_live's
  250 ms GOOSE supervisor sleep (native lane N3).
- **First canvas command (gap 3):** `studio_control` starts in the background at Run; first operate
  after Run p50 137 -> 66 ms, max 166 -> 90 ms; events still 60/26/9 identical.
- **Lab address lookup failure (gap 7):** Run answers Flask's generic 500 and a `runtime` stopped
  event with Python's "[Errno 87] GetAdaptersAddresses failed".
- **liveapi goldens through the composed pipeline (gap 6)** plus the flaky start step: the step
  waits (<= 10 s) for its recorded number of events; comparison unchanged; 40 runs at GOMAXPROCS=1.
- **Response writers (gap 5):** lanes use httpapi's writer; JSON answers and downloads carry
  Content-Length as Flask does.
- **Parity build (gap 8):** callback client, network/license/update/fault seams, inspect ops
  (`service/stop`, `fault`, `runtime/state` via `Coordinator.ForceState`, never used by the product,
  `runtime/workers`, network journal, `project/set`, `project/update`), Python XSD verdict recorder.
- **Flaky events test (gap 10):** free ports >= 20000. **Shared cookie (gap 9):** no code; open the
  second engine as `http://localhost:<port>` (cookies are per host, both hosts accepted).

## Evidence

- `STUDIO_ENGINE=go` pytest 432/104/9 failed/16 errors -> 455 passed / 104 skipped / 1 failed
  (python-http 456/104). Golden replay dual and `golden --engine go`: 4/4, 0 differences.
- `go vet ./...` and `go vet -tags parity ./cmd/...` clean; `go test -p 2 -trimpath ./...` green.
- Integrator: only Go engine paths; `verify.py`: see ledger line.

## Remaining

- `test_recovery_that_stopped_to_protect_autosave_refuses_work` fails under go (needs a copy seam
  in `internal/store/recovery`). `studio_xsd.exe` not built (go suite uses recorded verdicts).
- Parity-only `project/set` bumps the revision where Python's in-place edit did not (no test sees it).
