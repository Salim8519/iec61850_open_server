# HTTP conventions the Go server must copy (cross-cutting)

Source: `studio/backend/studio_app/app_factory.py`, `api/security.py`, `api/startup_gate.py`,
`api/events.py`, `runtime/event_bus.py`, `launcher/entrypoint.py`, `launcher/frontend_assets.py`.
Python stack: Flask 3.1.2, Werkzeug 3.x, Waitress 3.0.2 (`threads=16`), pydantic 2.10.6.
App is never run with `app.debug=True` (dev uses the Werkzeug reloader only), so the
non-debug behaviour below applies in BOTH dev and packaged.

## 1. Request pipeline order (per request)

| Step | Python | Effect |
| --- | --- | --- |
| 1 | `security.protect_request` (app_factory.py:65) | loopback/origin/session/license checks (below) |
| 2 | `startup_gate.wait_for_project` (app_factory.py:66) | waits for background startup recovery |
| 3 | URL routing (Werkzeug) | 404/405 are raised only AFTER steps 1-2, so an unauthenticated unknown `/api/...` path answers 401, not 404 |
| 4 | handler | blueprints mounted at `/api/v1` (app_factory.py:67-70) |
| 5 | error handlers | see section 4 |

### protect_request (security.py:27-47), exact order
1. `request.host` without port must be `127.0.0.1` or `localhost`, AND `remote_addr` must be loopback
   (`ipaddress.ip_address(..).is_loopback`; unparsable -> refuse; `None` -> allow).
   Else **403** `{"error":"Administration is available on loopback only"}`. Applies to `/` and `/assets/*` too.
2. `origin = Origin header`; `dev_origin = ADMIN_ORIGIN` (dev only, `--frontend-url` rstrip "/").
   `frontend = X-Studio-Frontend header or origin`.
   - dev: refuse when `frontend` present and `frontend.rstrip("/") != dev_origin`.
   - packaged: refuse when `Origin` present and `urlsplit(origin).netloc != request.host`.
   Refusal: **403** `{"active_frontend": <dev_origin or null>, "error": msg}` with msg dev =
   "This is not the active ScadaBench browser tab. Reopen ScadaBench from start-studio.cmd.",
   packaged = "Cross-origin administration is not permitted". Frontend `api/client.ts` redirects
   the tab to `active_frontend` when it is a 127.0.0.1/localhost URL.
3. Paths not starting with `/api/` and exactly `/api/v1/session` stop here (allowed).
4. Auth: cookie session `authenticated` OR `Authorization: Bearer <LAUNCH_TOKEN>` (constant-time
   compare; a missing header compares "" with the token). Else **401**
   `{"error":"This browser tab has no ScadaBench session. Open the full address printed by <L> (it ends with #session=...)."}`
   with `<L>` = `start-studio.cmd` (dev) or `ScadaBench.exe` (packaged).
5. License gate (`_require_license`, security.py:50-58): only when the license service `enforce`
   is true (packaged = `sys.frozen`), and path does not start with `/api/v1/license`;
   unlicensed -> **402** `{"code":"license_required","error":"A license is required. Start the free trial or activate your key."}`.
   `is_licensed()` uses a 60 s cached evaluation (licensing/service.py:16,132-137).

### startup_gate (startup_gate.py)
- Skipped for non-`/api/` paths and paths starting with `/api/v1/session`, `/api/v1/license`,
  `/api/v1/preferences` (prefix match, so `/api/v1/licenseX` also skips).
- Waits up to **300 s** for `BackgroundRecovery`; timeout -> **503**
  `{"code":"project_opening","error":"ScadaBench is still opening your last project. Try again in a moment."}`.
- Recovery failed -> **503** `{"code":"startup_recovery_failed","error":"ScadaBench stopped to protect your last autosave: <error>. Close ScadaBench, inspect the data folder, then start it again."}`.

### Session cookie
`SESSION_COOKIE_NAME=studio_session`, HttpOnly, SameSite=Strict, Flask signed cookie
(itsdangerous) with `SECRET_KEY` = random 32-byte hex per launch (packaged) or the fixed
`"studio-dev-secret"` (dev, entrypoint.py:156) so a dev reload keeps the tab signed in.
No `Secure` flag, no explicit expiry (browser-session cookie). `POST /api/v1/session` sets it.
Launch token: packaged `secrets.token_urlsafe(32)` per launch; dev `--dev-token` /
env `STUDIO_DEV_TOKEN`, default `"dev"`. The Go cookie format may differ (opaque to the browser)
but a tab signed in to a dev backend must survive a backend restart (dev reload parity).

