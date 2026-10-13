# 0014: Stopped startup recovery and launcher ownership

Date: 2026-09-09. Status: source/API tested; packaged and visual acceptance pending.

The launcher now locks its data directory before opening services. OS byte-range
locking prevents another launcher from overwriting the same autosave and releases
on normal or abrupt process exit. Lock files may remain harmlessly; symbolic-link
lock targets are refused.

Startup restores valid autosave engineering state without rewriting the archive
or starting workers. A damaged autosave receives an exact separate recovery copy
before new work is allowed. Failed preservation stops startup. The browser shows
a stopped-recovery notice or a damage warning with the preserved file location.
Project restore also rolls back in-memory state if the final save fails.
See [decision](../../decisions/ADR-0007-startup-recovery.md).

Evidence:

- Tests cover stable project/IED identities, diagram bindings, sources and enabled
  scenarios, unchanged original autosave bytes and zero running workers/listeners.
- Corrupt archives remain preserved after creating a new project. Injected backup
  failure stops recovery; injected save failure preserves the current project.
- Independent-process tests verify concurrent lock refusal, ordinary release and
  OS release after immediate process exit without normal cleanup.
- Full regression: 76 passed in 17.16 seconds with all native protocol tests using
  the isolated array-fix candidate. A subsequent symbolic-link lock guard is a
  defensive source change, not separate filesystem-link acceptance evidence.
- Frontend TypeScript and production build pass. Browser visual/workflow checks
  remain blocked by the previously recorded browser-tool startup error.
- Normal worker promotion and portable rebuild remain pending. Existing frozen
  preview tests do not verify these new recovery/locking behaviors in a new bundle.

This is engineering-state recovery, not automatic resumption of runtime control
state or a guarantee against sudden power-loss/disk failure. No IED, source or
scenario starts until the user explicitly presses Run.
