# 0027: CONCEN compatibility and authoritative tab synchronization

Date: 2026-09-16. Implements [ADR-0018](../../decisions/ADR-0018-concen-sbow-compatibility.md)
and [ADR-0019](../../decisions/ADR-0019-single-session-project-revisions.md).

Enhanced-SBO DPC command mappings can explicitly enable a warned CONCEN profile.
Only exact origin `CONCEN` is affected. Its Boolean value is inverted and a
successful `SBOw` queues immediate breaker execution; later matching `Oper` is
acknowledged without duplicate movement. Standard clients, all SPC controls and
the corrected Trip Relay path are unchanged. Native events preserve the raw value,
executed value, semantic actions and compatibility trigger.

Project format 8 stores the explicit profile. Formats 1 through 7 migrate to
standard behavior and receive a byte-identical verified recovery archive. The
native worker advertises a new required capability so a reloaded backend cannot
silently run an older engine.

Every browser tab now shows one backend session ID and project revision. Saved
mutations notify connected tabs. Remote edits replace stale local projections,
while revision-checked diagram writes return conflict before an old tab can
overwrite newer work. Tabs served by an inactive development frontend port redirect
to the backend's authoritative Vite origin.

Evidence:

- The isolated and normal native worker/probe builds completed successfully.
- An independent MMS probe using origin `CONCEN` sent only `SBOw(false)`; the
  mapped enhanced-SBO DPC moved from Open to Closed and emitted raw=false,
  executed=true, action=Close evidence.
- A standard-origin control on the same profiled object retained false=Open.
- All 123 Python tests passed against the rebuilt normal worker/probe, including
  native protocol, mapped feedback, format migration and stale-tab rejection.
- All 9 frontend contract tests and the production TypeScript/Vite build passed.
- File-length, architecture and documentation verification passed.
- The stopped live C03 project was revision-checked before enabling the profile on
  its mapped `CB-1`; the Trip Relay binding remained standard and unchanged.

The supplied customer packet capture remains local. No packaged executable was
built. Interactive multi-tab/browser acceptance and third-party concentrator
retest remain release gates.