## 2. JSON response encoding (`flask.jsonify`, DefaultJSONProvider)
- `sort_keys=True`: every jsonify object has keys sorted **alphabetically** (recursively).
- `ensure_ascii=True`: non-ASCII escaped as `\uXXXX` (surrogate pairs above BMP).
- Compact: separators `(",", ":")`; body ends with a single `"\n"`.
- `Content-Type: application/json`.
- Python floats: repr form (`1.0`, `0.1`, `1e-05`, `1e+16`); ints stay ints; `bool` -> true/false.
  An int-valued float stays `x.0` (e.g. pydantic-normalised positions `100.0`). Go must not
  turn `1.0` into `1` or `1` into `1.0`.
- NaN/Infinity: `allow_nan` default True -> literal `NaN`/`Infinity` (invalid JSON; frontend
  `response.json()` would throw). Mostly prevented by validation; keep same.
- Tuples -> arrays; dataclass -> object (LicenseStatus via `as_dict()` first).
- `jsonify(list)` returns a top-level array (e.g. `GET /scenarios/runs`).

## 3. SSE stream (`GET /api/v1/events`, api/events.py, event_bus.py)
- Cursor = `int(Last-Event-ID header, else ?after=, else 0)`, then `max(0, ...)`.
  A non-integer -> ValueError -> **400** `{"error":"invalid literal for int() with base 10: '...'"}`.
- Response `text/event-stream; charset=utf-8`, header `Cache-Control: no-cache`, streamed.
- First chunk: `retry: 1000\n\n`.
- Loop: `bus.after(cursor, wait=10)`; empty -> `: heartbeat\n\n` (every ~10 s when idle);
  each event -> `id: <n>\ndata: <json>\n\n`.
- Event JSON uses plain `json.dumps(event)`: separators `", "` and `": "`, **insertion order**
  (`id`, `kind`, `timestamp`, then payload keys in call order), ensure_ascii, NOT sorted.
  `timestamp` = `time.time()` float seconds.
- Bus: capacity **5000** (deque), sequence starts at 1 per process. Resync rule: if
  `cursor` (non-zero) `< items[0].id - 1` -> a single synthetic
  `{"id": <current sequence>, "kind": "resync", "timestamp": now}` (not stored).
  Cursor 0 => replay ALL buffered events (up to 5000) on connect.
- A restarted backend restarts ids at 1; a client with a higher Last-Event-ID simply waits
  until ids pass it (no resync). Keep identical.

## 4. Error mapping (app_factory.py:76-97)
| Exception | Status | Body (sorted keys) |
| --- | --- | --- |
| `StudioError` (subclass of ValueError) | 409 if `code=="conflict"` else 400 | `{"code":..,"details":..,"error":msg}`; `details` default `[]` (a falsy `{}` also becomes `[]`) |
| pydantic `ValidationError` | 400 | `{"details":[{"loc":[..],"msg":..,"type":..,"url":..}],"error":"Invalid request"}` (include_context/input False; `url` kept) |
| `ValueError`, `KeyError` | 400 | `{"error": str(e)}`; KeyError text includes quotes: `"'ied_id'"` |
| `RuntimeError`, `TimeoutError` | 503 | `{"error": str(e)}` (native worker errors, "Native <op> request timed out") |
| Werkzeug `HTTPException` | its code | `{"error": description}` (no `code` key) |
| anything else (AttributeError, TypeError...) | 500 | `{"error":"The server encountered an internal error and was unable to complete your request. Either the server is overloaded or there is an error in the application."}` (InternalServerError routed to the HTTPException handler) |

Handler resolution is by exception MRO: e.g. `BadRequestKeyError` (missing multipart `file`)
is an HTTPException first -> 400 `"The browser (or proxy) sent a request that this server could not understand."`.
`LicenseServerError` is a RuntimeError (503 outside license routes). `ConnectionError` in
updates is caught internally. Some handlers return their own `(jsonify, status)` (listed per route).

Werkzeug texts that reach the UI (frontend shows `data.error`):
- 404: "The requested URL was not found on the server. If you entered the URL manually please check your spelling and try again." (frontend replaces any 404 by its own "running backend does not provide this action" text, client.ts).
- 405: "The method is not allowed for the requested URL." (+ `Allow` header).
- 413 (body > `MAX_CONTENT_LENGTH` 100 MiB): "The data value transmitted exceeds the capacity limit."
- 415 (`request.get_json()` non-silent without JSON content type): "Did not attempt to load JSON data because the request Content-Type was not 'application/json'."
- 400 on malformed JSON (non-silent `get_json`, non-debug): generic BadRequest description above.
- `get_json(silent=True)` returns None on any problem -> handlers use `or {}`.
- A JSON body of `null`/list where the handler calls `.get` -> AttributeError -> 500 text above.
- Routes have strict trailing slash: `/api/v1/project/` -> 404 (no redirect: rules have no slash).
- Flask adds automatic HEAD (same as GET, no body) and OPTIONS (200, `Allow` header) for each rule.

