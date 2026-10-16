# 0073: ScadaBench behaves like a normal desktop app

Date: 2026-10-17. Decision: [ADR-0048](../../decisions/ADR-0048-desktop-app-window.md).
Spec: [app window](../../specs/app-window.md).

## Owner report

- Users of the packaged app saw a black console window with addresses and tokens.
- Starting ScadaBench again while it ran made the new launch close silently.
- Users don't know about the engine behind the window. Closing the app should ask
  and then stop everything, like any other app.
- The "Diagram" tab should be called SLD and come second, after Devices.

## Change

- `packaging/studio.spec`, `studio_protected.spec`: `console=False`.
- `launcher/windowed.py`: message boxes and a dated log file when there is no
  terminal; attaches to the parent terminal for `--self-test`.
- `launcher/single_instance.py`: per-data-folder named event, so a second launch asks
  the running one to show itself.
- `launcher/window_focus.py`: restore and focus the window of one browser process.
- `launcher/app_window.py`:
  - `AppWindow` opens and re-shows the window;
  - hand-off is detected by the held Chromium profile lock instead of a 10 s timer;
  - locked leftover profiles are skipped.
- `launcher/entrypoint.py`:
  - heavy imports happen after the lease, so a second launch is fast;
  - the window pid is recorded in `launcher.json` and the second launch focuses it itself;
  - start-up failures are shown as a dialog.
- Frontend:
  - `app/useExitConfirm.ts` adds the leave prompt only in the launcher-marked
    window (`window=app`, kept in sessionStorage);
  - `TopBar` downloads through a link, not a navigation;
  - `places.ts` order is Devices, SLD, Test, Settings, with the heading "Single-line
    diagram (SLD)"; the app opens on Devices;
  - the tour step is renamed.
- Docs: design layout and interactions, UI-structure spec (UI-01 order, new UI-08), tour
  spec, packaging README, licensing spec LIC-17 test references.

## Evidence

- `python studio/tools/verify.py`: all checks PASS.
- `pytest studio/tests/unit studio/tests/integration`: all pass. New:
  `test_app_window.py` (11) and `test_packaging_windowed.py`.
- `node --test studio/tests/frontend/*.test.ts`: 101 pass. New: `exitConfirm.test.ts`,
  UI-08.
- Real source launcher, non-dev, on a temporary data folder:
  - a second launch printed "ScadaBench is already running; its window was brought to
    the front." and exited 0 in 356 ms;
  - the first instance kept serving.
- Real Edge `--app` window via `AppWindow`:
  - the profile was locked while open;
  - minimise, then `show()`, restored the window;
  - WM_CLOSE (like clicking X) ran the stop callback and released the lock.
  - It did not take the foreground, because the test process had no foreground right.
    A double-clicked launch has that right, and it is why the second launch focuses the
    window itself.

## Limitations

- No EXE built, as owner policy requires. Windowed output, the dialog texts, focus on
  double-click and the close prompt in the real packaged window still need a build.
- The prompt wording is the browser's ("Leave site?"). It needs a prior click in the page.
- The default-browser fallback (no Edge/Chrome) has no exit control.
