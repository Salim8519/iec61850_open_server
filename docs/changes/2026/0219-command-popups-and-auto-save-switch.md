# 0219 Command screens open as pop-ups; auto save can be turned off

Date: 2026-11-11. Owner: "make the command screen as pop up instead of covering the data
points", "Same for gateway tools", "add toggle button to turn off project auto save, default is
on. Remember user choice".
Decision: [ADR-0139](../../decisions/ADR-0139-command-popups-and-auto-save-switch.md).
Spec: [Saving the project](../../specs/project-save.md) SAVE-01..04, UI-10.
Layers: frontend, Go engine (`project/service`, `project`, `api/projectapi`, `app`).

## Problem

The Devices command strip (0218), the gateway's REAL-device confirmation and the SCADA master's
command form sat in boxes above their tables and pushed the data points out of view. Every
project change was written at once with no way to turn that off.

## Change

- **Pop-ups.** New shared `ui/PopupDialog.tsx` (+ `popup-dialog.css`, shared tokens).
  - Devices › Data points: one line "Commands · state sentence · Send commands…"; the pop-up
    lists the commands, shows the REAL-device warning and keeps the device's answer.
  - Gateway point list: the confirmation is a pop-up titled with the command and target and
    shows the answer after Yes (it is no longer in the message area).
  - SCADA master / converter: "New command…" and a row's "Command…" open the form,
    confirmation and Repeat automatically in a pop-up.
- **Auto save switch.** File › "Turn off auto save" / "Turn on auto save" (on by default).
  - Off: the engine keeps changes in the open project and writes neither journal nor snapshot
    until File › Save project or Ctrl+S; closing without saving gives back the last saved
    project at the next start.
  - On again: unsaved changes are written first, under the mutation lock.
  - Remembered as Go-only preference `autosave_off`; applied at engine start.
  - Routes `GET`/`PUT /project/autosave` (Go only).
  - Top bar: "Auto save off" or "Not saved · Ctrl+S" with a tip.

## Evidence

- Go: `project/service` TestAutoSaveOffWritesOnlyOnSave, TestSaveAndTurningAutoSaveOnWrite;
  `project` TestAutoSaveOffPreference. Full `go test ./...` passes except the two runtime tests
  whose fixed ports 23201/23901 Windows refuses on this PC (WinError 10013, as in 0218), so they
  are not caused by this change. Parity replay (`app`) passes.
- `TestEveryPythonRouteServed` had failed since 0218 because the Go-only routes
  `GET /live/{key}/controls` and `POST /live/{key}/operate` were not on the live area's Go-only
  list; they are now, with `GET`/`PUT /project/autosave`, and the test passes.
- Frontend: 487 tests pass (new: top bar words, pop-up structure); `tsc --noEmit` clean.
- Two app copies from this source (headless Chrome, 1366x768):
  - Devices pop-up: Close operated (sbo-with-enhanced-security), read back closed; Open read
    back open. The table stayed in place behind it.
  - Gateway pop-up: "Close" sent through the gateway to the real IED, the IED operated it.
  - SCADA master pop-up connected to the other copy's 104 gateway: form, confirmation and the
    device's answer ("refused by the device (unknown object address)" for a status address).
  - Auto save: menu item; "Not saved · Ctrl+S" after adding a sample IED; Ctrl+S → "Auto save
    off" (engine `unsaved: false`); another unsaved IED dropped after an engine restart while the
    saved one stayed and auto save stayed off; turned on, a new IED survived the next restart.

## Limitations

- Changes made while auto save is off are lost if the PC or app crashes before Save; the tip
  says so.
- The packaged EXE gets the change with the next release build.

## Release 0.1.34 (with change 0218)

- `compile_go_release.py --version 0.1.34` with `studio\.venv`: antivirus gate 19 programs, 0 findings.
- Packaged portable on a throwaway data folder: `GET /app/update` current 0.1.34; sample IED
  imported, Start, an independent MMS read of `Feeder01CTRL/XCBR1.Pos.stVal` with `studio_probe`
  (ok, 01), Stop; auto save off → a new IED is unsaved → on again saves it.
- `website/deploy.cmd` under `fnm exec --using=24`: Pages 7abd1e91, "Saved release 0.1.34 from
  2026-11-11"; installed apps are told about 0.1.34 by the signed release record.
- Live installer 004c7245 (19,613,622 bytes) and portable 83d84bf1 (53,312,000 bytes) match the
  build; copies of both downloads scan clean (2 programs, 0 findings). The changelog page shows
  "0.1.34 · Real devices stay read only until you allow commands".
