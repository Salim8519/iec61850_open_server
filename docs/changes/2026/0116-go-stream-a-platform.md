# 0116: Go stream A merged: server core, security order, session cookie, SSE, launcher lock

Date: 2026-10-20. Workstream A of [ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)
(phase P0). Branch `go-a-platform` (8031fbd, 03ea4e6, f35ff1c) merged into `master` (3d2c43e). Stream
log: [a-platform](../../architecture/go-engine/streams/a-platform.md). Python stays the engine.

## Change

- `internal/pycompat`: CPython 3.12 `int()` (exact error text), `round`, `strip`/`split`/`isspace`,
  `casefold`, str `repr`, `isoformat`/`fromisoformat`, `fromtimestamp`, utf-8 / utf-8-sig decoding.
  Integrator addition: `pycompat.KeyError` (Python repr, double quotes for keys holding `'`);
  `apperr.KeyError` documented as ASCII-only (apperr cannot import pycompat: pycompat uses apperr).
- `internal/events`: 5000-event bus (ids from 1, `after`, `resync`), each event encoded once outside
  the lock; `GET /api/v1/events` (retry 1000, 10 s heartbeat, full replay, one writer per client).
- `internal/httpapi`: Python request order (Waitress bad URI, 403 with `active_frontend`, 401,
  402 `LicenseGate`, startup `StartupGate`, Werkzeug routing incl. 404/405/HEAD/OPTIONS/308),
  itsdangerous-compatible `studio_session` cookie, Flask/Werkzeug error bodies, `get_json`
  semantics, static `/` and `/assets` (`frontend.pak` or dist).
- `internal/launcher`: `studio.lock` byte lock shared with Python, `launcher.json`, options, engine
  choice (`--engine`, `SCADABENCH_ENGINE`, Python default). `internal/app`: composition; unported
  routes answer 501. `cmd/studio_server`: server mode, readiness line.
- `studio/tools/run_dev_engine.py`: Python by default (`run_dev.py` unchanged); `--engine go` builds
  and starts `studio/build/engine/studio_server.exe` with the same port, token and secret.

## Evidence

- pycompat vs CPython: 300,000 random cases + full code-point tables, 0 differences.
- Event bus vs Python `EventBus`: 4,000 operations, 0 differences; publish 3.7 µs (~270k events/s).
- Flask parity: frozen Python app vs Go on 58 requests × 5 variants (dev, packaged, unlicensed,
  startup failed, still opening) = 290 answers identical in status, compared headers and body bytes.
- Session cookies accepted across engines both ways; tampered cookie refused; lock: each engine
  refuses while the other holds it; `launcher.json` byte-identical.
- Integrator: branch touches only owned paths; `go vet` clean; `go test -p 2 ./...` all pass
  (server build test 29 s); `KeyError` repr test added; `verify.py` on `master`: see ledger line.

## Limitations

- Not done: `--network-helper`, `--self-test`, packaged single-instance focus, app window,
  license/update service wiring, background recovery, Go server logging.
- Differences not reachable from the UI: `Allow` header order sorted (Python's varies per process),
  no `Server: waitress` header or 16-thread cap, compressed cookies read but never written,
  non-ASCII `Content-Disposition` fallback not NFKD-normalised.
- `start-studio.cmd` still starts `run_dev.py`; switching it is a later shared-file change.
- `verify.py` now also runs the Flask parity test (~10 s) and two server builds (~25 s).
