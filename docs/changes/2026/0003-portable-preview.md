# Change 0003: Windows portable developer preview

Date: 2026-09-09. Status: build verified; release acceptance pending.

PyInstaller one-folder build bundles Python, Flask/Waitress, frontend assets,
native worker/client executables and the native dependency license files.
Output: studio/dist/SubstationStudio/SubstationStudio.exe.
The complete folder is required; the executable alone is not portable.

Build command (after native and frontend builds):
`studio/.venv/Scripts/python.exe studio/tools/build_portable.py`.
The build tool refuses to replace a preview folder containing a data directory.
Generated build output is ignored by Git.

The current launcher opens the browser and keeps a console available for Ctrl+C
shutdown. Tray mode and automatic recovery are not yet implemented. Source
definitions persist in project archives; runtime overrides reset on restart.

This preview has not been verified on a clean Windows machine without developer
tools. External redistribution requires complete dependency license/source
compliance review. It must not be presented as a finished release.

The packaging smoke test launches the frozen executable on an ephemeral local
port, checks the HTML shell and unauthenticated API rejection, and terminates only
its own process. It starts no IED workers and does not modify adapter settings.

Verification: full suite passes (16 tests), including frozen startup/security,
project persistence and real MMS reads, controls, buffered/unbuffered reports.
File-length, architecture and documentation checks pass. These tests ran on the
development PC; clean-machine portability remains a separate acceptance gate.
