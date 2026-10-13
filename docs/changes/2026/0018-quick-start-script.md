# 0018: One-command start for the packaged app

Date: 2026-09-15.

Added `start-studio.cmd` at the repository root for development sessions that
must not rebuild. It selects the newest `SubstationStudio.exe` under
`studio/dist` (currently the feature-preview candidate promoted live in change
0017), passes the shared `studio/data` project folder, forwards any extra
launcher arguments, and prints the executable it chose. Ctrl+C in the same
window stops the app gracefully.

Evidence: `start-studio.cmd --self-test --data-dir <temporary folder>` selected
the feature-preview executable and returned `{"ok": true, ...}` including the
independent loopback MMS checks. `python studio/tools/verify.py` passes for the
new files.

Limitations: the script runs the packaged snapshot only; backend, frontend and
native edits require rebuilding and repackaging before they appear. It performs
no build, changes no project data, and makes no clean-machine or visual
acceptance claim.

Superseded on 2026-09-16 by change 0021: the root command now always starts the
source development workflow so it cannot be confused with this packaged snapshot.
