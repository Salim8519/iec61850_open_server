# 5. Gates per phase

Python remains the shipped engine until G5 passes. Every gate below is cumulative: a
later phase re-runs all earlier gates. "Pass" means exit code 0 **and** zero skips that
were not also skipped (with the same reason) in the Python reference run.

## Common commands (referred to as C1..C9)

```powershell
# C1 repo rules (file lengths, architecture, docs, ledger, specs, go test of goose_engine)
python studio/tools/verify.py
# C2 Python reference, all suites (repo root on sys.path; native + Npcap present on the rig)
cd studio; $env:PYTHONPATH=".."; .venv\Scripts\python -m pytest -q -rs
# C3 harness fidelity: Python over real HTTP must equal C2 for class a + b
$env:STUDIO_ENGINE="python-http"; .venv\Scripts\python -m pytest -q -rs tests\integration tests\protocol tests\unit\test_infeed.py tests\unit\test_startup_gate.py tests\unit\test_startup_recovery.py
# C4 Go over the same tests
$env:STUDIO_ENGINE="go"; (same selection as C3)
# C5 Go unit ports, vet, race (race needs CGO only for this run, never for the product)
cd studio\core; go vet ./...; go test ./...; $env:CGO_ENABLED="1"; go test -race ./...
# C6 frontend
node --experimental-strip-types --test studio/tests/frontend/*.test.ts
# C7 SCL corpus, both engines + Python self-diff
python studio/tools/parity/corpus.py --engines python,python; python studio/tools/parity/corpus.py --engines python,go
# C8 golden replay (committed Python transcripts) + dual-run
python studio/tools/parity/replay.py --golden studio/tests/golden --engine go
python studio/tools/parity/replay.py --engines python,go studio/tests/golden/scenarios
# C9 parity report: routes, event kinds, inprocess budget, spec rules with Go twins
python studio/tools/parity/report.py --strict
```

## G0 - Harness (no Go product code yet)

| Check | Pass criterion |
| --- | --- |
| C1, C2, C6 | pass; C2 collects 550 tests with no collection errors |
| C3 | identical pass/skip set to C2 for the selected files (`python-http` == `python`) |
| C7 self-diff | zero differences on all corpus cases (Python is deterministic) |
| Golden self-diff | every scenario recorded twice (`PYTHONHASHSEED` 0 and 1): zero diffs; NaN occurrences = 0 |
| Recorder privacy | recorder refuses an unlisted SCL hash (dedicated test) |
| Perf baseline | `perf.py --engines python` evidence stored with hardware provenance |

## G1 - Pure core: SCL + domain in Go (no HTTP yet)

| Check | Pass criterion |
| --- | --- |
| C5 | Go twins exist for all `domain` (29 files) and `scl` (10 files) tests, all pass |
| Vectors | Python tests read the same `tests/vectors/**` files and still pass (C2) |
| C7 | every corpus case: bytes equal for `cfg`, `source`, export, error strings, warnings; JSON rules pass for the rest |
| Generators | CPython MT19937 vector (10 000 rows) equal |
| Perf (micro) | Go <= Python median for import / cfg build / export of demo, model_substation and the 20 MB case |
| C1 | file-length rule holds for Go (`check_file_lengths.py` extended to `.go`) |

## G2 - Projects, persistence, stateless HTTP

Scope: session/security, project CRUD/open/save/download, `scl/*`, `ieds/*` (no run),
`sld`, `signal-flow`, `preferences`, `license/*`, `app/update`, `network` previews.

| Check | Pass criterion |
| --- | --- |
| C4 | all a-pure and a-seam tests pass for `go` |
| Cross-read | Python-written data folder opened by Go and vice versa: `autosave.scadabench`, `preferences.json`, `tool-settings.json`, `update_check.json`, `recovery/`, license home, network journal - byte-identical re-save, no migration prompt |
| Cookie | `python-http` cookie accepted by `go` (dev reload switch) |
| C8 | golden scenarios for these routes: zero diffs |
| Perf | each route p50/p95 Go <= Python; cold start-to-ready Go <= Python |

## G3 - Runtime: workers, coordinator, events

Scope: `runtime/*`, `events` SSE, `equipment/*`, `signals/*`, `scenarios`, `traces`,
`test-panel`, `ieds/<id>/test-mode`, `mms/access`, energization, interlock hints.

| Check | Pass criterion |
| --- | --- |
| C4 | b-app 9 files + b-coord rewrites pass for `go` |
| b-ipc with Go cfg | 19 files pass with `model_for` = Go |
| IPC tap | per-worker frame sequences equal in all runtime scenarios |
| SSE | framing bytes, causal event order, resync behaviour equal |
| Orphans | zero surviving `studio_*.exe` after every teardown, both engines |
| Perf | run start/stop time, operate -> event p50/p95, operate -> independent client p95: Go <= Python |

## G4 - Acquisition, tools, gateway, master, network apply

