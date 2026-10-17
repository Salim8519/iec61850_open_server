# 0075: Release hardening — dev lockout, version stamp, anti-debug, HWID v2

Date: 2026-10-17. Decision: [ADR-0050](../../decisions/ADR-0050-release-hardening.md).

## Owner report

- The first published app was reviewed for reverse-engineering resistance.
- Four code gaps to close now: the installed app still ran developer mode (fixed
  session secret); there was no version shown anywhere; the debugger check was
  minimal; the hardware ID used only two easily changed values.
- Authenticode code signing is deferred (a later certificate purchase).

## Change

- `launcher/entrypoint.py`: refuse `--dev` when `sys.frozen` (installed app never runs
  the reloader or the fixed `studio-dev-secret`/dev token); source dev unchanged.
- `studio_app/version.py` (new): single `__version__`. `tools/protected_source.py`
  (`version_source`, `prepare(..., version)`) stamps the real version into the sealed
  copy; `tools/compile_protected.py` passes `--version` to the backend step.
- `api/license.py`, `api/health.py`: responses include `app_version`.
- Frontend `api/license.ts` (type) and `features/settings/LicenseSection.tsx`: a
  "Version" row in Settings › License.
- `native/common/integrity.{hpp,cpp}` (new): `check_no_debugger()` (PEB BeingDebugged +
  NtGlobalFlag added; no timing checks), and `verify_module()` + `module_keys()` for the
  compiled-module signature (Fix 2). Added to the `studio_license` library in `CMakeLists.txt`
  (plus `native/release` on the include path for `module_key.hpp`).
- Module signature (Fix 2): `licensing/ed25519_sign.py` (new, dependency-free signer),
  `tools/module_signing_key.py` (new, one-time key + `native/release/module_key.hpp`;
  secret seed git-ignored), `tools/compile_protected.py` (`sign_module` after Nuitka),
  `packaging/studio_protected.spec` (ships the `.sig`), `launcher/entrypoint.py`
  (`_announce_signed_module`, only when a `.sig` is present), `testtools/license_check.cpp`
  (`verify-module` command), `require_valid_license()` calls it when the backend provides
  the module path.
- `native/common/license_guard.cpp` + `.hpp`: HWID v2 (SMBIOS system UUID third factor,
  `substation-studio-hwid-v2|…`); `require_valid_license()` calls
  `integrity::check_no_debugger()`; the local `check_debugger` is removed.
- `licensing/hwid.py`: HWID v2 in lockstep (`_firmware_uuid`, `_smbios_system_uuid`).
- Tests: `test_licensing.py` and `test_native_license.py` updated to the 3-argument
  `compose`/v2 format; `test_release_guards.py` (dev lockout + `app_version`) kept.

## Pending (not in this change)

- Authenticode signing: a certificate purchase; build hook already exists
  (`compile_protected.sign`, `STUDIO_SIGN_PFX`).
- Fix 2 packaged end-to-end: confirmed by a signer/verifier round-trip here; the live
  packaged run (workers verifying the real shipped `.pyd`) is checked on the next
  authorized EXE build. Safe by design until then (unsigned build enforces nothing).

## Evidence / limitations

- 280 unit tests pass (`studio/.venv`); `verify.py` PASS (file lengths, architecture,
  documentation, history, specifications). All new/edited files < 300 lines.
- Native rebuilt for tests (root + `license-enforced`): C++ `compute_hwid` equals
  Python `hwid.machine_id()` on this machine; `compose` parity with fixed inputs incl.
  the UUID; the enforced worker accepts a v2-bound certificate and still refuses a
  wrong-machine / missing certificate.
- Module signature round-trip proven (`studio_license_check verify-module`): a
  Python-signed module verifies "ok"; a tampered module and a wrong key both fail.
- HWID v2 invalidates existing licenses (re-activation needed). No EXE/installer built.
- Full packaged end-to-end (windowed app, live module signature) pending an authorized build.
