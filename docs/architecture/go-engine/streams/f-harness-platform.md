# Stream F: harness (P0) and platform (P5)

Branch `go-f-harness` (cut from `go-engine` @ c983a8f). Owner of `studio/tests/parity/`, the engine
switch in `studio/tests/conftest.py`, `studio/tools/benchmark_engines.py` (+ `studio/tools/engine_bench/`),
`studio/engine/internal/{netcfg,licensing,updates}` and this log. Shared contracts (`pyjson`, `api`,
`apperr`, `go.mod`) were not changed.

## 1. Parity harness (tests/03-parity-harness.md)

Switch: `STUDIO_ENGINE=python` (default, unchanged), `python-http`, `go`. The conftest change only
registers two markers and, for HTTP engines, registers `parity.plugin`, which installs
`remote_create_app` as `studio_app.app_factory.create_app` before any test module imports it. No test
file was edited.

| Piece | File | Notes |
|---|---|---|
| Spawn + readiness + 401 check + orphan check | `parity/launch.py`, `parity/jobobject.py` | Job object `KILL_ON_JOB_CLOSE`; after shutdown any `studio_*.exe` left in the job fails the test |
| python-http server (test-only) | `parity/server.py` | real `create_app` + Waitress, `threads=16`, `port=0`; no lease/logging/launcher.json so the data folder equals in-process |
| Inspect API | `parity/inspect_api.py`, `parity/inspect_ops.py` | `/api/v1/_parity/*`, parity key header, outside product security |
| Client with Flask's test-client signature | `parity/http_client.py`, `parity/response.py` | `json=` encoded like Flask (`sort_keys`, `ensure_ascii`), werkzeug-style multipart, cookie jar |
| `app.extensions[...]` proxies | `parity/proxies.py`, `parity/live_view.py` | live, write-through project views addressed by id, so `ied.update(ip=...)` and later reads behave as in process |
| Seams | `parity/seam_translate.py` (harness), `parity/server_seams.py` (engine), `parity/callbacks.py`, `parity/wire.py` | see decisions D1-D4 |

Decisions:
- **D1 Seams are data first.** A real `LicenseService`/`UpdateService` handed to `create_app` becomes
  data (home, enforce, hwid value, frozen clock, test keys as hex), so the engine under test runs its own
  licensing/update code. The update opener (outbound HTTP) is a callback seam.
- **D2 Callback seams** (JSON over loopback HTTP, `POST /call {id,args,kwargs}` -> `return|raise|original`)
  replace only what a test cannot control from outside: the network fakes (`_list_adapters`, `_list_rules`,
  `_probe`, `_elevate`), duck-typed services (`UnlicensedService`) and `monkeypatch.setattr` on
  `studio_app.*` targets (recorded by the plugin, installed before `create_app` runs). A patched function
  that touches an engine object (e.g. `held(projects)` calling the original) answers `original`, so the
  engine runs the real function after the fake's side effects (the wait). Go can implement the same JSON
  protocol for its named seams.
- **D3 `environ_base={"REMOTE_ADDR": ...}`** is carried by `X-Parity-Remote-Addr`, accepted only with the
  run's parity key; parity headers are stripped before the product app sees the request.
- **D4 Repo root not added to `sys.path` in the default mode** (keeps "python unchanged"); the two history
  tests still need `PYTHONPATH=..` as before. HTTP engines put `studio/backend` and `studio/tests` on the
  child's `PYTHONPATH`.
- Go parity binary default: `studio/engine/build/studio_server_parity.exe` (`STUDIO_GO_BIN`), same CLI:
  `--port 0 --no-browser --data-dir --native-exe --frontend --parity-seams`.

