# Spec: Desktop app window

Decision: [ADR-0048](../decisions/ADR-0048-desktop-app-window.md).
Owner request (2026-10-17): no console window, starting the app again shows the open
window, closing asks first and stops everything.

## Behaviour contract

- The packaged EXE is windowed; user-facing messages are dialogs, diagnostics a log file.
- One ScadaBench per data folder. A second launch brings the running window forward and exits.
- Closing the packaged window asks for confirmation, then stops the backend and all workers.
- A window handed to another browser process does not stop the app.
- Browser tabs that do not stop the app never ask on close.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| WIN-01 | Both release specs build a windowed EXE (no console). | `studio/tests/unit/test_packaging_windowed.py::test_release_specs_build_windowed_exe` |
| WIN-02 | A second launch asks the running instance (per data folder) to show itself and gets an answer. | `studio/tests/unit/test_app_window.py::test_second_launch_asks_the_running_instance_to_show`, `studio/tests/unit/test_app_window.py::test_event_name_is_per_data_folder`, `studio/tests/unit/test_app_window.py::test_second_launch_focuses_the_recorded_window_itself` |
| WIN-03 | Showing again focuses the live window, reopens a default-browser tab, and never reopens a window the user closed. | `studio/tests/unit/test_app_window.py::test_show_focuses_the_running_window`, `studio/tests/unit/test_app_window.py::test_show_reopens_the_default_browser_tab_without_edge`, `studio/tests/unit/test_app_window.py::test_show_does_not_reopen_a_window_the_user_closed` |
| WIN-04 | Closing the window stops the app; a hand-off detected by the held profile lock does not; a leftover window never adopts the new profile. | `studio/tests/unit/test_app_window.py::test_closing_the_window_stops_the_app`, `studio/tests/unit/test_app_window.py::test_handoff_to_another_browser_keeps_the_app_running`, `studio/tests/unit/test_app_window.py::test_profile_lock_held_by_a_browser_is_detected`, `studio/tests/unit/test_app_window.py::test_a_leftover_window_never_adopts_the_new_profile` |
| WIN-05 | Only the launcher-marked app window registers the exit prompt; the project download does not navigate. | `studio/tests/frontend/exitConfirm.test.ts::only the packaged app window asks before closing`, `studio/tests/frontend/exitConfirm.test.ts::project download is a link, not a navigation` |
| WIN-06 | Launches from a terminal print messages instead of showing dialogs. | `studio/tests/unit/test_app_window.py::test_terminal_launches_print_instead_of_dialogs` |

## Evidence still required

A built windowed EXE: double-click, second launch focusing the window, close prompt and
worker shutdown. None is claimed until a change record shows it.
