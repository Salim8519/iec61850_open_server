# ADR-0051: Keep pre-v2 licenses valid on the same PC after an update

Accepted: 2026-10-17. Owner report: after building with HWID v2 (ADR-0050) the owner's
own lifetime license failed with "This license belongs to a different computer".
Owner requirement: no customer may suffer the same when they update the app.

**Problem**

ADR-0050 changed the machine ID from v1 (MachineGuid + volume serial) to v2 (+ SMBIOS
system UUID) and accepted that existing licenses need re-activation. Re-activation of
a bound key is a server *transfer*, limited to 2 per key per year
(`license-server` `MaxTransfersPerYear`). Every 0.1.0 customer would spend one of
their two yearly moves just to update. Worse, the weekly online check sends the
machine ID; a v1-bound key queried with the v2 ID answers `moved`, which deletes the
local certificate.

**Decision**

- **Legacy acceptance, client side only.** Python (`licensing/hwid.py`:
  `compose_legacy`, `legacy_machine_id`, `bound_hwid`) and native
  (`license_guard.cpp`: `compose_legacy_hwid`, `compute_legacy_hwid`) also compute the
  v1 ID. A certificate whose `hwid` equals this PC's v1 ID is accepted **only when
  issued before `LEGACY_ISSUED_BEFORE` = 2027-07-01T00:00:00Z** (same constant in
  `hwid.py` and `license_guard.hpp`). New activations always bind v2.
- **Native:** `require_valid_license()` tries v2 first; only on a wrong-machine
  failure it retries with the v1 ID and the cutoff (`verify_certificate_text`
  gained an optional `hwid_issued_before`, default 0 = unchanged behaviour).
- **Online check:** `verify_online_if_due` asks the server about the machine ID the
  certificate is actually bound to, so a v1-bound key is not reported as moved. A
  reissued certificate is not written over a legacy-bound one (a new `issued_at`
  would fall after the cutoff).
- **No server change, no transfer spent, no customer action.**

**Compatibility and limits**

- Pre-cutoff certificates keep v1 strength (MachineGuid + volume serial), which is
  exactly what they had before; v2 hardening applies to every newer activation.
- After 2027-07-01 any still v1-bound certificate needs one re-activation. Extend the
  constant in a later release if 0.1.0 installs are still common, or add a server-side
  free rebind.
- A former v1 *trial* user could start one more trial under the v2 ID (server
  dedupes trials by machine ID). Accepted: one extra 7-day trial per old machine.
- Supersedes the "re-activation needed" consequence of ADR-0050; the rest of
  ADR-0050 stands.
