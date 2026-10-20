# Speed lane I2: gap closure of the composed Go server

Branch `speed-i2` from master c9ecf1c ([speed plan](../speed-plan.md), after lane I,
[speed-i](speed-i.md) "Gaps and differences"). Owns `internal/{app,acquisition,runtime,native,
httpapi,api}` and `cmd/` for these gaps. Python, the frontend and `studio/native` are unchanged.
No EXE was built. Gap numbers below are lane I's list.

## Status per gap

| Gap | Status | What changed | Proof |
|---|---|---|---|
| 4 runtime values not saved | **no gap** | nothing: Python never saved them either | `TestRuntimeValuesAcrossRestartMatchPython` |
| 2 Stop slower | closed | `acquisition.Manager.Stop` closes GOOSE monitor, engine, SV and every session's live client at the same time; session "stopped" `live_status` events still follow in connection order | `TestStopClosesSessionsTogetherInOrder`, `TestComposedStopTiming` |
| 3 first command waits for studio_control | closed | Run start (with simulated IEDs) starts the `studio_control` process in the background; `close` waits for a warm-up still starting | `TestComposedFirstOperateTiming`, events parity unchanged |
| 8 parity build seams | closed except 1 test | callback seams and inspect ops (below) | `STUDIO_ENGINE=go` pytest |
| 6 liveapi goldens through the composed server | closed | `TestComposedRoutesMatchPython`: the 48 recorded Flask answers and 13 injected events through `httpapi.New` (session token, loopback) with the same fakes | test |
| liveapi flake (integrator note, master 7294f81) | fixed | the start step waits (bounded, 10 s) until its recorded number of events arrived; the comparison is unchanged | 40 runs at `GOMAXPROCS=1` |
| 7 lab-mode address lookup failure | closed | `runtime.Options.LabAddresses` returns an error; Run fails like Python's `OSError(87)`: Flask's generic 500 and `runtime` stopped with "[Errno 87] GetAdaptersAddresses failed" | `TestLabLookupFailureMatchesPython` |
| 11 (task gap 10) flaky events test | fixed | composed real-worker tests take free ports >= 20000 on the IED hosts (`freePortBase`) instead of the fixed base 21300 | events test passed in all 3 runs |
| 5 per-lane JSON writers | closed | liveapi, runtimeapi, projectapi, toolsapi JSON answers use `httpapi.WriteJSON`, downloads `httpapi.WriteBody`; bodies byte-identical, Content-Length now present like Flask; error mapping stays per lane | api + app parity tests |
| 10 (task gap 9) shared session cookie | proposal only | see below | — |

### Gap 4: what Python does

`values_restart_python.py` runs the real Coordinator on a ProjectService folder (parity project,
real workers): write BayA AnIn2 = 42.5, Stop, Run again, Stop, save, restart (`recover_startup`).
Python keeps the value on the project's signal dict for the session (after Stop 42.5, next Run
42.5) but the archive stores only `IED_FIELDS` + `source_id` (`document.compact`), so a restart
rebuilds the signals from the SCL: 0.0. Go answers identically at every stage (the per-IED value
tables of lane R already survive Stop and re-Run), and its archive has the same IED keys
(`filename goose_publish id ip lab_ip live name port role source_id test_mode`). Operated
breaker positions are saved through the diagram (`initial_position`), already covered.

### Gap 8: parity build (cmd/studio_server, `-tags parity` only)

- `parity_callback.go`: the JSON callback protocol of `tests/parity/callbacks.py`
  (`return`/`raise`/`original`, wire tags bytes/path/handle/set/tuple/exception).
- `parity_fakes.go`: license object seam as the 402 gate (`enforce` attr, `is_licensed()`),
  update opener callback (HTTPError code becomes the status), fault seam on
  `background_recovery.recover_startup` (`app.Options.RecoverStartup`; a handle touch means
  "run the original").
- `parity_inspect.go`: `network/seam` (`_list_adapters`, `_list_rules`, `_probe`, `_elevate`
  on `Engine.Network`), `network/journal/outstanding`, `service/stop` (explorer, control,
  scanner, gateway, master), `fault`, `POST runtime/state` (`Coordinator.ForceState`, never
  used by the product), `GET runtime/workers` (ids, no pids).
