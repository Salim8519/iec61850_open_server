# 0017: Report mapping, switch motion, saved tests and bulk deletion

Date: 2026-09-09. Implementation follows the user's confirmed feature plan and
[ADR-0009](../../decisions/ADR-0009-report-mapping-and-signal-tests.md).

Added qualified report/dataset membership and description search; shared explicit
report-first picker; CB/DS/ES SVG transitions and typed position interpretation;
saved independent Signal Test controls; native revision-guarded Boolean pulses;
source-override release; bulk deletion with combined impact and stale-preview guard.

New HTTP interfaces under `/api/v1`:

- `GET /ieds/{id}/signals`: optional `report`, exact `ref`, description/path `q`,
  existing `fc`, `offset` and `limit` filters. Report uses its full reference.
- `GET/PUT /project/test-panel`: enriched live entries / saved ordered definitions.
- `POST /signals/pulse`: IED, reference, Boolean `active_value`, `duration_ms`.
- `POST /signals/release-override`: IED and reference; resumes its automatic source.
- `POST /ieds/removal-preview`: unique `ids`; returns combined impact/fingerprint.
- `POST /ieds/remove`: same IDs, fingerprint and `confirm: true`; returns recovery path.

Project format 3 retains compact source storage; formats 1/2 remain readable.
Backend pulses continue without a browser and while automation is paused. Sample
writes and real MMS controls invalidate stale native pulse guards. A failed return
is visible and never retried over a newer value. Structural edits require Stop.

Candidate builds are isolated from the user's 13 active IED workers. The user's
latest autosave (13 IEDs, 3 drawing items) passed migration and engineering
roundtrip checks without modifying that autosave. This reflects the user's latest
changes, not the older 15-IED backup.

Verification: 93 backend/unit/integration/protocol tests passed, including exact
report filtering, legacy migration, saved controls, stale-preview rejection,
save-failure rollback, real MMS pulse reads, buffered/unbuffered pulse reports,
pause behavior, newer writes, control revision guards and source override release.
Three frontend state tests passed; TypeScript checks and the isolated UI build passed.
The portable candidate also passed both packaging tests, including saved-panel
recovery, pulse restoration, deletion recovery and independent native MMS checks.
File-length, architecture and documentation checks passed.

Candidate outputs: `studio/native/build/feature-preview/Release`,
`.local/studio-ui-candidate`, `studio/dist/feature-preview/SubstationStudio`.
Build logs: `.local/native-feature-build.log`, `.local/portable-feature-build.log`.
The native start handshake now advertises guarded-update support; an outdated
worker is rejected with a rebuild instruction and all partially started workers
are closed. The original running process is unaffected by this source change.

The browser connection still fails before session setup (missing kernel-assets path).
No interactive or visual acceptance is claimed. The existing live session remains
on its previous version until a stopped, backed-up promotion can be performed.

Live promotion: the user explicitly requested an immediate restart despite ongoing
save changes. Preserved a quick autosave copy under data/recovery, stopped only
the identified old backend, and launched the candidate portable executable with
the existing studio/data directory on port 8765. HTTP 200 and the updated UI asset
were confirmed; the launcher opened a fresh browser session. No saved project was
replaced with an older backup. Visual acceptance remains pending.
