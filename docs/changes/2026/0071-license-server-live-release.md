# 0071: License server live, payments tooling, first real release

Date: 2026-10-17. Owner requests: one lazy command to deploy the backend to their own server
(shared with the unrelated Baisa project, low end, dedicated account, tunnel), Mailgun and
PayPal set up with a $0.10 real test, keep the owner license tool, build the app.
Decisions: [ADR-0046](../../decisions/ADR-0046-licensing-and-protected-release.md),
[ADR-0047](../../decisions/ADR-0047-scadabench-brand-website-tour.md). Follows
[0069](0069-licensing.md) and [0070](0070-scadabench-brand-website-tour.md).

## Change

- **Server deploy** (`deploy-server.cmd` → `license-server/deploy/push.ps1`):
  - cross-compiles linux amd64/arm64 on Windows (no Go on the server), uploads with scp and
    runs `setup.sh` as the dedicated `scadabench` user (`bootstrap-user.sh`: limited sudo);
  - listens on `127.0.0.1:18090` (port 8090 belongs to Baisa) and runs its own
    `scadabench-tunnel.service` with `cloudflared-scadabench`; Baisa's `cloudflared.service`
    is untouched;
  - the tunnel token travels over stdin, never in arguments; a UTF-8 BOM is stripped
    because it made systemd ignore `TUNNEL_TOKEN`;
  - `-Price` and `-Configure` change settings through `license-server set`.
- **Payments and email secrets** (`set-payments.cmd` → `deploy/secrets.ps1`): hidden input
  for the PayPal client secret and the Mailgun sending key, sent over stdin to
  `license-server-admin set --secrets-stdin`, then a restart. IDs are remembered in
  `%LOCALAPPDATA%\ScadaBench-deploy\payments.json`; secrets are never stored on Windows.
  `-Mode live` switches to real PayPal.
- **Owner tool:** `studio/tools/license_admin.py --real` reaches the live server through the
  deploy key and the admin wrapper (`activate-here`, `remove`, `key`, `list`).
- **Release configuration:** `configure_release.py` set the live URL and key id `768dc76e`
  (`licensing/release_key.py`, `native/release/release_key.hpp`).
  `test_license_api` now points at an unreachable loopback server, so it no longer depends on
  the release URL.
- **Release build:** `studio/dist-protected/ScadaBench-Setup-0.1.0.exe` (23 063 023 bytes,
  sha256 `1cc490db8a97458d…`) and the portable `ScadaBench/` folder.
- **Website:** the installer is published at `/download/ScadaBench-Setup.exe`.
- **Checkout pages:** `/buy` and `/activate` now use the website design. They share
  `pages/layout.html` (header, logo, footer with terms, privacy and contact), Geist fonts served
  from the binary (`FONTS-OFL.txt`), an order summary with the total, a white payment panel and
  trust notes. `pageData.Amount` shows "$149.00".
- **Live PayPal:** a live app "ScadaBench" and a webhook to `/api/v1/webhooks/paypal`.
  `secrets.ps1` now single-quotes the sender name, because Windows PowerShell dropped the inner
  double quotes and bash read `<...>` as a redirect. It also stops early when the IDs are missing.
  Kaspersky quarantined `secrets.ps1` once; it was recreated.

## Evidence

- Live: `https://license.scadabench.com/api/v1/health` answers 200 with kid `768dc76e`
  through the tunnel. The admin API answers 403 from the internet. The server status shows
  the right listen address, public URL, price and key id.
- The tunnel secret was rotated after the BOM-broken first start exposed it in the service
  log. The fixed unit starts cleanly with the new secret.
- Real build: the leak check passed. `ScadaBench.exe --self-test` from the real portable
  build licensed against the live server and read an IED value over MMS (42.5).
- The owner PC holds a lifetime license issued by the live server (`activate-here --real`).
- The live download is byte-identical to the build (sha256 compared after download).
- Tests: Python 333 passed, 9 skipped (portable tests need `dist/ScadaBench`; GOOSE/SV need
  a lab adapter); frontend 98 passed; `tsc` clean; Go all packages pass; `verify.py` pass.

## Limitations

- Live checkout and email are ready, at a test price of 0.10 USD. The owner's two card attempts
  were refused inside PayPal's card form. Our three create-order calls returned 201, and no capture
  followed. The account settings have "PayPal account optional" on and no payment blocks. The
  refusal is PayPal's or the bank's risk decision.
- The new pages were checked in a desktop browser only; the phone layout relies on the 860 px
  breakpoint and was not checked in a phone-width window.
- The real $0.10 test, a refund and the return to 149.00 are still to do.
- The installer is not code-signed, so SmartScreen warns on first download.
- Tutorial videos are not rendered yet.
