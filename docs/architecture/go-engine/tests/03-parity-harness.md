# 2. Parity harness: run class (a) and (b) unchanged against either engine

Goal: the same pytest files judge Python and Go. Python stays the reference; a Go result
only counts when the identical test, with identical inputs, passes.

## Engine selection

| Variable | Values | Meaning |
| --- | --- | --- |
| `STUDIO_ENGINE` | `python` (default) | today's in-process `create_app` + Flask test client; fastest, unchanged |
| | `python-http` | spawn the Python backend as a real Waitress server; tests talk HTTP |
| | `go` | spawn the Go core (`studio_core_parity.exe`); tests talk HTTP |
| `STUDIO_GO_BIN` | path | Go parity binary, default `studio/core/build/studio_core_parity.exe` |
| `STUDIO_NATIVE_BIN` | path | unchanged; both engines get the **same** native folder |

`python-http` is mandatory: it proves the HTTP shim itself changes nothing. A test that
passes in `python` but fails in `python-http` is a harness bug and blocks every gate.

## Process launch (both engines, one CLI contract)

```
<engine> --port 0 --no-browser --data-dir <tmp>/data
         --native-exe <STUDIO_NATIVE_BIN>/studio_ied.exe --parity-seams <tmp>/seams.json
```

- Python: `python -m parity.server ...` - a **test-only** module under
  `studio/tests/parity/` that builds `Settings`, applies seams, calls the real
  `create_app`, mounts the parity inspect blueprint and serves with
  `waitress.create_server(app, host="127.0.0.1", port=0, threads=16)` exactly as
  `entrypoint._serve` does. The product backend gains no code.
- Go: the parity binary is the product `main` built with `-tags parity`; the tag only adds
  the seams loader and `/api/v1/_parity/*`. Gate G5 re-runs every seam-free suite against
  the untagged release binary, so the tag can never hide a product difference.
- Readiness: both print the existing line `ScadaBench: http://127.0.0.1:<port>` and write
  `<data>/launcher.json`; the harness waits (30 s max) for the line, then requires
  `GET /api/v1/project` -> 401 with the "#session=" message.
