# Speed lane I: one composed Go server (S3, S4, S5, S7, S8, S10, S11)

Branch `speed-i` from master 2030e69 ([speed plan](../speed-plan.md), phase 1). Owns
`internal/app/**`, `cmd/studio_server/**` and `studio/tools/run_dev_engine.py`. No other
package was changed. Python stays frozen and the default engine. No EXE was built.
Master (lane F frontend 26b2b22, lane G push 8e24b7a) was merged in at 847c9a4.

## What is wired (`internal/app`)

`app.Compose(Options)` builds create_app's services in its order; `Engine.Close` stops them.

| File | Responsibility |
|---|---|
| `compose.go` | Options, Engine, order: project → runtime → live → tools → platform → routes → background |
| `wire_project.go` | `autosave.Fold`, XSD validator (+ verdict seeds), `service.New` (OnChange → `project_changed` on the bus, only a publish under the mutation lock), `recovery.StartBackground(RecoverStartup)` |
| `runtime_bus.go` | `events.Bus` as `runtime.Bus` (returns the `{"id","kind","timestamp",**payload}` envelope for the run log) |
| `wire_runtime.go` | `runtime.New` with `NativeDir` (from `--native-exe`), late-bound Live adapter, `LabAddresses` = IP Helper; runtimeapi routes (owner of `GET /project/test-panel`) |
| `wire_live.go`, `live_ports.go` | `acquisition.Manager` on `ports.ForAcquisition(coordinator)`, `Members = scl.SubscriptionMembers`, one `project.GooseDeclarations` → `goosedom.DeclarationsFromJSON`; `runtime.Live` adapter (`[]string` → set, `BusFor` object → map); acquisition `Projects` adds `RunID`, `SetTestMode`, `SaveReach` through `Service.Update`; adapters/Npcap and clock |
| `wire_tools.go` | one `studio_control` client, tool memory, Explorer (bus-only events), Find IEDs, gateway on `ports.Gateway(coordinator)`, master (`coordinator.emit`), converter; `Master.OnValues = Gateway.MarkRemote` (the change-driven gateway would otherwise miss converter values Python's 100 ms diff saw) |
| `wire_platform.go` | one `*licensing.Service` for the 402 gate and `/license/*`; updates asks `License.ServerURL`; network service + gateway firewall; platformapi |
| `project_view.go` | projectapi routes; `GET /project` overlays `IedStates()` (`state` when set, `goose_state` only while `HasGoose`); `POST /project/open` resets the runtime values of every IED, IED removal those of the removed IEDs |
| `startup.go` | gates (`licenseGate` panics on an `is_licensed` error → Flask 500; `startupGate` = Background), license verify + update loops, `_resume_gateway`: `Master.Autoconnect()` then `Gateway.Autostart()` only after recovery succeeded |
| `wire_push.go` | lane G push stream per [speed-g](speed-g.md) "Wiring notes": `push.Catalog(push.IndexRoutes(live, runtime, tools), {Running: State() != "stopped", Scanning: Scanner.Scanning})`, `push.New{Bus, Topics, Values: coordinator.Subscribe, Listen: project_changed → Gateway.ProjectChanged()}`, `OnChange(area.Changed)` on gateway, master, Explorer and scan; routes `GET /capabilities`, `GET /push`, `PUT /push/topics` mounted in the same area list (same 403/401/402/startup pipeline as `/events`, no server write timeout); `missing` logged and kept in `Engine.PushMissing` (empty) |
| `shutdown.go` | Close: loops → push area → gateway (`Stop(false)`, auto-start kept) / master → Live → Coordinator → Explorer/scan → control client, XSD helper → wait for recovery → `Service.Close` (final snapshot, journal removed); the caller then releases the lease |

`GET /mms/access` is served from `coordinator.MmsAccess()`: the runtime is the one owner of MMS
access (it receives the worker summaries); acquisition's copy is never fed. Test modes: the
route uses acquisition's `TestModes`, Run start uses the runtime's `applyTestModes` (one owner each).

`cmd/studio_server`: composes the engine, serves until Ctrl+C/SIGTERM (Shutdown 2 s, then
Close, waits for the serve loop), closes the engine, releases the lease. New: `--fold-journal
DATA_DIR` (takes the lease, folds, keeps both files on a mismatch), `--network-helper` →
`netcfg.HelperMain`, native/schemas/version resolution (beside the exe or up the source tree;
version from `studio_app/version.py`, `-X main.version` for a package; license enforced only
with `-X main.packaged=1`), `STUDIO_XSD_SEED` verdict files (dev). `run_dev_engine.py --engine
python` folds a leftover `studio/data/autosave.journal` with `studio_server --fold-journal`
before starting Python (builds the server only when a journal exists).

Parity build (`-tags parity`, `parity.go`, `parity_seams.go`, `parity_ops.go`): the
tests/parity CLI (`--frontend`, `--parity-seams`), token/secret/origin and license/update data
seams, `X-Parity-Remote-Addr`, and the inspect operations shutdown, project raw/snapshot/
revision/export/restore/add_template/scenarios, events, runtime state/close, startup
wait/done. Not ported (refused or 404): fault, object and update-opener callback seams,
network seams, project/set and project/update write-through, runtime/state writes,
supervisor reads, service/stop.

## Evidence (2026-10-20, GOMAXPROCS=4, go test -p 2 -trimpath, CPython 3.12 venv)

Native workers: copies of `studio/native/build/Release` in a scratch folder; `studio_control`
rebuilt from master into `studio/native/build/speed-i` (`-DSTUDIO_GATEWAY=OFF`, has
`control_mux_v1`). Loopback 127.67.x.y / 127.63.1.x, ports ≥ 20000; temp data and license homes.

| Check | Result |
|---|---|
| `TestFlaskParity` through `Compose` (5 variants × 58 requests) | 290/290 identical |
| `TestEveryPythonRouteServed` (Flask url_map via STUDIO_PYTHON) | 127/127 routes served, none extra, no GET answers 501 |
| `TestComposedClickPath` (template → diagram → Run → 3 CB operates → Stop → Close → restart) | pass; operated position restored after restart, journal folded on Close |
| `TestComposedEventsMatchPython` (runtime lane's script over HTTP vs `record_python.py`, `compare_events.py`) | identical: global 60, IED A 26, IED B 9 events |
| `TestComposedPushWiring` | `PushMissing` empty, `/capabilities` push 1 with topics, `/push` and `/capabilities` answer an anonymous request like `/events`; a template add (`project_changed`) reaches `Gateway.ProjectChanged` (`gateway.points` hook) |
| `TestEveryPythonRouteServed` after the push merge | 127/127 Flask routes; only the 3 push routes extra (`PushOnlyRoutes`) |
| `TestStartupGateHoldsUntilRecovery` | GET held while recovery runs, preferences not held, 503 `startup_recovery_failed` with the recovery text |
| `TestFoldBeforePython` (built binary hard-killed, `--fold-journal`, frozen Python `recover_startup`) | Python restores exactly the snapshot Go served |
| `TestComposedLockOrderStress` (STUDIO_STRESS=1, 60 s) | no stall, no 500: lifecycle 140, operate 126,778, gateway saves 65,134, edits 68,953, readers 53,423, SSE frames 71,415 |
| `python -m parity.replay dual --engines python-http,go` and `golden --engine go` | 4/4 scenarios, 0 differences |
| `STUDIO_ENGINE=go pytest` (studio/tests, parity build, no native build) | 434 passed, 104 skipped, 7 failed, 16 errors (python-http: 446 passed, 104 skipped). All 23 remaining: SCL without a seeded XSD verdict (10 + 3 follow-ups + 1 restore), callback seams not ported (network 4, fault/opener 4), service/stop 1 |
| Live UI comparison, same station on both engines, 27 GET routes | 23 byte-identical; 4 differ only by random uuid sort order (`running_ieds`, health `modes.ieds`) or `age_ms` |

### Measured (composed server, real workers)

| Measure | Python | Go |
|---|---|---|
| Canvas CB operate, click → symbol in the UI (demo station, 12 ops) | p50 ~156 ms, max 204 ms | p50 ~56 ms, max 68 ms |
| Same through HTTP, `TestComposedOperateTiming` (40 ops, keep 0) | — | answer p50 57.8 / p95 69.7 ms; symbol p50 56.5 / p95 68.5 ms |
| First operate after start (starts warm `studio_control`) | — | 138-548 ms |
| Test button (no MMS command), click → symbol | 8-11 ms | 7-13 ms |
| Diagram save `PUT /project/diagram` (small project) | 10.1 ms (browser) | 5.7 ms (browser); 1.2 p50 / 3.6 p95 ms (test) |
| Import open_substation.scd / open archive | 31 / 47 ms | 19 / 17 ms |
| Run 4 IEDs, click → "Running" | 495-528 ms | 291 ms |
| Stop 4 IEDs, click → stopped | 159 ms | 286 ms (slower: see gaps) |

The ≤ 50 ms click-to-symbol target needs `CommandKeep` > 0 (lane C: ~2-4 ms with a kept
association), an owner decision (visible MMS client row); with keep 0 each command opens an
association like `studio_probe`.

## Gaps and differences (remaining)

1. **`studio_xsd.exe` is not built**: any SCL import/open without a cached or seeded verdict
   answers 503 "SCL schema validation is not available". Tests and the dev run use Python
   verdicts (`testdata/xsd_verdicts.py`, `STUDIO_XSD_SEED`). Blocks a Go-only dev session.
2. **Stop is slower** (286 vs 159 ms in the UI): Live.Stop closes the GOOSE monitor, engine
   and SV listener processes one after the other before the workers stop; owner: lanes D/R.
3. First canvas command after start pays the `studio_control` start (~0.1-0.5 s); a warm-up
   at Run start would hide it (runtime lane).
4. Runtime values are not written back into the project (Python persisted them with the next
   save); after a restart Go shows initial values. Open/remove reset the value tables.
5. Respond copies in projectapi/toolsapi/liveapi/runtimeapi were not replaced by httpapi's
   writer: they are not identical (projectapi maps `ContractError`, liveapi omits
   Content-Length), so swapping is a behaviour change for those lanes.
6. liveapi goldens (fake workers) were not re-run through the composed server; the routes are
   mounted unchanged and the live UI comparison above matched.
7. Lab-mode preflight: an IP Helper failure lists no address (Python raised the OSError).
8. Parity build: the callback seams (fault, object, update opener, network) and the
   write-through/supervisor/service-stop inspect operations are missing (they need the JSON
   callback protocol of `parity/callbacks.py` in Go); `license.runs_file = None` is not
   representable in `licensing.Options`.
9. `studio/engine/build/` is not git-ignored (the parity binary lands there; not committed).
10. Two engines on one host share the `studio_session` cookie name (cookies ignore ports): a
    tab of the other engine signs the first one out. Same in Python; only seen when both run.
11. `TestComposedEventsMatchPython` failed once in a combined run after the push merge (output
    not captured); 5 reruns and a full `internal/app` run passed. Fixed port base 21300 is a suspect.
12. The push stream was not exercised in the live UI by this lane (lane F4 owns the client);
    only the composed route/hook wiring is tested here.
13. `go test ./...` without `STUDIO_PYTHON` in a worktree: `launcher.TestLeaseSharedWithPython`
    skips while holding its lease, so `t.TempDir` cleanup fails (lane A test, not changed here);
    with `STUDIO_PYTHON` set it passes.

## How to reproduce

```powershell
cd studio/engine; $env:GOMAXPROCS=4; $env:STUDIO_PYTHON='<venv python>'
$env:STUDIO_NATIVE_BIN='<copy of native/build/Release with a control_mux_v1 studio_control>'
go test -p 2 -trimpath -count=1 -v ./internal/app/ ./cmd/...        # STUDIO_TIMING=1, STUDIO_STRESS=1 for the opt-in ones
go build -trimpath -tags parity -o build/studio_server_parity.exe ./cmd/studio_server
# seeds while studio_xsd.exe is missing:
python -I internal/app/testdata/xsd_verdicts.py --backend ../backend --out seed.json ../../scd/open_substation.scd template:Feeder01
cd ../tests; $env:PYTHONPATH='..\backend;.'; $env:STUDIO_XSD_SEED='seed.json;..\engine\internal\scl\testdata\golden\xsd_seed.json'
python -m parity.replay dual --engines python-http,go
```
