# Stream A — platform-core (P0)

Branch `go-a-platform` (cut from `go-engine` = `master` at c983a8f). Owner of `internal/pycompat`,
`internal/httpapi`, `internal/app`, `internal/events`, `internal/launcher`, `cmd/studio_server` and
`studio/tools/run_dev_engine.py` ([workstreams](../workstreams.md)). Python is untouched and stays the
default engine.

## Delivered

| Package | Content | Python source copied |
|---|---|---|
| `pycompat` | `ParseInt` (int() of str: Unicode digits and spaces, `+`, `_`, 4300-digit limit, exact ValueError text with `%.200R`), `Round`/`RoundDigits` (half-even, negative ndigits, OverflowError), `Strip`/`LStrip`/`RStrip`/`Split`/`IsSpace` (incl. U+001C..U+001F), `Casefold` (full folding, generated table), `Repr` of str, `Isoformat` (timespecs, `+00:00`, offsets with seconds/µs), `FromIsoformat` (port of the C implementation incl. week dates, any separator, `,` fractions, `Z`), `FromTimestamp` (Windows CRT range), `TimeTime`, `DecodeUTF8Sig`/`DecodeUTF8` (CPython UnicodeDecodeError texts and positions), `DecodeUTF8Replace`/`DecodeUTF8With` | CPython 3.12.6 |
| `events` | `Bus` (ids from 1, 5000 ring, `After(cursor, wait, done)`, synthetic `resync`), each event encoded once with `json.dumps` rules (payload encoded outside the lock, envelope under it), `Area` serving `GET /api/v1/events` (`retry: 1000`, 10 s heartbeat, `id:`/`data:` frames, full replay, `int()` cursor errors) | `runtime/event_bus.py`, `api/events.py` |
| `httpapi` | Pipeline Waitress-URI check → leading-slash normalisation → 403 loopback/origin (`active_frontend`) → 401 → 402 (`LicenseGate`) → startup wait (`StartupGate`, 300 s, `project_opening`/`startup_recovery_failed`) → Werkzeug-style routing (404, 405 without `Allow`, automatic HEAD/OPTIONS, merge-slashes 308 redirect page) → handler; Flask session cookie (itsdangerous HMAC-SHA1, salt `cookie-session`, 31-day max age, `Vary: Cookie` when read); `POST /api/v1/session` incl. its 500 quirks; Flask/Werkzeug error bodies; `ReadJSON`/`ReadBody` (get_json silent/strict, 413/415/400, UTF-8/16/32 detection); static `/` and `/assets/` from `frontend.pak` (SHAKE-256 unmask) or a dist folder (send_file headers, Werkzeug ETag, `Content-Disposition: inline`) | `app_factory.py`, `api/security.py`, `api/startup_gate.py`, `launcher/frontend_assets.py`, Flask 3.1.2 / Werkzeug 3.1.8 / Waitress 3.0.2 |
| `launcher` | `AcquireLease` (`studio.lock`, `LockFileEx` byte 0 — excludes and is excluded by Python's `msvcrt.locking`), `WriteRecord`/`ReadWindowPID` (`launcher.json`), `ParseOptions` (argparse-compatible server flags, env defaults), `ResolveEngine` (`--engine`, `SCADABENCH_ENGINE`, default python), `LaunchToken`/`SecretKey`/`DevSecret` | `launcher/entrypoint.py`, `launcher/directory_lease.py` |
| `app` | Composition: bus, events area, other streams' areas, 501 placeholders for every Python route not yet ported (so 404/405 already match), httpapi server | `app_factory.create_app` |
| `cmd/studio_server` | Server mode of `entrypoint.py`: lease, port (dev: busy → exit text; packaged: OS port), `launcher.json`, readiness line `ScadaBench: http://127.0.0.1:<port>`, dev Vite lines / packaged `Open in your browser: ...#session=`, dev token and `studio-dev-secret`, Ctrl+C shutdown | `launcher/entrypoint.py` |
| `studio/tools/run_dev_engine.py` | Dev engine switch: `--engine python` (default) runs `run_dev.py` unchanged; `--engine go` builds `studio/build/engine/studio_server.exe` (`go build`, incremental) and starts Vite + the Go server with the same port, data folder, token and secret; `--print-command` dry run | `studio/tools/run_dev.py` |

## Evidence (2026-10-20, this worktree, Go 1.25.6, CPython 3.12.6 venv, Windows 11)

All commands with `GOMAXPROCS=4`, `-p 1|2`, `STUDIO_PYTHON=<main checkout>\studio\.venv\Scripts\python.exe`.

| Check | Result |
|---|---|
| `go vet ./...`, `go test -p 1 ./...` (CGO_ENABLED=0) | all packages PASS |
| `python studio/tools/verify.py` | all 7 checks PASS (incl. file lengths, Go engine tests with the Flask parity test) |
| pycompat vs CPython, seed 61850, 20 000 cases (default) and seed 7, **300 000 cases** | 0 differences; plus full code-point tables (isspace, isprintable, decimal digits, casefold over all 1 112 064 code points): 0 differences |
| events bus vs Python `EventBus` (capacity 7, 4000 random publish/after operations, patched clock) | 0 differences (bytes of every event and every `after()` result, resync and eviction included) |
| SSE framing test (replay, heartbeat, resync, live event) | PASS |
| **Flask parity** `internal/app TestFlaskParity`: the frozen Python app (`create_app` + Waitress, venv Python, temp data dir, ports ≥ 20000) and the Go app on the same 58 requests in 5 variants (dev, packaged, packaged-unlicensed, dev-startup-failed, packaged-still-opening) | **290/290 identical** status, `Content-Type`, `Content-Length`, `Vary`, `Allow` (as a set), `Cache-Control`, `Accept-Ranges`, `ETag`, `Last-Modified`, `Content-Disposition`, `Location`, normalised `Set-Cookie`, body bytes; pinned answers guard the harness |
| Cross-engine session cookie | Python cookie accepted by Go and Go cookie by Python (same answer, not 401); tampered signature refused |
| httpapi helpers vs Python (`httpapi_vectors.py`) | Flask/itsdangerous cookies at fixed times (byte-equal; compressed ones read), `urlsplit().netloc` 19 cases incl. ValueErrors, Werkzeug `request.args` 14 cases, `frontend.pak` written by Python unpacked, `Content-Disposition` 8 names: 0 differences |
| Lease vs Python `DirectoryLease` | Python refused with the exact text while Go holds `studio.lock`, and vice versa; both acquire after release; lock file content `0` |
| `launcher.json` vs Python `write_launcher_record` | byte-identical with and without `window_pid` |
| Built `studio_server.exe` | readiness line, `launcher.json`, dev session through the Vite origin, SSE `retry: 1000`, second launch on the same data folder refused; packaged mode falls back from a busy port and prints `#session=` + 43-char token |
| `BenchmarkPublish` (signal event, 7 payload keys) | 3.7 µs/op, 26 allocs → ~270 000 events/s on one core (budget 50 000/s) |

Requests covered by the parity test: static index/assets (js, css, svg, binary, missing, traversal,
folder, HEAD, POST), foreign/localhost Host, no session, bad/latin-1 bearer, bearer and cookie on
unknown paths, trailing slash, 405, OPTIONS (known/unknown), HEAD unknown, `GET /session`, session
POST (ok, wrong, number, no content type, malformed, list → 500, `0`, non-ASCII → 500, BOM), dev
`X-Studio-Frontend` (ok with trailing slash, old tab → `active_frontend`), packaged Origin (same,
foreign, `null`, bad IPv6 → 400, IPv4 in brackets → 400), license-prefix and preferences-prefix
paths, events cursor errors (`abc`, 300-char header with `%.200R`, empty header beats `?after`,
`%e9`, raw byte → Waitress "Bad URI"), event stream start (with and without `+1_0` cursor), POST
events, merge-slashes redirects (GET, query escaping, POST → 405, OPTIONS, variable path).

## Known differences (documented; none reachable from the ScadaBench UI)

1. `Allow` header order: Werkzeug lists a Python set (order changes per process); Go sorts.
2. No `Server: waitress` header; Waitress' 16-thread request cap is not copied (Go serves every
   request at once). Malformed HTTP (bad request line/headers) gets Go's own 400 text; only the
   non-ASCII request target ("Bad URI") is reproduced.
