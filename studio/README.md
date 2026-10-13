# Substation Studio

Portable browser-based IEC 61850 laboratory simulator. Development is in progress.

Read ../AGENTS.md and ../docs/INDEX.md before changing this application.
Verify structure with `python studio/tools/verify.py` from the repository root.
The original repository demonstration remains available independently.

See [native build instructions](native/README.md) for normal and isolated builds.

Run `start-studio.cmd` from the repository root. This is the single development
entry point: it starts current backend source and Vite in one console, opens the
browser, applies frontend edits through HMR and restarts on Python changes. It
uses `studio/data`; Ctrl+C stops only the frontend/backend trees it started.

After native C++ changes, build `studio/native/build/Release`; the development
backend watches that worker and restarts when the executable is replaced. Launch
a frozen preview directly from its `studio/dist/.../SubstationStudio.exe` path.
