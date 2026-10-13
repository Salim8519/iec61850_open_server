# 0021: One development launcher

Date: 2026-09-16. Implements
[ADR-0012](../../decisions/ADR-0012-one-development-launcher.md).

Removed `start-dev.cmd`. `start-studio.cmd` is now the only root Studio launcher
and always starts the current source backend plus Vite hot reload. It no longer
searches for whichever packaged executable has the newest timestamp.

The launcher uses one console and one owner process. It validates Python, Node and
the Release native worker before starting, uses exact child PIDs for cleanup, and
stops both frontend and backend when either exits. This removes the extra command
window, feature-preview worker preference and wildcard window-title termination.
Backend-port preflight runs before either layer starts. If 5173 is occupied, the
launcher selects the next free frontend port and opens that exact URL instead of
failing or terminating an unowned process. Real startup errors pause visibly.
Vite starts directly under Node so an intermediate batch process cannot detach it
from cleanup.
The backend origin guard accepts only the exact selected Vite URL in dev mode;
fallback ports work without weakening packaged same-origin enforcement.
The browser is opened only after Waitress accepts connections, preventing the
first proxied session request from racing backend startup.

Evidence: with an older Vite still occupying 5173, the launcher selected 5174,
served the frontend with HTTP 200 and accepted an authenticated project request
through its proxy with HTTP 200. A different origin remained forbidden. Editing
the backend triggered its reloader. Ctrl+C removed the owned 5174 and 8765
listeners; the pre-existing 5173 process was untouched. The complete maintained
suite passed 103 tests and the architecture/documentation checks passed.

Native C++ source still needs a build; frontend and Python changes reload
automatically. Browser visual acceptance remains a release gate.