3. Session cookies: Go reads zlib-compressed Flask cookies but writes uncompressed ones; it only
   ever writes `{"authenticated": true}`, which Flask does not compress either (byte-equal).
4. `ContentDisposition` with a non-ASCII name drops non-ASCII in the fallback `filename=`; Werkzeug
   NFKD-normalises first (é → e). The `filename*=` part is identical.
5. Python logs 500 tracebacks to stderr; the Go server does not log yet (logging is stream work).
6. An event payload key `id` would, in Python, also change the ring's id; Go keeps the sequence
   (payload keys are renamed before publishing in Python, so this never happens).
7. `apperr.KeyError` (shared contract) builds `'key'`; Python's `str(KeyError)` is `repr(key)`, which
   uses double quotes for keys containing `'`. Proposal: `apperr.KeyError` should use
   `pycompat.Repr` (integrator change; `pycompat` already exports it).
8. Not in Go yet (P5): `--network-helper`, `--self-test`, packaged single-instance
   `show_running_instance` (the Go server prints the lease text and exits 1), app window, license
   and update services, background recovery (the `StartupGate` seam is ready).
9. Routes not ported answer 501 `{"error": "The Go engine preview does not provide this action yet.
   Use the Python engine for it."}` (dev preview only, as ADR-0074 says).

