# ADR-0048: ScadaBench behaves like a normal desktop app

Accepted: 2026-10-17. Owner report: users of the packaged app see a black console
window ("horror for them"). Starting the app a second time made it close with no
explanation, because the data folder was already in use. Users do not know about an
engine running behind the window. Owner rule: one window, starting it again shows that
window, closing it asks first and stops everything, and the drawing tab is called SLD
and comes after Devices.

**Decision**
- **Windowed EXE.** Both PyInstaller specs build with `console=False`. Without a console:
  - `launcher/windowed.py` shows plain Windows message boxes for anything the user must read;
  - diagnostics go to `data/logs/launcher-YYYYMMDD.log` (append-only);
  - a launch from a terminal (e.g. `ScadaBench.exe --self-test`) attaches to that
    terminal and prints as before.
- **One instance per data folder.** The existing data-folder lease still decides who runs.
  - A later launch that cannot get the lease asks the running instance to show itself
    through a per-session named event (`Local\ScadaBench-Show-<folder hash>`,
    `launcher/single_instance.py`). The event carries no token or data.
  - The launch exits at once. It first gives the running instance the right to take the
    foreground (`AllowSetForegroundWindow`).
  - Heavy modules load only after the lease is taken, so a second launch is quick.
  - If no instance answers (e.g. a development backend holds the folder), the
    lease message is shown in a dialog.
- **Show again.** The running instance records its window's browser process id in
  `launcher.json`. The new launch is the process Windows allows to take the foreground, so
  it restores and focuses that window itself. When it can't, it falls back to the named
  event, and `AppWindow.show` in the running instance restores and focuses the window owned by its browser
  process (`launcher/window_focus.py`). Without Edge or Chrome it reopens the
  default-browser tab. If the window was just closed, it does nothing, because the
  app is already stopping.
- **Closing stops everything, after a prompt.**
  - The launcher marks its own window with `window=app` in the session fragment.
    `useExitConfirm` then registers the browser's standard leave prompt (Leave / Cancel).
  - Confirming closes the browser process. The watcher stops Waitress and the runtime,
    which stops every worker.
  - Browser tabs (development, or no Edge/Chrome) stop nothing when closed, so they
    never ask.
  - The project download became a download link, so it no longer triggers the prompt.
- **Hand-off detection without timing.** A Chromium browser keeps `<profile>/lockfile`
  open. If the browser process exits while the lock is still held, another browser has
  the window, so the app keeps running. Otherwise the user closed the window.
  - The old "exited within 10 s" rule would have kept a windowless app running after a
    quick close.
  - A leftover window from an earlier run never takes over the new profile:
    `choose_profile` skips locked profiles.
- **SLD tab.** The sidebar order is Devices, SLD, Test, Settings, and the app opens on
  Devices. The drawing place's label is "SLD" and its page heading is "Single-line
  diagram (SLD)". The internal place id stays `diagram`, so no stored data changes.

**Compatibility**
- No project, API or protocol change.
- The dev launcher still prints its addresses and never opens a window (ADR-0012).
  Only the packaged window prompts on close, because only it stops the app.
- `test_portable` now expects a duplicate launch to succeed and report "already running".

**Limitations**
- The prompt text is the browser's own ("Leave site?"). It only appears after the user
  has interacted with the page (a browser rule).
- Without Edge or Chrome the app opens in the default browser. It then has no exit
  control besides signing out or Task Manager.
- Behaviour of the windowed EXE is unverified until the owner asks for a build.
