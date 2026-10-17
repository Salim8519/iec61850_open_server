# 0074: Installed app no longer crashes when Edge relaunches its window

Date: 2026-10-17. Follows [0073](0073-desktop-app-window.md).
Spec: [app window](../../specs/app-window.md) (WIN-04), [licensing](../../specs/licensing.md) (LIC-17).

## Owner report

The installed ScadaBench crashed on start with a `KeyboardInterrupt` traceback in
`write_launcher_record`. Afterwards `launcher.json` was empty (0 bytes).

## Cause

- `watch_window` calls `_thread.interrupt_main` as soon as the started `msedge.exe` exits.
- Edge can exit within milliseconds of start: it relaunches itself (first run on a new
  profile, a pending update) or hands the window to a broker process. The profile lock
  is not held yet at that moment, so this looked like "the user closed the window".
- The interrupt reached the main thread while it was still writing `launcher.json`,
  outside the `try` around `server.run()`, so it surfaced as a crash traceback.

## Change

- `launcher/app_window.py`: a launcher that exits within 15 s gets a 5 s grace period
  to see the profile lock. While another browser process holds the profile, ScadaBench
  keeps running. It stops when that process releases the profile, so closing a
  handed-off window now also stops the app (before, the app kept running forever).
- `launcher/entrypoint.py`: the window opens inside the same `try`/`finally` as the
  server, so an interrupt during start-up shuts down cleanly (workers, listener, server).
- Tests: the hand-off test now also checks the stop after release, and a new test
  covers a relaunch right after start. Spec rules WIN-04 and LIC-17 point to both.

## Evidence

- `studio/.venv` pytest `studio/tests/unit`: 275 passed; `verify.py` all PASS.

## Limitations

- No EXE was rebuilt (the owner did not ask). The fix reaches users only after the next
  requested build. The real Edge relaunch was not reproduced on demand. The fix was
  derived from the traceback and the empty `launcher.json`.
- Closing a window the user really closed within 15 s of start now stops the app up to
  5 s later.
