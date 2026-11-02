# ADR-0135: Undo/Redo for the whole project in the engine, File › New/Save as/Close, and clearer controls

Accepted: 2026-11-02. Owner: "why there is no close current project? … new project button?", "small
buttons for redo and undo … check UI of each page", "make all disconnect buttons red", "why there
is refresh button on each path of the IED explorer? … is auto refresh?", then chose: File menu with
New project, Save project, Save as, Close project; "top bar … one redo and undo manager that is
fast, caches last 20 steps only, lightweight CPU and RAM, general in any page"; solid red
Disconnect; Explorer auto refresh every 2 s plus one button; "Operate · real MMS command" and
"Test · forced change, no command" titles (breaker, DS, ES); Undo inside the Go engine rather than a
separate program. Change [0210](../changes/2026/0210-project-undo-file-menu.md).

## Problem

- Undo existed only for the single-line diagram, in the browser (50 steps, `historySession.ts`).
  Devices, Test, Tools and the gateway lists had none or a one-time Undo.
- The app keeps one project; the only way to switch was Open file. There was no New, Close or
  Save as (noted as wish A16 in change 0177).
- IED Explorer read an opened data object once; a refresh arrow on each opened object read it
  again. Disconnect buttons looked like any other button. The breaker sheet's "Operate" and
  "Test" did not say which one sends a real MMS command.

## Decision

1. **Undo history in the engine.** New pure package `engine/internal/project/history` keeps the
   last 20 steps in memory only. A step is the top-level project keys a commit changed, with their
   values before and after. Project trees are copy-on-write, so a step references unchanged parts
   and costs only what changed. The project service records in `commit` (the one door every edit
   passes), so every page is covered. Quick follow-ups of the same keys from the same source within
   1 s join one step (typing, dragging).
2. **What is not a step.** The `id` and `network` keys never are: this PC's card, lab mode and GOOSE
   reach drive real Windows addresses. An IED's `lab_ip` and the gateway's `auto_start` are
   ignored when comparing and carried over when a step is put back. Commits made while IEDs run
   are not recorded (a runtime `OnState` hook sets `Service.SetRunning`), and Undo/Redo are
   refused until they stop. A step whose result was later changed by something that is not a step
   can never apply again: that stack is dropped, so Undo never overwrites a later change.
3. **Undo is a normal change.** Undo/Redo commit through `commit` (journal, autosave,
   `project_changed`). Opening a file, startup recovery, New and Close clear the history.
4. **Routes (Go-only).** `GET /project/history` (`can_undo`, `can_redo`, plain-words labels such as
   "IED change", `running`, `limit`), `POST /project/undo`, `/project/redo`, `POST /project/new
   {name}` (trimmed, at most 80 characters, no control characters) and `POST /project/close`. New
   and Close refuse while IEDs run and keep this PC's network choice.
5. **Browser.** Two small icons next to File (`TopBar` `UndoRedo`), with tips from the engine's
   labels; Ctrl+Z, Ctrl+Y and Ctrl+Shift+Z on every page outside text fields and dialogs
   (`useUndoKeys`). A pending diagram edit is saved before Undo. The SLD toolbar's own arrows and
   the browser history were removed; `diagramHistory.ts` stays as the test model that each canvas
   gesture is one edit.
6. **File menu.** Open file…, New project…, Save project, Save as…, Close project…. Save as uses
   the Windows Save dialog (`showSaveFilePicker`) with a Windows-safe name from the project name,
   else a named download; it replaces Download project. New and Close share one dialog that offers
   "Save a copy…" first.
7. **Startup recovery notice** shows only while the recovered project is the open one
   (`Service.StillRecovered`), so it disappears after New, Close or Open.
8. **Explorer.** Open data objects are read when opened and again every 2 s while shown and the
   window is visible (`useNodeReads`, never a command field, 200 attributes per request). One
   "Read all now" button replaces the arrow on each object. A failed automatic read pauses the
   refresh and says so once.
9. **Look.** The four Disconnect buttons (Devices real IED, IED Explorer, gateway/converter device,
   SNMP) and every Stop button (top bar Stop / Stop all, Stop this IED, gateway, Find IEDs scan,
   register search, SNMP trap listening, Stop watching) use `button.danger` with their icon and
   word; the run state beside the top-bar Stop keeps ○ ◐ ● and its green mark. Section titles read "Operate · real MMS
   command" (also the tap changer) and "Test · forced change, no command".

## Rejected

- A separate undo program: a second copy of the project, IPC on every edit, another executable to
  start, match and scan, and the risk of the two copies disagreeing.
- Whole-project snapshots: undoing an old step would also revert automatic changes made since.
- Recording changes made while IEDs run: operations and runtime addresses are not engineering edits.

## Compatibility

Existing routes and their bodies are unchanged except GET /project's `startup_recovery`, which is
null once another project is open. The Python engine has no history route; the browser hides the
Undo/Redo buttons when the route is missing. Project files are unchanged.
