# 0069: Machine licensing, Go license server and protected release pipeline

Date: 2026-10-16. Owner request: implement `PROTECTION_AND_LICENSING_PLAN.md` completely;
development must behave as before, production must be solid, deployment must be lazy.
Decision: [ADR-0046](../../decisions/ADR-0046-licensing-and-protected-release.md).
Spec: [Licensing](../../specs/licensing.md).

## Change

**Go license server** (`license-server/`, Go 1.22+, bbolt): `serve`, `configure`, `status`,
`keygen`, `issue`, `revoke`, `reset-hwid`, `certificate`, `list`, `backup`. Trial (1 per HWID
ever; a reinstall during the trial gets the same trial), activation (1 PC), self-service
`transfer` (default 2 per year), `verify` (revoked/moved/valid + refreshed cert), stateless
HMAC proof of work with single-use nonces, bounded 10,000-IP LRU limiter with 30-minute ban
after 8 bad keys, `CF-Connecting-IP` trusted only from loopback, admin API only from a
direct local connection with the token (never through the tunnel), 3×5 MB capped JSONL logs,
`/buy` (PayPal Orders v2, amount/currency checked, idempotent per order), PayPal webhook
(verified via PayPal; refunds/reversals revoke), Mailgun delivery, `/activate` offline page.
Admin CLI works while the server runs (falls back to the local admin API). `deploy/setup.sh`
(installs Go if needed, builds, configures, hardened systemd unit), `deploy/push.ps1`
(one command from Windows), `deploy/license-server.service`.

**Backend:** `domain/license_policy.py` (pure rules), `licensing/` (`ed25519_verify.py`,
`certificate.py`, `hwid.py`, `client.py`, `time_anchor.py`, `service.py`, `release_key.py`),
`api/license.py`, 402 gate in `api/security.py`, wiring in `app_factory.py`, self-test
licence message, `launcher/frontend_assets.py` (folder or `frontend.pak`),
`launcher/app_window.py` (packaged only), packaged data folder in `settings.py`.

**Native:** `common/license_guard.*`, `common/obfuscate.hpp`, `release/release_key.hpp`,
vendored `third_party/monocypher/`; guard called before `start` (studio_ied), `connect`,
`goose_start`, `sv_start` (studio_live) and in `studio_probe`; CMake option
`STUDIO_ENFORCE_LICENSE` (default OFF) and `STUDIO_RELEASE_KEY_DIR`; test tool
`studio_license_check` (never packaged).

**Frontend:** `api/license.ts`, `copy/license.ts`, `features/license/` (boundary, gate,
shared key/offline actions), Settings › License, 402 → activation screen.

**Tools:** `configure_release.py`, `pack_frontend.py`, `collect_imports.py`,
`compile_protected.py` (+ `--dry-run`), `packaging/studio_protected.spec`,
`packaging/installer.iss`; `check_architecture.py` forbids `licensing` in domain;
`check_file_lengths.py` covers `license-server/` and excludes vendored `third_party/`.

## Evidence

- `go test ./...` in `license-server/`: 7 packages pass (PoW, limiter + ban + memory bound,
  log cap, one trial per HWID, 1-PC lock + reset + transfer limit, order idempotency,
  PayPal capture idempotency + amount check + refund revocation, admin refuses tunnel and
  wrong token, PayPal/Mailgun clients against fakes, pages render).
- Real binary smoke: `keygen`, `issue`, `serve`, `/api/v1/health`; `list` and `backup` while
  the server holds the database (admin API fallback).
- Python: `test_licensing.py` 14, `test_license_api.py` 4, `test_license_server_e2e.py` 3
  (real Go server: trial once, 1-PC lock, move, old PC sees "moved", unknown key),
  `test_native_license.py` 4, `test_native_enforcement.py` 4, `test_protected_packaging.py` 9.
- Native: C++ and Python machine IDs are equal on the owner PC. An enforced build with a
  test key refuses `start` without a license and for another PC, passes the gate with a valid
  one; `studio_live connect` and `studio_probe` refuse; dev builds report "not enforced".
  The normal dev binaries were rebuilt (enforcement OFF).
- Full suite: 293 passed; the one failure (`tests/packaging/test_portable.py`) checks the
  stale 2026-09-16 portable folder, built before `studio_live.exe` existed; not rebuilt
  because no EXE build was requested. Frontend: 95 node tests, `tsc`, `vite build` pass.
- Browser (throwaway backend, enforcement on, temp folders): activation screen; trial with
  an unreachable server shows a plain error; offline activation with a valid file unlocks
  the app; Settings › License shows "Lifetime license" and the masked key. The check found a
  bug (trial button disabled for new users), fixed and pinned by LIC-18.
- `verify.py`: all five checks pass.

## Local sandbox (follow-up, same day)

`studio/tools/license_sandbox.py` runs the real Go server on 127.0.0.1 and a Studio backend
with packaged licensing that trusts only that server's key, in `studio/build/license-sandbox`
(real data, license folder and release keys untouched). Browser walkthrough on the owner PC:
trial granted with real proof of work; restarted with `--days-ahead 8` the trial-ended
screen appeared with the trial button disabled; a key issued from the CLI while the server
ran activated as "Lifetime license"; `/buy` and `/activate` render (PayPal off without
sandbox credentials).

## Protected sandbox EXE and installer (follow-up, same day; owner asked to build and test all)

Additions: integrity seal + backend debugger check (`licensing/integrity.py`, LIC-19),
`tools/protected_source.py` (sealed copy; repository source untouched), Nuitka
`--mode=module --python-flag=no_docstrings` with MinGW64 fallback when no VS 2022 C++ is
installed, `compile_protected.py --sandbox`, per-user installer, `license-server
reset-trial|delete`, `tools/license_admin.py` (LIC-20).

Evidence on the owner PC (`studio/dist-sandbox`, installer 23 MB, folder 49 MB):
- Leak scan: only `studio_app.cp312-win_amd64.pyd`; no `.py/.pyc/.js/.map/.pdb/.html`; no
  docstring or source text in the `.pyd`; no JS text in `frontend.pak`; release key and
  license strings not present as plain text in the `.pyd` or `studio_ied.exe`.
- Silent per-user install (no administrator prompt) to `%LOCALAPPDATA%\Programs`.
- Installed app: activation screen; trial from the local server; template IED ran on the
  licensed sealed worker and was read over MMS by an independent client; removing the
  license gave 402 at once; `license_admin.py activate-here` re-licensed it; a copy with one
  byte appended to `studio_ied.exe` refused to run ("program files were changed");
  `--self-test` passed in 1.8 s; double-click start answered in 0.6 s in an Edge `--app`
  window with its own profile; closing the window stopped Studio and its workers in 1.2 s;
  silent uninstall removed the program and kept data and license.
- A trial started once without an intended click during browser automation (the
  automation pointer rested on the button); a hands-off reload did not reproduce it.

## Limitations

- Only the sandbox build was produced; a real release still needs `configure_release.py`
  with the server's key, then `compile_protected.py` (no `--sandbox`).
- PayPal and Mailgun were exercised only against fakes; live sandbox evidence is pending.
- The release key and server URL are not configured yet; `compile_protected.py` refuses
  until `configure_release.py` runs with the server's public key.
- Client-side licensing deters casual copying only; a patched binary can bypass it.
  Offline PCs never learn about refunds or moves. Unsigned builds trigger SmartScreen.
- libiec61850 GPL-3.0 distribution terms are handled by the owner's team (not in scope).