## 5. Bool/int quirks the handlers rely on (Python semantics)
- `isinstance(True, int)` is True: checks written `isinstance(x, int)` without a bool guard accept
  `true` as 1 (e.g. explorer `port`, scan `port`, gateway command `address` IS guarded). Each
  route file lists where a guard exists. Go must reproduce per check, not "fix" it.
- `data.get("confirm") is not True` -> only JSON `true` confirms (not 1, not "true").
- `int(request.args.get(...))` accepts `" 7 "`, `"+7"`, `"0_7"`; rejects `"7.0"`.
- `str.casefold()` used for signal search (Unicode full case folding, e.g. `ß` -> `ss`).

## 6. Non-API routes
| Route | Behaviour |
| --- | --- |
| `GET /` | frontend unavailable -> **503** `text/html; charset=utf-8` body "ScadaBench frontend build is not available yet." (Flask str response); else index.html |
| `GET /assets/<path>` | from `frontend.pak` (memory) or `frontend/dist/assets` |
- `.pak` mode (protected build, frontend_assets.py:345-374): missing file -> 404 text "Not found";
  `index.html` `Cache-Control: no-cache`; others `public, max-age=31536000, immutable`;
  `.js` -> `text/javascript`, else `mimetypes.guess_type` or `application/octet-stream`.
- Folder mode (dev and unprotected builds): `send_from_directory` (ETag, Last-Modified,
  conditional 304, Werkzeug 404 JSON via handler, path traversal -> 404).
- `.pak` format: `b"SSPAK1"` + 16-byte nonce + XOR(SHAKE-256(key|nonce)) of
  `<u32 LE header len><compact JSON {name:[offset,len]}><zlib blobs>` (key in frontend_assets.py:301).
  The Go server must read the same pack produced by `studio/tools/pack_frontend.py`.

## 7. Downloads (send_file / Response)
| Route | Content-Type | Content-Disposition |
| --- | --- | --- |
| `/project/download` | application/zip | `attachment; filename=substation.subsim` |
| `/ieds/<id>/source` | application/xml | `attachment; filename=<ied.filename>` (Werkzeug quotes/RFC 5987-encodes non-ASCII as `filename*=UTF-8''...`) |
| `/ieds/<id>/export-scl` | application/xml | `attachment; filename=<NAME>.edited.scd` |
| `/live/<id>/comparison.csv` | text/csv; charset=utf-8 | `attachment; filename=<ied name>-scd-vs-device.csv` (raw, unquoted) |
| `/signal-flow/edges/<id>.csv` | text/csv; charset=utf-8 | `attachment; filename="<pub>-to-<sub>-<type>.csv"` (quoted) |
| `/health.csv` | text/csv; charset=utf-8 | `attachment; filename=station-health.csv` |
| `/health/report.html` | text/html; charset=utf-8 | `attachment; filename=station-health.html` |
| `/tools/explorer/watch.csv` | text/csv; charset=utf-8 | `attachment; filename=watch-<host with . -> ->.csv` |
| `/tools/gateway/points.csv` | text/csv; charset=utf-8 | `attachment; filename=gateway-<protocol>-points.csv` |
| `/tools/master/points.csv` | text/csv; charset=utf-8 | `attachment; filename=master-<protocol>-points.csv` |
CSV writers: Python `csv.writer` default dialect = `\r\n` line endings and minimal quoting,
EXCEPT signal-flow edge CSV (`lineterminator="\n"`). `send_file` of an in-memory body adds
`Content-Length`, `Cache-Control: no-cache`, `Accept-Ranges: bytes` (Range requests honoured),
no ETag/Last-Modified. Keep byte-identical CSV/HTML/ZIP bodies; headers as listed. Frontend
downloads go through `api/download.ts` (fetch + blob) or plain links; filename comes from the header.

## Parity risks
- Alphabetical key order in jsonify vs insertion order in SSE/files: two different encoders.
- Python float repr and `\uXXXX` escaping; Go `encoding/json` escapes `<>&` and uses
  shortest-float without `.0` - needs a custom encoder.
- Error text of Werkzeug/pydantic/Python exceptions (KeyError quotes, int() messages) is shown verbatim.
- Middleware order (401/402/503 before 404) and the exact prefix list of the startup gate.
- Non-debug Flask: malformed JSON gives the generic 400 text, not the parser message.
