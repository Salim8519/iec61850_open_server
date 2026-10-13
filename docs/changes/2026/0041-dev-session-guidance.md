# 0041: Accurate missing-session guidance in development mode

Date: 2026-09-29. Opening the development frontend without the
`#session=` fragment (for example Vite's own `Local:` banner address, an
old tab, or a bookmark) returned "Open the app through
SubstationStudio.exe to establish a session", which is wrong for
`start-studio.cmd`.

Changes (authentication semantics unchanged in both modes):

- The 401 message names the launcher that is actually running
  (`start-studio.cmd` in development, `SubstationStudio.exe` packaged) and
  states that the full printed address ends with `#session=...`.
- `run_dev.py` starts Vite with `--logLevel warn`, so the only printed
  address is the launcher's tokened link.
- The stale-tab redirect to the active frontend keeps the URL fragment,
  so a `#session=` token is not dropped while switching ports.

Evidence: running dev backend reproduced the 401 without a token and
accepted `{"token": "dev"}` through the Vite proxy; integration
`test_api.py`: 6 pass with new message assertions for both modes;
`tsc --noEmit` clean; `verify.py` passes.

Limitations: no packaged EXE rebuilt (not requested).
