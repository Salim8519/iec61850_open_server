# ADR-0012: One development launcher

Status: accepted, 2026-09-16.

Two root commands previously represented different applications:
`start-studio.cmd` launched an old packaged snapshot while `start-dev.cmd`
launched current backend source plus Vite. The identical naming made it easy to
run stale code and interpret the missing update as unreliable hot reload.

The repository now exposes one human entry point: `start-studio.cmd`. It always
runs source development mode. A small Python supervisor starts Vite and the Flask
source launcher in one console, uses the normal Release native worker, and owns
both exact process trees. If either layer exits, the other is stopped. Ctrl+C
stops only processes created by that invocation; title-based process killing and
a second terminal window are removed. Vite is started as its Node process directly,
not through a detachable npm/cmd wrapper, so its PID remains owned through cleanup.

Before starting either layer, the supervisor checks the fixed backend port and
chooses the first free Vite port from 5173 through 5199. A stale frontend therefore
cannot block launch, and the browser opens the newly owned instance. A true startup
error remains visible in the command window. The supervisor never kills a process
it did not create.

If port 8765 is occupied, the launcher probes the allowed Vite range through its
session endpoint using the configured development token and exact Origin. A
verified Studio development session is reopened and the second launcher exits
successfully. An unrelated or unhealthy listener is never killed or silently
adopted; it remains an actionable startup error.

Dev mode gives the backend the one exact Vite origin chosen by the supervisor.
The administration guard accepts that origin in addition to its own host and still
rejects every other cross-origin request. Packaged mode receives no exception.
If an old Vite tab contacts the backend from another development port, the backend
identifies the active frontend and the browser redirects to it instead of leaving
the user to choose between apparently different engines.
Browser launch waits until the backend socket is accepting connections so the
initial session request cannot race Waitress startup through the Vite proxy.

Frontend TypeScript/CSS changes use Vite HMR. Loaded Python module changes restart
the backend through Werkzeug and the reconnecting frontend refreshes authoritative
state. A running simulation stops on a backend reload. C++ changes still require
a native build; replacing the watched Release worker then restarts the backend.
The frozen preview remains directly launchable as
`studio/dist/SubstationStudio/SubstationStudio.exe`, but it is not given a second
root command that can be confused with live source development.

Development reload is a delivery mechanism, not a different product mode. Backend
services, defaults, project data and native semantics stay identical to a future
package; only asset serving, reload and development diagnostics differ. Packaging
is performed only when explicitly requested.
