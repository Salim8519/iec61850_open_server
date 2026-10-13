# 0019: Development hot reload

Date: 2026-09-15. Implementation follows the user's request and
[ADR-0010](../../decisions/ADR-0010-development-reload.md).

Added `start-dev.cmd` at the repository root and a `--dev` launcher mode. A
Python edit restarts the backend through Werkzeug's reloader; a Vite dev server
applies frontend edits through HMR and proxies `/api` to 127.0.0.1:8765. The
browser opens `http://127.0.0.1:5173/#session=<token>`; no manual refresh is
required for either layer.

Reload safety: `--dev` holds the directory lease only in the serving child, so
each reload reacquires it cleanly. The launch token and secret key are fixed in
dev so the session cookie survives restarts. A busy port fails loudly instead of
falling back to a random port. `atexit` closes the coordinator, stopping native
workers and flushing buffered run logs. Extra reload watch targets the worker
executable, so a rebuild restarts the backend.

Reconnect resync: `/api/v1/events` now sends `retry: 1000`, and the frontend
re-fetches the project snapshot on every stream `onopen`. This removes the stale
state that previously followed a backend restart and also covers browser sleep
and network interruptions in the packaged app.

`--native-exe` selects an explicit worker; `start-dev.cmd` points it at the
already-built `studio/native/build/feature-preview/Release/studio_ied.exe` so
Run works without rebuilding. `--dev`, `--frontend-url` and `--native-exe`
defaults leave packaged and test behavior unchanged.

Evidence: 78 unit/integration tests passed; 15 protocol tests passed with
`STUDIO_NATIVE_BIN` set to the feature-preview build; `tsc --noEmit` passed;
`python studio/tools/verify.py` passed. Live smoke test: `--dev` started,
POST `/api/v1/session` returned 200, touching `api/events.py` produced a
reloader restart with a second startup line and another 200 session response,
and killing the process tree released port 8899 and left no native workers from
that test. Vite smoke test: `npm run dev` served `http://127.0.0.1:5173/` with
HTTP 200, and `/api/v1/project` through the proxy returned the backend's 401
JSON, proving proxy routing and the same-origin check; port 5173 was released
after stopping the tree.

Limitations: a running simulation stops on backend reload; native C++ edits
still require a CMake build; the dev secret key and token apply only to
`--dev`; interactive visual acceptance of the Vite workflow is pending. A live
packaged session (feature-preview executable, `studio/data`, port 8765) must be
stopped before dev mode can take the lease and port.

Superseded on 2026-09-16 by change 0021: `start-dev.cmd` was removed and its
source workflow became the sole `start-studio.cmd` behavior.
