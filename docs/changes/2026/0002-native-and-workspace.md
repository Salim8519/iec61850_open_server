# Change 0002: native runtime and engineering workspace

Date: 2026-09-09. Status: implementation milestone; not release acceptance.

## Delivered code

- Flask/Waitress launcher, local session security, versioned API and SSE events.
- React equipment canvas, binding inspector, IED/signal table, source controls,
  endpoint list and event drawer using the fixed design tokens.
- Native libiec61850 workers with framed IPC, exact loopback binding, typed
  updates, reports, direct/SBO controls and explicit request/completion events.
- Independent MMS client executable for reads, controls and report observation.
- CID/ICD/SCD semantic import, original-source retention, laboratory template,
  portable project archive and atomic autosave.
- Source definitions and stable project/IED identifiers survive reopen.
  Restores validate diagram and source references before replacing a project.

## Evidence

- Frontend TypeScript and production Vite build pass.
- Native x64 Release worker and probe compile with VS2019 BuildTools.
- Native tests observe read/update, enhanced SBO control and changed feedback.
- Both buffered and unbuffered tests observe GI and nested FLOAT32 data-change
  reports over TCP/MMS. Nested BDA trigger inheritance and FLOAT32 updates fixed.
- Negative command termination is no longer treated as successful completion.
- Read-only deterministic sample audit of six customer CID files passed semantic
  import. Five IEDs in one sample each started and answered independent MMS reads.
  The largest IED in that sample contains 36,021 scalar attributes.
- The audit scripts read customer originals; no customer models are committed.
- `python studio/tools/verify.py` passes maintained-file length and boundary checks.

## Limitations

Browser automation failed before connection with an environment kernel-assets
path error; screenshots and interactive acceptance remain unverified. A successful
frontend build is not visual acceptance. See [known issues](../../known-issues.md)
for outstanding product work. Runtime listener tests use loopback only and make
no adapter or firewall changes. The existing Docker demo is preserved.
