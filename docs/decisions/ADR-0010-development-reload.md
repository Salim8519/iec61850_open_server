# ADR-0010: Development reload mode

Status: accepted by user, 2026-09-15.

Development edits previously required rebuilding the frozen portable package.
The user asked for code changes to apply without a manual browser refresh.

Development runs the same Flask app factory and Waitress server from source,
wrapped in Werkzeug's reloader (already bundled with Flask, no new dependency).
The reloader watches backend modules and the native worker executable; reload
exits the process, whose child workers already stop on stdin EOF. `--dev` uses
a fixed launch token and secret key so the browser session survives restarts,
and refuses a busy port instead of silently moving to another one. The browser
targets the Vite dev server, which applies frontend edits through HMR and
proxies `/api` to the backend. The SSE stream advertises a one-second retry and
the UI re-fetches its snapshot whenever the stream reconnects.

Packaged behavior is unchanged; `--dev` exists only in the source launcher.
A running simulation stops on reload because its native workers are child
processes. Native source changes still require a CMake build; the reloader then
restarts the backend when the rebuilt executable changes. The development
secret key and token are not used by the packaged application.
