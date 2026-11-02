# 0210: Undo/Redo for the whole project, File › New/Save as/Close, red Disconnect, Explorer auto refresh

Owner requests and choices of 2026-11-02 (see the decision). Decision:
[ADR-0135](../../decisions/ADR-0135-project-undo-file-menu.md).

## Change

| Area | Files |
| --- | --- |
| Undo history (Go, pure) | `engine/internal/project/history/history.go`, `equal.go`, `history_test.go` (new) |
| Project service | `project/service/undo.go`, `undo_rules.go`, `undo_test.go` (new); `service.go` (record in `commit`, fields), `restore.go` (clear on open, `StillRecovered`) |
| Run state hook | `runtime/coordinator.go` (`Options.OnState`), `run_index.go`, `app/wire_runtime.go` |
| Routes | `api/projectapi/history_routes.go`, `history_routes_test.go` (new), `api.go`, `data_classes_route.go` (Go-only list), `project_routes.go` and `app/project_view.go` (recovery notice) |
| Browser: Undo/Redo | `state/useHistory.ts`, `app/useUndoKeys.ts` (new); `state/useStudio.ts`, `app/TopBar.tsx`, `app/App.tsx`, `styles/layout.css`; removed `state/historySession.ts`, the SLD toolbar arrows (`DiagramPage.tsx`) and the canvas Ctrl+Z (`useCanvasShortcuts.ts`) |
| Browser: File menu | `features/project/ProjectFileDialog.tsx`, `state/saveProjectAs.ts`, `copy/projectFile.ts` (new) |
| Explorer | `features/tools/explorer/useNodeReads.ts` (new), `NodeDetail.tsx`, `explorer.css`, `copy/tools.ts`, `copy/hints.ts` |
| Disconnect, Stop, titles | `RunThisIed.tsx`, `ExplorerTool.tsx`, `MasterConnect.tsx`, `snmp/read/ConnectForm.tsx`, `TopBar.tsx` (red Stop), `GatewaySettings.tsx`, `find/FindTool.tsx`, `master/FindRegisters.tsx`, `snmp/read/TrapLog.tsx`, `explorer/WatchList.tsx`; `EquipmentSheet.tsx`, `EquipmentTest.tsx`, `tap/TapOperate.tsx`, `equipment/testButtons.ts`, `styles/workspace.css` |
| Tests | Go tests above; `tests/frontend/projectFileName.test.ts` (new); `bugHunt0177`, `exitConfirm`, `guideCopy`, `interlockHints`, `runShape` updated; spec WIN-05 points at the Save as test; `historySession.test.ts` removed |

## Evidence

- `go test ./internal/project/history/ ./internal/project/service/ ./internal/api/projectapi/
  ./internal/app/`: pass. Service tests cover undo/redo across IEDs and the Test page, Redo
  dropped by a new change, `auto_start` and `lab_ip` kept, refusal while running, a stale step
  never applied, 1 s joining, the 20-step limit, New keeping the network choice and Open clearing
  the history. Whole engine suite: pass. Three timing tests (events SSE heartbeat, licensing
  scenario, hostkey rate limit) failed once while another session ran tests and passed alone.
- Frontend: `tsc --noEmit` clean; `node --experimental-strip-types --test tests/frontend/*.test.ts`
  483 pass.
- Built engine on a throwaway data folder, page at 1366x768 (in-app browser): sample IED import →
  Undo (tip "Undo IED change (Ctrl+Z)") removes it → Ctrl+Y brings it back; SLD: add a breaker,
  drag, Ctrl+Z at once → position back, Redo offers "diagram change"; New project "North yard" →
  empty, history cleared, network kept; Close project → "Untitled substation", recovery notice
  gone; while running both arrows are greyed with "Stop the IEDs to undo or redo…"; IED Explorer
  on the running sample IED: Disconnect solid red, XCBR1.Pos opened → 4 reads in 7 s (open + every
  2 s), one "Read all now", no arrow on each object. Top bar fits at 1366 px with no cut text.

## Release 0.1.32 (together with change 0211)

- `compile_go_release.py --version 0.1.32` with `studio\.venv`: antivirus gate 19 programs, 0 findings.
  A first build numbered 0.1.33 was not published (the owner chose 0.1.32); its first antivirus run
  got no Kaspersky report and passed when run again.
- Packaged portable on a throwaway data folder: sample IED import, `GET /project/history`
  "IED change", Undo answers `can_redo`, the IED is gone, both top-bar arrows present.
- `website/deploy.cmd` needs Node 22.22+ for Astro; run with `fnm exec --using=24`. Pages
  389e4bed, "Saved release 0.1.32 from 2026-11-03". The stopped 0.1.33 run had already uploaded
  `download/ScadaBench-Portable-0.1.33.exe` to the downloads bucket (linked from nowhere).
- Live installer d9cf81b9 (19,584,425 bytes) and portable 33685e90 (53,190,656 bytes) match the
  build; copies of both downloads scan clean (2 programs, 0 findings).

## Limitations

- Save as with the Windows Save dialog was not clicked through (a native dialog); the file name
  rule is unit-tested and the fallback is the existing download link.
- The "Operate · real MMS command" titles were checked by type check and tests, not on screen
  (they need a breaker linked to a running IED).
- Changes made while IEDs run are not steps; a gateway point list edited while IEDs run keeps
  only its own one-time Undo.
- Undo restores whole top-level parts (for example the whole IED list); an automatic change to
  the same part after the step (other than `lab_ip`/`auto_start`) makes older steps unavailable.
