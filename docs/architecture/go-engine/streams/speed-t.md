# Speed lane T: tools workers and the test-panel route

Branch `speed-t` (cut from `master` @ e2efa96). Owner of `studio/engine/internal/tools/**`,
`studio/engine/internal/api/toolsapi/**` and this log ([speed plan](../speed-plan.md), lane T).
Nothing outside those folders changed. Conditions: `GOMAXPROCS=4`, `go test -p 2 -trimpath`,
`STUDIO_PYTHON=studio/.venv/Scripts/python.exe`, native workers read-only from the main
checkout's `studio/native/build/Release` through `STUDIO_NATIVE_BIN` (the worktree has no build).

## 1. Tool workers in a kill-on-close Job Object

- Before: `tools/worker.Spawn` (used by the explorer, scan, `studio_control` client, gateway,
  master and converter) started children with `CREATE_NO_WINDOW` only. A hard kill of the
  server left `studio_live`, `studio_control`, `studio_gateway` and `studio_master` running.
- After: `tools/worker/job_windows.go` creates one unnamed, non-inheritable Job Object with
  `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` on first spawn and assigns every child right after
  `Start` (same stdlib `syscall` calls as `internal/native/spawn_windows.go`; native's helper is
  unexported, so the lane keeps a small copy). The handle stays open for the server's life;
  Windows closes it on any exit, which kills every tool worker. A failed assignment leaves the
  worker running exactly as before (Python has no job); `Close` is unchanged.
- The two jobs (native's and this one) are independent; each child joins exactly one.
- Tests (`job_windows_test.go`, the test binary re-runs itself as the fake child/server):
  - `TestJobCloseKillsMember`: a sleeper in a fresh job dies within 5 s of the handle closing.
  - `TestHardKillLeavesNoChildren`: a fake server spawns a sleeper through `Spawn`, prints its
    pid and is killed with `TerminateProcess`; the sleeper dies with it (about 1 s in total).
  - Mutation check: with the `adopt` call removed, `TestHardKillLeavesNoChildren` fails
    ("worker outlived the hard-killed server").

## 2. One owner for `GET /project/test-panel`

Both `runtimeapi` and `toolsapi` registered it; `httpapi.New` refuses a (method, path) twice.

- `runtimeapi.testPanel` runs over the real `runtime.Coordinator` (`Signal`, `SourceState`) and
  the project tree. `toolsapi.panel` needed two seams nobody implements (`Runtime.Signal`,
  `Runtime.Source`), and its parity scenario only covered a stopped panel with a fake runtime.
- Evidence that runtimeapi's handler is Python's: its own golden (`stopped_golden.json`, empty
  panel) passes, and a lane-T scratch replay (not committed) against the frozen Flask app with
  the runtime test project, three panel entries and sources configured in between gave 5/5
  identical status and body bytes: panel before sources (1162 bytes), `POST /signals/source`
  ramp and manual, panel after (`automatic` true for ramp, false for manual and for a signal
  with no source), and a stale entry with an unknown reference (400, 37 bytes).
- Change: toolsapi keeps `PUT /project/test-panel` and drops the GET. `toolsapi.Runtime` no
  longer needs `Signal`/`Source`, and `toolsapi.Projects` no longer needs `TestPanel()`
  (smaller seams for the integrator; existing implementations still satisfy them).
- `TestRoutesAgainstFlask` still sends the Flask scenario's GET; the Go replay now checks what
  this area's PUT stored against the entries in Flask's answer (minus `signal`, `overridden`,
  `automatic`). A deliberate break of that comparison fails the test.
- `TestComposesWithRuntimeRoutes` builds `httpapi.New` with the tools and runtime routes
  together (fails "registered twice" before this change).

## 3. Idle timers and gateway CPU

| Path | Driven by | Timers while idle |
|---|---|---|
| Gateway value push (`gateway/service/sync.go`) | value-hub `Subscribe` wake channel, <= 1 batch / 100 ms | one `FullSweep` ticker, every 10 s, re-checks all points |
| Command confirmation (`observe.go`) | hub notification of that IED/status; 3 s deadline timer only during a command | none |
| SCADA master (`master_connect.go` `keep`) | user interval (1 s, or Modbus `poll_ms`), only while connected | Python's protocol loop; not an idle-app timer |

Measured with the existing `TestGatewayCPUAgainstPython` (`STUDIO_GATEWAY_BENCH=go`, 5000
points, real `studio_gateway`, 20 s windows, exact cycle counts):

| Changes/s | Go gateway CPU |
|---|---|
| 0 (idle) | **0.08 %** of one core (0.016 s / 20 s) |
| 1000 | 0.73 % (gate <= 1 %) |

A scratch run (not committed) that sets `SweepInterval` gave idle 0.055 % and 0.039 % with the
10 s sweep and **0.001 %** (0.0001 s / 20 s) without it, twice each. So nearly all idle gateway
CPU is the safety sweep; the push path itself is idle-silent. The speed-plan gate (<= 0.1 %)
holds either way.

## Suite

`go vet ./...` clean (also `GOOS=linux` for the lane's packages). `go test -p 2 -trimpath
./...`: 50 packages ok; one unrelated flake, `store/recovery`
`TestRecoverStartupSizeLimitAndNonFile` ("TempDir RemoveAll cleanup ... directory is not
empty", it uses `t.TempDir`, not `testutil.TempDir`), passed 3/3 when re-run alone.

## Open items for other lanes

- **Gateway service owner (`internal/gateway/service`):** replace the 10 s `FullSweep` ticker
  with an event: mark all points dirty on the project `changed("ieds")`/role edit the hub does
  not report, then the gateway has no timer while idle (0.001 % measured). Needs the hub to
  report every value/quality/timestamp/IED-state/run-state change (stream E requirement).
- **Integrator (lane I):** mount `runtimeapi` for `GET /project/test-panel` and `toolsapi` for
  `PUT`; the toolsapi `Runtime` adapter now needs only `Stopped` and `RunningSequences`.
- **Runtime (lane R):** `internal/native` could export its job helper (`Adopt(pid)`) so the
  tools worker uses the same job; until then two identical small helpers exist.
- Not covered by a golden: `overridden: true` in the panel (needs a running IED with a manual
  write over an automatic source); runtimeapi maps it from the coordinator's `overridden` flag.
