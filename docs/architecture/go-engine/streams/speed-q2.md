# Speed lane Q2: bug hunt on the composed Go server, side by side with Python

Branch `speed-q2` from master 3a1998a, fast-forwarded to 650d0f0 before the commit
([speed plan](../speed-plan.md); follows [0126](../../../changes/2026/0126-bug-hunt-q-frontend-fixes.md),
[speed-i2](speed-i2.md), [speed-f4](speed-f4.md)). Every page and function was used in the
browser on the composed Go server (`studio_server.exe`, dev mode, own port and data folder) and on
the Python launcher (own port and data folder) at the same time, with the same seed project.
No EXE was built. The owner's app (Vite 5173, its backend, its data folder,
`studio/native/build/Release`) was not touched.

## Setup

- Native workers: copies in a scratch folder. Measurements marked **pre-N3** used the merged
  natives before lane N3; **N3** used the integrator's `studio/native/build/integrate2/Release`
  (lane N3 waits). Both engines always used the same copy.
- Project: public demo data only (the OpenSCD-based demo station, the sample IED, Sample5-7
  copies). IEDs on 127.67.x.x with ports >= 20000. No `test_cids/`.
- XSD: the Go engine has no schema helper yet (lane I gap 1). Go ran with `STUDIO_XSD_SEED`
  (verdicts recorded from Python for the seed project's files and the "Waited" template).
- Comparisons: page text saved from both tabs and diffed after removing times, ids and counters;
  click-to-visible timings measured in the page (tab in front; a background tab runs 1 frame/s).

## Bugs found and fixed

| # | Severity | Engine | Bug | Root cause | Fix | Test |
|---|---|---|---|---|---|---|
| B1 | critical | Go | Gateway (IEC 104 slave) showed every point as "no value (IED not running or not connected)", sent quality IV+NT (192) and refused every SCADA command while the IEDs ran | The gateway read `ied["state"]` from the stored project. Python keeps the runtime state on the project IED dict; Go keeps it in the coordinator overlay and only applied it in GET /project | `app/ied_state_view.go`: the gateway gets a project view that overlays the runtime state on copies of the IEDs (published project unchanged); `project_view.go` uses the same overlay | `TestGatewayProjectsCarryRuntimeState` (fakes), `TestComposedGatewaySeesRunningIEDs` (real workers; fails before the fix) |
| B2 | high (test gate) | Go parity build | `test_recovery_that_stopped_to_protect_autosave_refuses_work` failed under `STUDIO_ENGINE=go` | The parity build had no seam for the `shutil.copyfile` fault the test injects | `store/recovery`: copy call through a package variable; `copy_seam_parity.go` (`//go:build parity`) sets it; `cmd/studio_server` installs the callback fault | `TestCopySeamFailureProtectsAutosave`; `unit/test_startup_gate.py` 3/3 with `STUDIO_ENGINE=go` |
| B3 | medium (speed) | Go | Test buttons on the SLD: symbol changed ~31 ms after the click on Go vs 10-17 ms on Python | A press publishes `signal` then `equipment_state` within 0.3 ms. The push stream wrote the first event at once and gathered for 16 ms before the rest, so the symbol's event waited 16 ms | `push/stream.go`: when the stream had to wait for events, it waits `Settle` (1 ms) once and reads again, so a burst leaves in one write; gather unchanged afterwards | `TestFirstBurstAfterQuietLeavesTogether` (also after an earlier burst; fails on both earlier versions) |
| B5 | medium (test gate) | Go suite | `TestComposedEventsMatchPython` failed 7 of 12 runs on master 650d0f0 with N3 natives (also seen pre-N3) | The scenario operated ds1 right after cb1; cb1's reported position (`equipment_state` value 2, observed feedback) arrives at the worker's pace, so its place before or after ds1's events was a race in both engines | The scenario waits 0.3 s after each of the two operates, identically in `record_python.py`, the runtime `parityScript` and the composed test | 0 of 12 (N3) and 0 of 6 (pre-N3) failures |
| B4 | low | both (frontend) | Gateway/Master/Converter role buttons greyed out while a role runs, tooltip only said what the role does | Missing reason text | `copy/master.ts` `roleTitle`/`ROLE_LOCKED`; `RoleSwitch.tsx` | `bugHuntQ2.test.ts` |

B3 server-side check (node client, `POST /equipment/<id>/test`, 20 presses, N3 natives): Go push
frame with the symbol state p50 **3 ms** (3/20 at 16-18 ms, when another event's gather window
was already open); Python `/events` p50 5 ms; before the fix half the presses took 16-20 ms.
The browser re-check of B3 could not be timed: the pane ran at 1 frame/s (window not visible)
when the build was ready. B1 was re-checked in the browser: 60/60 gateway points online.

## Function table: Go vs Python

Same text after normalising (times, ids, counters, value history) unless noted.

| Area | Functions used | Result |
|---|---|---|
| Devices | list, all sub-tabs, health, station signal flow, add sample IED, delete with confirm | identical |
| SLD | build from SCD preview/apply, add CB + line feeder, wire, diagram, Run/Stop, operate, test buttons, GOOSE / Who is reading overlays | identical (after B3 faster on Go) |
| Data points | writes, test mode (5 modes + simulation) | identical |
| Test points, generators, sequences | writes, values move, sequence "Failed at step 3 of 3" | identical |
| Gateway | points, start, IEC 104 GI, commands (dc 50005, sc 50001) | identical after B1 (GI group order follows IED ids) |
| Master, converter | connect 60 points, converter page | identical |
| IED Explorer, Find IEDs | connect, browse, operate (SBO enhanced), scan 127.67.x/29 port 20102 | identical |
| GOOSE/SV, Settings | overview, network/license/about, network preview + exact changes (not applied) | identical |
| Tools list | all tools | identical |

## Timings (click to visible, tab in front)

| Measure | Python pre-N3 | Go pre-N3 | Python N3 | Go N3 |
|---|---|---|---|---|
| Run | 416-859 ms | p50 ~267 ms | 405-527 ms | 250-299 ms |
| Stop | 181-326 ms | 168-312 ms | 61-89 ms | 43-84 ms |
| CB operate (p50) | 146-165 ms | 71-77 ms | 149-152 ms | 68-79 ms |
| Test button "Closed" (p50) | 21 ms | 33 ms | 10-17 ms | 17-38 ms before B3 fix; server side 3 ms vs 5 ms after |
| Explorer operate (watch row) | 182-312 ms | 56-148 ms | | |
| Find IEDs, 6 addresses | 0.9 s | 0.3 s | | |
| Sequence running visible | 353 ms (poll) | 60 ms (push) | | |
| Generator / test point write | 9-20 ms | 9-20 ms | | |

Gateway start: Go 309 ms vs Python 196 ms (second start). Master connect: Python 1059 ms vs Go
154 ms (each against the other engine's gateway). One Python Stop run of 571-971 ms happened while
an integrator `verify.py` ran; re-run 61-89 ms (load, not N3).

## F4 push plan, end to end on Go

1. `capabilities`: `push: 1`, 22 topics, paths. OK.
2. One `/api/v1/push` per tab, no `/api/v1/events`. OK.
3. Topic PUT per view (Devices: activity, health, goose.links, mms.access; test points:
   test.panel; sequences: scenario.runs; tools/GOOSE-SV: goose.overview, sv; settings: goose.share;
   gateway points: gateway.status, gateway.points{protocol: iec104}; commands: gateway.commands;
   Explorer/Find idle: activity only). OK.
4. Idle 60 s on Devices: 0 requests running; 2 PUTs + 1 app/update stopped (Python: 82/min
   running, 30/min stopped).
5. Byte identity with the real `pushStream.ts` in node: every topic's held text equals GET at rest
   (time-only fields excepted), 53 frames incl. 8 deltas.
6. Hidden tab: PUT `[]` within 18 ms, no polls; shown again: one PUT, no burst.
7. Server restart (dev token kept): polling meanwhile, EventSource retry every 2-3 s; after
   restart one /project, equipment/states and one PUT, then polling stopped.
8. Dropped gateway.traffic delta: exactly one resync PUT and a full frame; final text identical.
9. Duplicated tab (copied sessionStorage): new distinct tab id, both streams up.
10. 32 streams; the 33rd gets 503; the tab polls with 1-2-4-8-10 s backoff and joins when a slot
    frees (after the 30 s grace).
11. Python: one capabilities 404 + `/events`, activity polled every 2 s. OK.

Not tested: the Vite proxy path.

## Remaining issues (not fixed, same on both engines or by design)

- Export SCL is disabled for the demo IED; the reason is only in a tooltip.
- Find IEDs "This PC" is fixed to 127.50.0.x port 102.
- Activity list is flooded by report_control events while running.
- The "Test: ... set in the simulated IED" note stays after later actions.
- A test press makes the page re-read GET /project (studio.perform), on both engines.
- Product mode: after a server restart the open tab gets 401 (new token) on both engines.
- Go XSD gap (lane I gap 1): without seeded verdicts, adding a template shows "SCL schema
  validation is not available...".
- Held push text keeps old time-only fields (age_ms, connected_s, checked_at) by design; not shown.

## Verification

GOMAXPROCS=4, `-p 2`, N3 natives copy as `STUDIO_NATIVE_BIN`, venv Python as `STUDIO_PYTHON`.

- `go vet ./...` and `go vet -tags parity ./cmd/... ./internal/store/recovery/`: clean.
- `go test -p 2 -trimpath ./...` on 3a1998a + fixes: all packages ok except one
  `TestComposedEventsMatchPython` failure (B5). After the fast-forward to 650d0f0: app, push,
  recovery, toolsapi, gateway/service, cmd (also `-tags parity`) ok; B5 test 12/12 after its fix.
- Frontend: 228/228 tests; `npm run build` clean (existing chunk-size warning).
- Python in process, unit + integration: 453 passed, 8 skipped.
- `STUDIO_ENGINE=go` pytest, whole `studio/tests`, parity build with B2 seam, `STUDIO_XSD_SEED`
  from `xsd_record_plugin` (in-process run 456 passed / 104 skipped): **456 passed, 104 skipped,
  0 failed** (lane I2: 455 / 104 / 1).
- `check_file_lengths.py` PASS; `verify.py` on the final code: 6/7 PASS incl. the full Go engine suite and GOOSE engine; Documentation failed on one wrong relative link in this log, fixed and re-checked (documentation, file lengths, history PASS).
