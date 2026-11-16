# ADR-0139: Command screens as pop-ups, auto save on/off

Accepted: 2026-11-11. Owner requests in this session:
- "make the command screen as pop up instead of covering the data points";
- "Same for gateway tools";
- "add toggle button to turn off project auto save, default is on. Remember user choice".

## Context

Devices › Data points of a real IED (ADR-0138) showed its command buttons, the REAL-device
warning and the answer in a box above the table. The gateway point list and the SCADA master
put their confirmation and command form into the message area above their tables. Both pushed
the points down and hid the rows the user was about to switch.

Every project change is written at once: journaled and fsynced, then snapshotted into
`autosave.scadabench` (store/autosave). There was no way to try changes and keep the last good
project.

## Decision

### Command pop-ups

- One shared modal, `ui/PopupDialog.tsx` (native `<dialog>`, the app's dialog look from shared
  tokens). Escape and X close it, except while a command is on its way.
- **Devices.** Data points shows one line above the table: "Commands", the state sentence and
  "Send commands…" (enabled only while Allow commands is on and the IED is Live). The pop-up
  lists the IED's on/off and open/close commands; a button shows the REAL-device warning in the
  same pop-up, and the answer (what the device said, where the position went) stays there.
- **Gateway point list.** A row's Close…/Open… (or Send…) opens the warning as a pop-up titled
  with the command and target; after Yes the pop-up shows the answer until Close.
- **SCADA master / converter.** "New command…" and a row's "Command…" open the command form,
  its REAL-device confirmation and Repeat automatically in a pop-up.
- Nothing changes in what is sent or refused: the engine checks every command as before.

### Auto save switch

- File menu: "Turn off auto save" / "Turn on auto save". On is the default.
- **Off** (`project/service/manual_save.go`): `commit` still validates and publishes every
  change, but writes neither the journal nor a snapshot; the service remembers the project has
  unsaved changes. File › Save project and Ctrl+S write it (`Save`). Closing ScadaBench without
  saving loses those changes; the next start opens the last saved project (startup recovery
  restores `autosave.scadabench`, as always).
- **Turning it on** saves the unsaved changes first, under the mutation lock, so the journal
  always extends the file on disk. When that write fails, auto save stays off.
- Remembered in `preferences.json` as the Go-only `autosave_off` (0/1), stored only once set, so
  the frozen Python engine's default preference document is unchanged. The engine applies it
  when it starts, before the first change.
- Go-only routes: `GET /project/autosave` → `{on, unsaved}`; `PUT /project/autosave {on}`.
- Top bar: with auto save off it says "Auto save off", or "Not saved · Ctrl+S" while changes
  wait, with a tip saying what happens when the app closes first. The packaged window already
  asks before closing.
- Undo/Redo is unchanged (in memory, ADR-0135).

## Compatibility

- Projects, files and every Python-parity answer are unchanged; the new preference key exists
  only once a user turns auto save off.
- Auto save on behaves byte-for-byte as before (same journal and snapshot calls).
- The pop-ups are frontend-only.

## Rejected

- A per-page Save button: the owner asked for one project-wide switch.
- Asking on every close: the app window already asks before closing; the top bar shows unsaved
  changes all the time.