Evidence (this worktree has no native build, so native-dependent tests skip in both modes; running them
here would also bind the owner's 127.50.0.x addresses):

| Run | Result |
|---|---|
| `python` reference, `PYTHONPATH=.. pytest -q -rs` | 446 passed, 104 skipped (48 s) |
| `python-http`, same command | 446 passed, 104 skipped (263 s); skip list identical line by line |

## 2. Golden replay and SCL corpus

- `parity/replay/`: recorder (`record.py`, SSE tap `sse.py`), normaliser (`normalise.py`, rules in
  `golden/normalise.json`), lexeme-preserving JSON (`jsonlex.py`), diff (`diff.py`, classes value, numtype,
  order, missing, extra, header, status, event), privacy manifest (`manifest.py`), CLI
  `python -m parity.replay record|golden|dual|selfdiff` (run from `studio/tests`, backend on `PYTHONPATH`).
- Privacy: uploads must hash-match `golden/generated/public_manifest.json` (demo station, the three public
  `scd/` files, corpus cases); IED names in the project must be in the manifest or declared template names;
  a refusal deletes the partial transcript (`test_golden_rules.py`).
- Scenarios (public demo station + sample IED): `session/auth`, `scl/demo-import`, `project/demo-diagram`,
  `errors/basics`; committed Python transcripts in `golden/transcripts/`.
- Normaliser rule added beyond the plan: `id_keyed_order` - objects keyed only by uuid ids are ordered by
  `<uuid:n>` because jsonify sorts random ids differently every run (`/health` `modes.ieds`). Found by the
  self-diff; raw byte order is a G5 check.
- SCL corpus (shared with stream B): `parity/corpus/` generator (`scl_corpus.py`, case modules
  `cases_bytes|xml|model|values|comms.py`), inputs + manifest in `parity/corpus/fixtures/`, runner
  `python -m parity.corpus.run selfdiff|dual|record`. 182 cases (93 accept, 35 warn, 54 reject as the frozen
  Python decides); the two 20 MB size cases are rebuilt on demand and hash-checked, not committed.

Evidence: see section 5.

## 3. Benchmarks (`studio/tools/benchmark_engines.py`)

HTTP only, product launcher without `--dev`, IEDs moved to `127.66.<block>.<n>:18700` through the project
archive (download, rewrite, `POST /project/open`), Job-object sampling of backend and children (working set,
private bytes, CPU % of one core, handles), provenance (source hash, natives hashes, versions, CPU).
S1 idle, S2 generators, S3 scale, S4 operate (+ studio_probe confirmation), S5 external MMS (studio_probe
readers), S7 real IEDs (second engine as targets on 127.66.1.x), S9 project ops, S10 fan-out. S6 (Npcap rig)
and S8 (gateway, firewall) refuse with a reason on this laptop. Go: `--engine go` with `STUDIO_GO_SERVER`.

## 4. Platform packages (Go, stdlib only, `CGO_ENABLED=0`, Windows calls via `syscall`)

Pattern for every package: Python generators in `testdata/make_*.py` import the frozen backend and write
vectors to `testdata/generated/*.json` (run `studio/.venv/Scripts/python.exe -I -B <script>` from the repo
root); Go tests read the vectors and compare bytes. Only fakes for anything that changes the machine.

| Package | Proven against Python | API for other streams |
|---|---|---|
| `internal/licensing` (+ `licensing/pytext`) | certificates, Ed25519 (89 cases incl. non-canonical), policy (120 combinations), hwid compose/SMBIOS parsing, ~3200 `fromisoformat` cases, time anchor vs `runs.jsonl`, 24 full service scenarios through Python's `api/license.py` (status/body/files byte-identical, CRLF, revoke rename, 60 s cache), 22 client cases against `httptest`, `go_vector.json`; real `MachineID()` equal to Python on this PC (checked once, not stored) | `Gate{Enforced(); Licensed()}`, `Require(g, path)`, `Exempt(path)`, `RequiredBody()` (Flask bytes of the 402), `RequiredStatus/Error/Code`; `ErrorResponse(err)` for `/license/*`; `Service` with `Options` seams (home, data dir, enforce, hwid, clock, keys, client factory) |
| `internal/updates` | versions, `_safe_url` (37), `clean_record`, `_verify` (17), 8 service scenarios (ETag/304/404/500/offline/bad_answer/no_server/manual gap/dismissed), `update_check.json` bytes (`json.dumps(indent=2)`) | `New(Options{Directory, ServerURL, Current, Keys, Opener, Clock, Logf})`, `Status/Configure/Check/Due/Run`, `StartBackground` off with `SCADABENCH_UPDATE_CHECK=off`; `*ConfigError` -> 400 |
| `internal/netcfg` | adapter normalisation + words, address plans (160 random), change/gateway plans (fingerprints, MMS/gateway rule names, texts), journal bytes (CRLF, ISO time), 116 helper validation cases with identical refusal texts, runner argv/result bytes, 10 full NetworkService/GatewayFirewall flows with fakes (bodies, request files, journal, helper argv); read-only differential adapter listing Go vs Python: 8 adapters identical | `AdapterSource`, `Executor`, `Elevator`, `Prober`, `Binder`, `Projects` interfaces; `NewNetworkService`, `NewGatewayFirewall`, `HelperMain` (`--network-helper`), pure plan/rule-name functions, `Journal` |

Known differences (documented in the package docs, none reachable through files the app writes): int64
clamping of absurd license integers; naive `runs.jsonl` times use Go's local zone; UTF-16/32 JSON request
files refused; non-UTF-8 PowerShell output decoded per byte; some Python-internal ValueError/TypeError texts
(uncaught in Python -> generic 500 in both); a few non-ASCII `lower/upper` cases. Not ported: HTTP route
handlers for `/license/*`, `/app/update*`, `/network*` (stream A / integrator), `ed25519_sign.py`
(build-only), the dev-mode `run.py` helper launch, the `integrity.EXPECTED` build step.

## 5. Evidence

| Check | Result |
|---|---|
| `python studio/tools/verify.py` | all PASS (file lengths, architecture, docs, history, specs, GOOSE go test, engine go test) |
| `go vet ./...`, `go test -p 2 -count=1 ./...` in `studio/engine` (GOMAXPROCS=4) | ok: licensing, licensing/pytext, netcfg, pyjson, updates |
| Full suite `python` / `python-http` | 446 passed, 104 skipped each, identical skip list (natives absent in the worktree) |
| `tests/parity` harness self-tests | 9 passed in `python` and in `python-http` |
| Golden self-diff (seeds 0/1), 4 scenarios | 0 differences after the `id_keyed_order` rule; 0 NaN |
| Golden replay vs committed transcripts (`python-http`) | 0 differences |
| Corpus self-diff, all 182 cases (seeds 0/1, one engine per case, ~70 min) | 0 differences, 0 NaN |
| Benchmarks (owner limit: small scale) | `f-evidence/generated/bench-python-s1-s9-{1,2}ied.json` |

Python baseline (Ryzen 9 5900HS, 16 logical CPUs; 5000 points per IED; 60 s idle; own copies of the natives):

| IEDs | ready | Run | Stop | backend WS idle | backend CPU idle | workers WS / CPU | S9 save / edit / delete IED |
|---|---|---|---|---|---|---|---|
| 1 | 1.59 s | 0.13 s | 0.02 s | 73.9 MB | 0.47 % | 27.3 MB / 2.07 % | 6.9 / 19.0 / 57 ms |
| 2 | 1.86 s | 0.24 s | 0.03 s | 81.1 MB | 0.29 % | 36.4 MB / 4.36 % | 6.1 / 6.4 / 137 ms |

Backend = the venv `python.exe` launcher stub plus the interpreter (grouped by image name); workers =
`studio_ied.exe` + `studio_goose.exe`. CPU is % of one core over the window.

## 6. Open items and limits

- Native-dependent tests (b-app, b-ipc) were skipped in both modes here: the worktree has no native build and
  they bind the owner's 127.50.0.x addresses. The harness supports them (job object, orphan check, `supervisor.read`,
  `events.after`), but `python-http` fidelity with natives still has to be run on a free machine/rig.
