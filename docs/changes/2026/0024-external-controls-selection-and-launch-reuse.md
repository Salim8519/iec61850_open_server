# 0024: External controls, stable selection and launcher reuse

Date: 2026-09-16. Implements
[ADR-0015](../../decisions/ADR-0015-native-control-feedback.md) and refines
[ADR-0012](../../decisions/ADR-0012-one-development-launcher.md).

SPC and DPC native handlers now apply their status atomically and publish the raw
Boolean control value. The coordinator mirrors every accepted Studio or external
MMS command into all diagram nodes sharing that exact control binding. Each node's
configured polarity and optional state mapping are respected; no reference-name
rules are used. A reset sent to the shared Trip relay command therefore updates
its separately selected state point.

React Flow selection is now derived from the same selected node ID used by the
inspector. Transient selection changes are excluded from persisted diagram edits,
and deleting the selected node clears the inspector.

When backend port 8765 is occupied, `start-studio.cmd` searches the allowed Vite
range and verifies Studio through the authenticated session endpoint. It reopens a
healthy existing development session instead of starting a duplicate. It still
refuses to kill or reuse an unrelated listener. The normal Release worker and probe
were rebuilt; the packaged executable was not rebuilt.

Evidence:

- Six focused unit tests passed, including launcher reuse and equipment operation.
- Three independent native protocol tests passed against an isolated candidate,
  including external SPC false/true, shared command mappings and DPC feedback.
- The normal Release `studio_ied.exe` and `studio_probe.exe` rebuilt successfully.
- Frontend TypeScript and production Vite build passed.
- The complete maintained suite passed all 113 tests against the normal development
  native build.
- Static file-length, architecture and documentation verification passed.
- A second real `start-studio.cmd --no-browser` invocation detected and reused the
  live Vite session on port 5174, returned success and started no duplicate tree.

The in-app browser harness failed before attaching because its local kernel-assets
path was unavailable, including after one clean reset. Interactive visual acceptance
and third-party interoperability remain release gates.
