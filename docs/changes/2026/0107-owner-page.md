# 0107: Owner page at scadabench.com/<owner path>

Date: 2026-10-19. Decision: [ADR-0072](../../decisions/ADR-0072-owner-page.md). Spec: [owner page](../../specs/owner-page.md).

Owner request: a way to make `SUBS-` keys for friends that stays secure if something is stolen, then
a phone-friendly owner page with two-step sign-in and recovery: watch purchases and buyer emails, make
keys, trace issues, free disk space, send news. The path is chosen by the owner and kept out of the repo.

## Change

- `internal/adminauth/`: PBKDF2-SHA256 password hash (stdlib, Go 1.24), RFC 6238 codes with replay
  refusal, 10 hashed recovery codes, account file with epoch, memory sessions, global lockout.
- `internal/audit/`: append-only `admin-audit.jsonl`.
- `internal/store/subscribers.go`: news consent list with unsubscribe tokens; `License.note`.
- `internal/api/owner*.go`: gateway check (404 without the secret), session/CSRF/origin checks,
  fresh-code confirmation, licenses, overview and purchases, issues from the server log, disk view
  and old-log cleanup, news test/send with progress; `diskspace_*.go`; public `/unsubscribe`.
- Checkout: "Email me about new versions" box (unticked); consent rides in the signed invoice nonce.
- `pages/admin.html`, `admin.css`, `admin-core.js`, `admin-views.js`, `admin-ops.js` (CSP-safe, all
  text via `textContent`), `unsubscribe.html`.
- `cmd/license-server/adminsetup.go`: `admin-setup --password-stdin [--json]`, `admin-status`;
  `set` gains `--admin-path`, `--admin-origin`, `--marketing-from`, `--marketing-domain` and the
  `admin_gateway_secret` stdin secret.
- `gateway/`: Cloudflare Worker `scadabench-owner-gateway` (route given at deploy).
- `deploy/admin.ps1` with root commands `admin-setup.cmd` and `reset-admin.cmd`; `setup.sh` Go
  fallback 1.25.6; `go.mod` go 1.24.

Follow-ups the same day: owner password minimum 8 characters (owner request); **Price** box on
Overview (`POST <owner path>/api/price`, fresh code required) saves `config.json` and applies at once
without a restart (`ownerprice.go`, all price reads go through `Server.price()`). **Versions** card on
Overview (`GET <owner path>/api/versions`, `ownerversions.go`): the installer version the website's
download page links (read at most every 5 minutes) against the published update-check record, a
"not in step" hint, and the server program's build time. Customer versions stay unknown (ADR-0071).

## Evidence

| Check | Result |
| --- | --- |
| `go vet ./...`, `go test -count=1 ./...` | pass (new: adminauth 6, audit 1, subscribers 1, owner API 9 tests) |
| Price change | `TestOwnerPriceChangeNeedsCodeAndAppliesAtOnce`: refused without code, bad price 400, 99 → checkout, invoices and `config.json` use 99.00; redeployed, health 200, price still 149.00 |
| Versions | `TestOwnerVersionsCompareWebsiteAndUpdateCheck`; live: server reads `ScadaBench-Setup-0.1.14.exe` from scadabench.com/download/, no `release.json` yet, so the card says apps are not told |
| RFC vectors | RFC 6238 SHA-1 codes at 59, 1111111109, 1234567890, 2000000000; RFC 7914 PBKDF2-SHA256 |
| Local end to end (server + `wrangler dev` Worker) | through the Worker 200, direct without secret 404; sign-in with password + code; overview, licenses (made a key with a note), purchases (sandbox sale shown as test, excluded from totals), issues, disk, email, activity at 375×812; delete asked for a fresh code in the dialog and removed only after it; no console errors (CSP) |
| `verify.py` | File lengths, Architecture, Documentation, Engineering history, Specifications: PASS |
| Live deploy (`push.ps1`) | health 200, PayPal live and email ready, `/buy` shows the news box, `/unsubscribe` 200 |
| Live gateway (`admin.ps1 -Action gateway`) | Worker `scadabench-owner-gateway` on route `scadabench.com/<owner path>*`; through it `GET /api/session` 200 `{"setup_done":false}`; `license.scadabench.com/<owner path>/` 404; first deploy answered 404 because the piped secret kept a line ending, so the Worker now trims it |

## Limitations

- Sessions are in memory: each deploy or restart signs the owner out.
- The owner's own password and authenticator were set by the owner in a terminal (`admin.ps1 -Action
  reset`); the codes were not read by the assistant, so live sign-in is confirmed only by the owner.
- Cloudflare Access is an owner dashboard step (spec section), not automated.
- News uses the license Mailgun domain until a separate one is verified and set.
- No QR code: the authenticator key is typed (32 letters in groups of four).
- Disk cleanup is limited to old ScadaBench logs because the server is shared.