- Environment passed to the child: `SCADABENCH_UPDATE_CHECK=off`, `LOCALAPPDATA=<tmp>/home`
  (license home and legacy-folder logic never touch the developer's real profile),
  `PYTHONHASHSEED=0` for Python.
- Each child runs in a Windows Job object (`KILL_ON_JOB_CLOSE`). Teardown:
  `POST /api/v1/_parity/shutdown` (= `runtime.close()` then exit), wait 10 s, then the
  harness lists the job's processes: any surviving `studio_*.exe` **fails the test**
  (orphan workers are a user-visible difference), then the job is closed.
- Before the session the harness hashes every `studio_*.exe` in `STUDIO_NATIVE_BIN` and
  stores it in the junit properties; a hash change mid-session aborts (no stale natives).

## Session and token handshake

The seams file carries `token`, `secret_key`, `admin_origin` exactly as tests pass them
to `create_app` today, so tests keep their fixed tokens ("test-session-token") and their
401/403 message assertions (`ScadaBench.exe` vs `start-studio.cmd`).
The HTTP client shim keeps a cookie jar (`studio_session`), sends `Host: 127.0.0.1:<port>`
and no `Origin` unless the test sets one - the same as Flask's test client.
Cookie compatibility: Go must accept Flask/itsdangerous `studio_session` cookies signed
with the dev secret (`studio-dev-secret`) so a dev reload that switches engine keeps the
browser tab signed in. Covered by a dedicated parity test: cookie minted by
`python-http`, request served by `go` -> 200.

## Minimal conftest change (~40 lines, one new package)

```python
# studio/tests/conftest.py (additions)
REPO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO))                       # history_log / runtime_log_query collect
ENGINE = os.environ.get("STUDIO_ENGINE", "python")
if ENGINE != "python":                               # before any test module imports it
    import studio_app.app_factory as factory
    from parity.remote_app import remote_create_app
    factory.create_app = remote_create_app

def pytest_configure(config):
    config.addinivalue_line("markers", "inprocess: needs Python objects; skipped for HTTP engines")
    config.addinivalue_line("markers", "engine_neutral: native/wire contract; engine only supplies cfg")

def pytest_collection_modifyitems(config, items):
    if ENGINE == "python": return
    for item in items:
        if item.get_closest_marker("inprocess"):
            item.add_marker(pytest.mark.skip(reason=f"inprocess under {ENGINE}"))
```

Because `from studio_app.app_factory import create_app` executes after conftest, the 28
files that call `create_app` get `remote_create_app` with **no edit**. It returns a
`RemoteApp`:

- `.test_client()` -> `HttpClient` with Flask's call signature: `get/post/put/delete/patch`
  with `json=`, `data=` (dict of `(BytesIO, filename)` tuples becomes multipart, exactly as
  werkzeug encodes it), `query_string=`, `headers=`, `content_type=`. Responses expose
  `.status_code`, `.json` (None unless `application/json`), `.get_json()`, `.data`,
  `.text`, `.headers`, `.mimetype`.
- `.extensions[name]` -> proxies that forward the attribute calls listed in section 1
  to `/api/v1/_parity/<service>/<method>`; anything else raises `InProcessOnly("<attr>")`
  so an unported internal access fails loudly rather than silently passing.
- `.config["LAUNCH_TOKEN"]` -> the seam token.

## Parity inspect API (parity builds only)

| Endpoint | Mirrors Python | Used by |
| --- | --- | --- |
| `GET _parity/project/raw` | `projects.project` (full, incl. `cfg`, `source`) | api, import_selection, sld_import, network |
| `GET _parity/project/export?ids=` | `projects.export(...)` bytes (base64) | import_selection, scl_export, test_mode |
| `POST _parity/project/restore` | `projects.restore(bytes)` | test_api |
| `POST _parity/project/diagram` | `projects.diagram(doc)` without revision check | b-app files |
| `POST _parity/project/add_template` | `projects.add_template(name)` returning the IED | b-app, scl_export, infeed |
| `POST _parity/project/ied` | in-place edit `ied.update(ip=, port=)` | b-app (they mutate the returned dict) |
| `GET _parity/events?after=` | `events.after(n)` non-blocking | api, b-app |
| `GET _parity/runtime/state` | `runtime.state` | api, mapping, startup_recovery |
| `POST _parity/runtime/start` / `close` | `runtime.start()` / `close()` | b-app |
| `GET _parity/runtime/read?ied=&ref=` | `supervisor.read(ied, ref)` | mapped_equipment_control |
| `GET _parity/runtime/workers` | `supervisor.workers` ids + pids | startup_recovery |
| `POST _parity/startup/wait` / `GET .../done` | `startup.wait(t)` / `done.is_set()` | tools, gateway, startup_* |
| `POST _parity/shutdown` | close everything, exit 0 | harness |

Inspect payloads use the engine's normal JSON writer, so they are also diffed.
`RemoteApp` mutation of a returned dict (`ied.update(ip=...)`) is detected by a
write-through dict proxy that calls `_parity/project/ied`.

## Seams file (one schema, both engines)

```json
{"token": "license-token", "secret_key": null, "admin_origin": null,
 "license": {"enforce": true, "hwid": "<HWID>", "clock": 1760000100, "keys": "vectors/licensing/test_keys.json"},
 "updates": {"current": "0.1.13", "clock": 1000000, "answers": ["vectors/updates/answer.json"]},
 "network": {"adapters": [...], "rules": [], "reachable": ["10.0.0.5"], "elevate": "record-ok"},
 "faults": {"startup_recovery.copyfile": "OSError", "projects.save": "OSError", "recovery.hold": true}}
```

Seams replace only things a test cannot control from outside: clock, hardware id, signing
keys, outbound HTTP answers, Windows network calls and named fault points. Recorded
elevation requests are readable at `GET _parity/network/calls`.

## Files that need real edits (11) and why

| File | Edit |
| --- | --- |
| integration/test_network_api, test_network_setup_page | lambdas on `network._list_adapters/_probe/_elevate` -> `seams.network` data; assertions read `_parity/network/calls` |
| integration/test_license_api, test_update_api, unit/test_release_guards | build injected services through `seams.license(...)` / `seams.updates(...)` helpers that return the Python object in `python` mode and seam JSON otherwise |
| unit/test_startup_gate, test_startup_recovery | `monkeypatch.setattr(...)` -> `seams.fault(...)` (same helper patches in-process) |
| unit/test_infeed | split HTTP tests (stay) from domain asserts (`inprocess` + Go unit twin) |
| protocol/test_csv_runtime, test_scenario_runtime, unit/test_automation_failure | rewrite through public `PUT /scenarios`, `POST /scenarios/<key>/start`, `POST /signals/source`; keep the Python-object versions as `inprocess` until G3 |

Everything else in class (a)/(b-app) runs untouched.

## Class (b-ipc) and (b-indep)

These test the C++ workers and the Go GOOSE engine, not the backend. They stay Python and
are marked `engine_neutral`. The only engine output they consume is the SCL import result
(`import_scl(...)[0]` -> `cfg`, `signals`, `goose`). A one-line edit per file replaces it
with `model_for(source, names)`: `python` -> `import_scl`, `go` -> `studio_core_parity.exe
scl-dump --json`. Run them once per engine so the natives are exercised with Go-built cfg.
The Go IPC client (`internal/worker`) gets its own tests that replay the recorded IPC
transcripts from section 3 (same frames, same ids after normalisation).

## Inprocess budget

`inprocess` skips are counted and printed (`pytest -rs`); the parity report lists them by
file. Gate G5 requires **zero** `inprocess` tests without a Go unit twin and zero
`InProcessOnly` errors.

## Commands

```powershell
cd studio
.venv\Scripts\python -m pytest -q                                        # reference
$env:STUDIO_ENGINE="python-http"; .venv\Scripts\python -m pytest -q tests\integration tests\protocol
$env:STUDIO_ENGINE="go";          .venv\Scripts\python -m pytest -q tests\integration tests\protocol -rs
```

Run serially (no xdist): b-app tests bind fixed lab IPs `127.50.0.x` and fixed ports.
Per-test server spawn costs ~1-2 s; acceptable for 76 HTTP tests. Do not share one server
across tests with a "reset" endpoint - state leakage would hide parity bugs.