- `parity_project_ops.go`: `project/set` and `project/update` write-through as a copy along the
  path committed by `Projects.Update` (the revision moves; Python's in-place edit did not; no
  test observes it).
- XSD verdicts: `internal/app/testdata/xsd_record_plugin.py` records every verdict the Python
  suite computes (in process, `-p xsd_record_plugin`, 58 verdicts) for `STUDIO_XSD_SEED`.

Not ported: `runtime/read` (supervisor reads), `project/diagram|source|add_scl` inspect calls
(no test uses them under go), the `shutil.copyfile` fault (needs a seam in
`internal/store/recovery`, lane P), `license.runs_file = None`.

## Measurements (2026-10-20, GOMAXPROCS=4, -p 2, CPython 3.12 venv)

Real workers: copies of `studio/native/build/Release` in a scratch folder with lane I's
`control_mux_v1` `studio_control` (`native/build/speed-i`); loopback 127.63.1.x / 127.67.x.y,
free ports >= 20000. Demo station = public `scd/open_substation.scd` (4 IEDs, GOOSE, SV).

| Measure | Before | After | Python |
|---|---|---|---|
| Stop 4 IEDs (HTTP, 12 rounds) p50 / runs > 300 ms | 82.1 ms / 4 of 12 | 70.2-74.8 ms / 1 of 12 | 105.7-132.5 ms / 2-3 of 12 (Coordinator call) |
| Run 4 IEDs (HTTP) p50 | 279-354 ms | 293 ms | 440-526 ms |
| First canvas CB operate after Run (fresh engine, 6 rounds) p50 / max | 136.8 / 166.3 ms | 65.9 / 89.6 ms | — |
| Next operate p50 | 44.1 ms | 44.0 ms | — |
| `STUDIO_ENGINE=go` pytest (no native build) | 432 passed, 104 skipped, 9 failed, 16 errors | **455 passed, 104 skipped, 1 failed** | python-http 456 / 104; in process 456 / 104 |

Stop breakdown before (instrumented, not committed): studio_live (GOOSE monitor) answers
"stop" in 35-316 ms, studio_goose 8-14 ms, the 4 IED workers 17-21 ms (already parallel).
The remaining tail is native: `GooseTraffic::supervise` sleeps 250 ms and `stop()` joins it
(plus a 50 ms inbox select). Python pays the same (its own > 300 ms stops). A condition
variable wake in `studio/native/live/goose_traffic.cpp` (lane N1) would remove it.

Before numbers of the pytest row used lane I's seeds; the 2 version failures of a binary
outside `studio/engine/build` disappear when it is built at the harness default path.

## Other evidence

- `go vet ./...`, `go vet -tags parity ./cmd/...`: clean. `go test -p 2 -trimpath ./...`
  (STUDIO_PYTHON set): all packages ok. Real-worker `internal/app` run: events parity
  identical (60 / 26 / 9), click path, Flask parity, every route served, values, lab test pass.
- `python -m parity.replay dual --engines python-http,go` and `golden --engine go`: 4/4
  scenarios, 0 differences. `check_file_lengths.py`: PASS.

## Remaining

1. `test_recovery_that_stopped_to_protect_autosave_refuses_work` under go: needs a copy seam in
   `internal/store/recovery` (lane P) for the `shutil.copyfile` fault.
2. Gap 1 (`studio_xsd.exe` not built) stays; the go suite runs on recorded Python verdicts.
3. Stop tail from studio_live's 250 ms supervisor sleep (native, lane N1).
4. `studio/engine/build/` is still not git-ignored (`.gitignore` outside this lane).

## Gap 10 proposal: two engines on one host share `studio_session`

Cookies are scoped by host name, not port, and the session check already accepts both
`127.0.0.1` and `localhost`. Opening the second engine of a side-by-side comparison as
`http://localhost:<port>/#session=...` (the first stays on `127.0.0.1`) gives each tab its
own cookie with no code change and no dev/production difference. If a tool must open both on
`127.0.0.1`, an engine-specific cookie name behind the parity build tag only (never the
product) would also work; not implemented, since the host-name route needs nothing.