| Check | Pass criterion |
| --- | --- |
| C4 + C8 | all remaining routes; route coverage 125/125 success + error; 48/48 event kinds |
| Wire evidence | `test_goose_*_wire`, `test_gateway_wire`, `test_master_*`, `test_sv_listen` pass with the Go engine running the app (Npcap rig, `STUDIO_GOOSE_ADAPTER`) |
| Benchmark (HTTP-driven) | 20 IEDs, 50 000 DA, startup < 30 s, 1000 updates/s for 10 min, 0 lost events: Go <= Python on every metric |

## G5 - Release candidate: Go selectable, then default

| Check | Pass criterion |
| --- | --- |
| Everything above | re-run on the **untagged** release binary for every seam-free suite (C8, UI, perf, b-ipc, b-indep) |
| C9 strict | 0 `inprocess` tests without a Go twin, 0 `InProcessOnly`, every spec rule names its Go test |
| Raw bytes | API bodies byte-equal (not only canonical); `numtype` diffs = 0 |
| Launcher | c-launcher + `test_release_guards` + `test_app_window` behaviour equal (single instance, show-again, close stops app, data-folder lease shared between engines: Python and Go refuse each other's live folder) |
| Packaging | `packaging/test_portable` against the Go-based build **only when the owner asks for an EXE** |
| Acceptance benchmark | 1 h at 1000 updates/s, < 4 GB, both engines, Go <= Python |
| UI gate | below, zero differences |

## Performance gate (applies from G1, per scope)

Tool `studio/tools/parity/perf.py --engines python,go --runs 7` (engine-neutral: drives
HTTP and reads worker/IPC timing; `benchmark_runtime.py` today calls the Python
`Coordinator` directly, so it gets an HTTP mode). Runs alternate engines
(P, G, P, G, ...) on the same machine, AC power, no other ScadaBench running; provenance via
`benchmark_provenance.capture`.

| Timing | Measure |
| --- | --- |
| Cold start | spawn -> ready line; spawn -> first 200 `GET /project`; with a 20-IED autosave -> `startup` done |
| SCL | preview, validate, import (demo, model_substation, 20 MB case); export-scl per IED |
| Project | save, open, download, upload restore (20 IEDs) |
| Run | start 6 demo IEDs -> all `ied_runtime` running; stop -> all workers exited |
| Control | operate -> `equipment_state` event; operate -> `studio_probe` sees value (200 ops) |
| API under UI load | recorded `sharedPoll` mix for 60 s: p50/p95/p99 per route |
| Throughput | 1000 updates/s: applied rate, lateness max, lost events (must be 0) |
| GOOSE/SV/pulses | existing `test_goose_engine_latency`, pulse start->wire, health refresh cycle |

Criteria, no tolerance band:
- every timing: Go median <= Python median, and Go p95 <= Python p95 for distributions;
- if Go loses by < 3 %, rerun that timing with 15 runs; the rerun decides, still `<=`;
- memory: backend process working set (natives excluded, `process_memory.working_set`)
  idle-after-import, peak during throughput and end-of-run: Go <= Python for each; total
  with workers < 4 GB (acceptance); end-of-run minus +10 min sample shows no growth trend;
- CPU: mean backend CPU during the API-load and throughput runs Go <= Python;
- thread and handle counts reported (not gated) to spot leaks.
Results are stored as JSON evidence linked from the change record of the phase.

## Final "no user-visible difference" UI gate

Tool `studio/tools/parity/screens.py --engines python,go --size 1366x768` (Playwright with a
pinned Chromium; adding it as a dev dependency needs an ADR - otherwise drive the built-in
browser pane with the same script). Same `frontend/dist` build hash for both engines.

1. Load the demo station through the golden scenario `ui/demo-station` (same diagram,
   same IED settings), `prefers-reduced-motion: reduce`, device scale factor 1, offline.
2. Visit, in both engines: the five places (Devices, SLD, Test, Tools, Settings); SLD with
   both signal-layer switches on; each tool page (Find IEDs, IED Explorer, Gateway,
   Master/Converter, GOOSE & SV traffic); dialogs: IED settings, export SCL, removal
   preview, network plan preview, test-mode card, license; run states stopped, running,
   after operating CBR_F1, after a test-mode change; the guided tour first step.
3. Per view compare:
   - PNG screenshot **pixel-exact** after masking a fixed, reviewed list of live regions
     (clock/time cells, event-log timestamps, live sparklines, frame counters);
   - DOM text snapshot (accessibility tree names/values) exact after the section 3
     normaliser;
   - `studio/tools/clip_check.js` returns `ok: true` in both (no truncated text);
   - browser console: zero errors/warnings in both;
   - request sequence (method + path, polling counts per route within +-1).
4. Any difference fails the gate; mask list changes need a change record.

Only after G5: the engine selector may default to Go; Python stays selectable for one
release with the same parity CI so a regression can be bisected by engine.
