# 0076: Pre-v2 licenses survive the update (no re-activation, no move spent)

Date: 2026-10-17. Decision: [ADR-0051](../../decisions/ADR-0051-legacy-hwid-acceptance.md).

## Why

The first HWID v2 build refused the owner's own lifetime license ("belongs to a
different computer"). Re-activating would spend one of the key's 2 yearly moves, and
the weekly check would have reported a v1-bound key as `moved` and removed it.
Owner: no customer should suffer this on update.

## Changed files

- `studio/backend/studio_app/licensing/hwid.py`: v1 `compose_legacy`,
  `legacy_machine_id`, `bound_hwid`, `LEGACY_ISSUED_BEFORE` (2027-07-01Z).
- `studio/backend/studio_app/licensing/service.py`: `legacy_hwid()`,
  `_machine_for(payload)` used by evaluation and install; online check queries the
  bound ID and keeps a legacy-bound certificate.
- `studio/native/common/license_guard.cpp` + `.hpp`: shared `machine_text`,
  `compose_legacy_hwid`, `compute_legacy_hwid`, `kLegacyIssuedBefore`; optional
  `hwid_issued_before` on `verify_certificate_text`; legacy retry in
  `require_valid_license()`.
- `studio/native/testtools/license_check.cpp`: `legacy-hwid`, `compose-legacy`,
  optional ISSUED_BEFORE for `verify`.
- Tests: new `tests/unit/test_licensing_legacy.py`; legacy parity/cutoff in
  `test_native_license.py`; enforced worker legacy acceptance in
  `test_native_enforcement.py`.

## Evidence

- Python: `test_licensing`, `test_licensing_legacy`, `test_release_guards`,
  `test_license_api`: 29 passed.
- Native rebuilt (dev + `license-enforced`): `test_native_license` +
  `test_native_enforcement` 11 passed; enforced `test_native_license` 6 passed. C++ v1
  ID equals Python `legacy_machine_id()` on this PC; the enforced `studio_ied` accepts
  a v1-bound pre-cutoff certificate and refuses a post-cutoff one.
- The owner's real certificate (issued 2026-10-17, from 0.1.0 era) matches this PC's
  v1 ID and is before the cutoff.
- `compile_protected.py --version 0.1.2`: enforcement ON, leak check clean,
  `ScadaBench-Setup-0.1.2.exe` 23 099 252 bytes, sha256 `c04537e31d68c4b4…`, unsigned.
  Packaged `--self-test` returned `ok: true` with the owner's real v1-bound license,
  no re-activation. Full unit suite 288 passed; `verify.py` PASS.

## Limitations

- Pre-cutoff certificates keep v1 (two-factor) strength. After 2027-07-01 remaining
  v1 certificates need one re-activation unless the constant is extended.
- A former v1 trial machine could obtain one additional trial under its v2 ID.
