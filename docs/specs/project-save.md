# Spec: Saving the project

Decisions: [ADR-0135](../decisions/ADR-0135-project-undo-file-menu.md) (File menu),
[ADR-0139](../decisions/ADR-0139-command-popups-and-auto-save-switch.md) (auto save switch).
Owner request (2026-11-11): a switch to turn project auto save off, on by default, remembered.

## Behaviour contract

- Auto save is on by default: every change is written at once and survives a crash.
- With auto save off, changes stay in the open project and every page uses them, but nothing is
  written until File › Save project or Ctrl+S. Closing first gives back the last saved project.
- Turning auto save on writes the unsaved changes first.
- The choice is remembered across restarts.
- The top bar says when auto save is off and when changes are not saved.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| SAVE-01 | With auto save off a change is not written; leaving without Save gives back the last saved project. | `studio/engine/internal/project/service/manual_save_test.go::TestAutoSaveOffWritesOnlyOnSave` |
| SAVE-02 | Save writes the unsaved changes; turning auto save on writes them, and later changes are journaled again. | `studio/engine/internal/project/service/manual_save_test.go::TestSaveAndTurningAutoSaveOnWrite` |
| SAVE-03 | The choice is the Go-only preference `autosave_off` (0 or 1), absent by default. | `studio/engine/internal/project/preferences_autosave_test.go::TestAutoSaveOffPreference` |
| SAVE-04 | The top bar says "Auto save off" or "Not saved · Ctrl+S" while auto save is off, and keeps its own words while it is on. | `studio/tests/frontend/autoSaveSwitch.test.ts::Top bar words follow the auto save switch` |

## Evidence still required

None for the switch itself; the packaged EXE is checked with the next release.
