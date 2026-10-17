# 0080: License server "cannot be reached" — TLS verified the Windows way

Date: 2026-10-17. Decision: [ADR-0056](../../decisions/ADR-0056-windows-tls-verification.md).

## Owner report

- "Cannot reach the license server. Check the internet connection, or use offline
  activation" in the installed app, also when starting the 7-day trial.

## Diagnosis

- Server healthy: curl `GET /api/v1/health` → 200 in 0.9 s.
- App client from source: `URLError(SSLCertVerificationError(... certificate has expired))`.
- Chain served since 2026-10-16: `CN=scadabench.com` (LE YE2, to 2027-01-14) → `YE2` (to
  2028-09-02) → `Root YE` cross-signed by `ISRG Root X2` (to 2032-09-02); openssl verify OK.
- Windows store here: root `ISRG Root X1` only; intermediate store holds `ISRG Root X2`
  signed by X1, expired 2025-09-15. OpenSSL used that path; CryptoAPI does not.

## Change

- `studio/backend/studio_app/licensing/client.py`: `https_context()` (truststore,
  Windows verification) and `open_https()` as the client's default opener.
- `studio/pyproject.toml`: `truststore==0.10.5`.
- `studio/tools/collect_imports.py`: `truststore` declared, so the protected build bundles it.
- `studio/tests/unit/test_licensing.py`: the client uses the Windows-verifying context.
- `docs/specs/licensing.md`: LIC-21.

## Evidence / limitations

- Live, through the app's own `LicenseClient`: `GET /api/v1/health` → `{"ok": true, ...}`
  (it failed with the expired-certificate error before the change).
- Same context still refuses `expired.badssl.com`, `wrong.host.badssl.com`,
  `self-signed.badssl.com`.
- 290 unit tests pass (53 licensing/release/protected, incl. the packaging guard that
  first flagged the undeclared import).
- **Not fixed for installed 0.1.x copies** until a new build is installed; their licenses
  keep working offline. No EXE/installer built in this change. No server change made.

## Release 0.1.3 (2026-10-17, owner-authorised deploy)

- Coordinated with the Tools-tab and startup-speed panes; built from the shared worktree
  after 293 unit + 105 frontend tests, typecheck and verify.py passed (incl. the fixed
  real-IED read-only contract). Ships this fix, change 0084 (fast restore) and the Tools tab.
- `compile_protected.py --version 0.1.3`: native workers (enforced) built clean, Nuitka
  module sealed and signed, leak check clean, `ScadaBench-Setup-0.1.3.exe` 23.2 MB,
  SHA-256 `fe893e22…a938360`, unsigned. `truststore` (incl. `_windows`) is in `PYZ.pyz`.
- Built app, isolated `LOCALAPPDATA`: activation of a well-formed unknown key reached
  `license.scadabench.com` and got `unknown_key` in 1.6 s (0.1.x: "cannot reach").
- Installed silently over 0.1.2 on the owner's PC (`DisplayVersion 0.1.3`); installed
  `--self-test` ok with the owner's real license. Website download not yet republished
  (deploy.cmd also publishes an unapproved website redesign; owner decision pending).

## Release 0.1.4 and website publish (2026-10-17, owner: "just deploy all")

- 0.1.3 was never published; 0.1.4 adds the launcher speed-up (change 0085, ADR-0058).
  Before the build: 341 Python unit+integration, 107 frontend tests, tsc, verify.py PASS.
- `ScadaBench-Setup-0.1.4.exe` 23.3 MB, SHA-256 `f6b83b71…0701af08`, leak check clean,
  unsigned. Built app (isolated, own port): version 0.1.4, enforced, unlicensed engineering
  API → 402, live activation probe → `unknown_key` in 1.6 s.
- Probe pitfall recorded: a dev backend on 127.0.0.1:8765 (the packaged default port) took
  the probe's requests (dev-wording 401); rerun with `--port 0` was clean.
- Installed silently over 0.1.3 (DisplayVersion 0.1.4); installed `--self-test` ok with the
  owner's license.
- Website: `site.ts` version 0.1.4, changelog entry for 0.1.4 (first release pinned to 0.1.0);
  `deploy.cmd` published the approved redesign + installer + 720p videos. A stale
  `astro preview` (since 00:35) locked `node_modules` and broke `npm ci`; it was stopped and
  the deploy rerun. Live checks: `/`, `/pricing/`, `/tutorials/`, `/legal/eula/` (`#refunds`
  present), `/changelog/`, `/download/` (Version 0.1.4), `/videos/getting-started.mp4`
  (3.5 MB) all 200; `/download/ScadaBench-Setup.exe` SHA-256 equals the 0.1.4 build.