## What other streams need (contract for using stream A)

- **Routes**: return `[]api.Route` with Python's paths in Go pattern syntax (`/api/v1/ieds/{key}/settings`,
  `{name...}` for the rest of the path, and `{edge_id}.csv` — a variable with a literal suffix in one
  segment, as Werkzeug's `<edge_id>.csv`). Read values with `r.PathValue`. HEAD/OPTIONS, 404, 405
  and merge-slash redirects are automatic. Pass areas to `app.Config.Areas`; their routes replace the
  501 placeholders automatically.
- **Errors**: return them through `httpapi.Handle(func(w, r) error)` or `httpapi.WriteError(w, err)`;
  a handler may also `panic(err)` — `*apperr.Error`, `*httpapi.HTTPError` (Werkzeug texts:
  `ErrBadRequest`, `ErrNotFound`, `ErrTooLarge`, `ErrUnsupportedJSON`, ...) map exactly; any other
  error (including `*pycompat.Error` OverflowError/OSError/TypeError) is the Flask 500 body.
- **Bodies**: `httpapi.ReadJSON(r, silent)` is `request.get_json(silent=...)` (nil = None);
  `httpapi.ReadBody(r)` is `get_data()` with the 100 MiB limit; `httpapi.Arg(r, name)` is
  `request.args.get` (returns the UnicodeDecodeError Flask answered with 400);
  `httpapi.Header(r, name)` is `request.headers.get` (latin-1).
- **Responses**: `httpapi.WriteJSON(w, status, value)` (= `jsonify`), `httpapi.WriteBody`,
  `httpapi.ContentDisposition("attachment", name)` for `send_file` downloads.
- **Events**: publish with `bus.Publish(kind, payload *pyjson.Object)` (payload keys in Python's
  call order, nil allowed); the returned `*events.Event` has `Data` = `json.dumps(event)` bytes for
  the run log (stream C) — never mutate `Payload` after publishing.
- **Stream F**: implement `httpapi.LicenseGate` (`Enforced()`, `Licensed()` with the 60 s cache) and
  give it to `app.Config.License`. **Stream B**: implement `httpapi.StartupGate` (background
  recovery `Wait(timeout) bool`, `Err() error`) for `app.Config.Startup`.
- **Number/text rules**: use `pycompat` instead of `strconv`/`strings` where Python semantics
  matter (`int()` of user text, `round`, `.strip()`, `.casefold()`, ISO timestamps, `utf-8-sig`).
  CPython `random` (noise generators) is not here; stream C owns it.

## Proposed ADR text (for the integrator to number)

**Go engine HTTP platform: Flask-identical pipeline, session cookie and event stream.**
Problem: the Go engine must answer every request, error and event exactly as Flask 3.1, Werkzeug
3.1 and Waitress 3.0 did, and a tab must stay signed in when the engine is switched. Decision:
`internal/httpapi` reproduces the request pipeline order of `app_factory.py` (Waitress URI check,
loopback/origin 403, session 401, license 402, startup wait, Werkzeug routing incl. merge-slash
redirects, automatic HEAD/OPTIONS), Flask's signed session cookie (itsdangerous HMAC-SHA1, same
secret per mode) and Flask/Werkzeug error bodies; `internal/events` keeps a 5000-event ring with
each event encoded once; the launcher shares `studio.lock` (byte 0) and `launcher.json` with Python;
`run_dev_engine.py` selects the engine in dev (Python default). Evidence: differential tests
against CPython and the frozen Python app (above). Compatibility: none for users while Python is
the default; listed known differences are unreachable from the UI. Dependencies: Go standard
library only (`crypto/sha3` needs Go 1.24).

## Open questions for the integrator / owner

- `verify.py` now also runs the Flask parity test (it sets `STUDIO_PYTHON`): about 10 s and five
  short-lived Python servers on ports ≥ 20000, plus two `go build`s for the server test (~25 s).
  Use `go test -short` there if that is too slow (the build tests skip under `-short`).
- `start-studio.cmd` still calls `run_dev.py`; switching it to `run_dev_engine.py` (same default)
  is a shared-file change left to the integrator.
- `studio/build/engine/studio_server.exe` lands in the already ignored `studio/build/`.