- Benchmarks S2-S5, S7, S10 are implemented but were not run (owner's RAM limit: only S1 with 1-2 IEDs now); S6
  (Npcap) and S8 (gateway/firewall) refuse on this laptop by design. S3 and the 1 h soak are later.
- Golden scenarios cover session, SCL import, diagram, errors on the demo station; runtime scenarios
  (`wait_event`, operate, IPC tap `studio_ipc_tap.exe`), route coverage report (C9) and the UI-recorded tour
  are not done. Corpus Python goldens are not committed (large); stream B records them with
  `python -m parity.corpus.run record --out ...` when it needs them.
- Line endings: corpus fixtures are byte-exact (`corpus/fixtures/.gitattributes`: `* -text`). The public files
  outside it (`scd/*.scd`, the demo station) are checked out with the machine's `core.autocrlf`, so their hashes in
  `golden/generated/public_manifest.json` match this checkout (autocrlf=true); on another setting the recorder
  refuses them until `python -m parity.replay.manifest --write` is re-run (the `corpus:public_*` copies are exact).
- Seams translate only what the 28 files use: a `LicenseService` with a custom `client_factory` or
  `legacy_hwid_provider` raises `NotImplementedError`; clocks given as lambdas are frozen to their first value.

## 7. Needs from other streams

- Stream A: a parity build (`-tags parity`) of `cmd/studio_server` at `studio/engine/build/studio_server_parity.exe`
  (or `STUDIO_GO_BIN`) with the CLI `--port 0 --no-browser --data-dir --native-exe --frontend --parity-seams`,
  the ready line `ScadaBench: http://127.0.0.1:<port>`, the `/api/v1/_parity/*` table of
  `tests/parity/inspect_ops.py` (tagged JSON of `parity/wire.py`), the seams schema in `parity/server_seams.py`
  (license/updates data seams, callback protocol of `parity/callbacks.py`, `X-Parity-Remote-Addr` with the parity
  key); the product server for benchmarks (`STUDIO_GO_SERVER`) printing `Open in your browser: <url>#session=<token>`.
  Stream A consumes `licensing.Gate`/`Require`/`RequiredBody` for the 402 step.
- Stream B: corpus inputs and manifest in `studio/tests/parity/corpus/fixtures/`; runner `parity.corpus.run`.
- Integrator: change record, ledger line and INDEX links for this stream; proposed ADR text below.

## 8. Proposed ADR text (parity harness and seams)

Context: Go must be judged by the unchanged Python tests. Decision: an HTTP engine switch in
`tests/conftest.py` with a test-only Python server, data-first seams plus a JSON callback protocol for
OS/network/HTTP boundaries and named fault points, a parity inspect API outside product security, live
write-through project views, Job-object orphan checks, golden transcripts with a versioned normaliser and a
public-only input manifest. Compatibility: the default `python` mode is unchanged; product code untouched.
