# ScadaBench license server

One small Go program (about 10-15 MB RAM) that gives out 7-day trials, sells lifetime
keys through PayPal, emails them with Mailgun and activates ScadaBench on one PC per key.
A key works on one computer only: it is never moved, freed or re-bound (no self-service move,
no owner reset). A new computer needs a new license.
It listens on `127.0.0.1:8090`; Cloudflare Tunnel publishes it, so no port is opened.

## Lazy install (once)

From Windows, in this repository:

```powershell
powershell -ExecutionPolicy Bypass -File license-server\deploy\push.ps1 -Server you@your-server
```

Or on the Linux server itself: copy this folder and run `sudo bash deploy/setup.sh`.

The script builds the server (it downloads Go if missing), asks a few questions
(press Enter to keep a value), installs a hardened systemd service and prints one line
to run on your Windows build PC:

```powershell
python studio\tools\configure_release.py --server-url https://license.example.com --public-key <64 hex>
```

Then publish it with Cloudflare Tunnel (`cloudflared tunnel ...`, ingress to
`http://127.0.0.1:8090`). In PayPal add the webhook `https://<your address>/api/v1/webhooks/paypal`
for: Checkout order approved, Payment capture completed, refunded, reversed.

## Everyday commands (on the server)

| Task | Command |
| --- | --- |
| See keys and trial count | `sudo license-server-admin list` |
| Make a key by hand | `sudo license-server-admin issue --email buyer@example.com` |
| Key for an offline PC | `sudo license-server-admin issue --email b@x.com --hwid <Machine ID>` (prints license.cert) |
| Cancel a key | `sudo license-server-admin revoke --key SUBS-... --reason refund` |
| Re-send a license file | `sudo license-server-admin certificate --key SUBS-...` (only for the computer the key is already bound to; it never binds or changes a computer) |
| Let a customer's PC start a trial again | `sudo license-server-admin reset-trial --hwid <Machine ID>` (also forgets the hardware that blocked it) |
| Stop asking one network for an email | `sudo license-server-admin reset-trial-ip --ip 203.0.113.7` |
| Let an email address confirm one more trial | `sudo license-server-admin allow-email --trial-email name@company.com` |
| Trial rules | more than `trial_ip_email_after` (3) new trials from one IP in `trial_ip_window_days` (30) ask for an email; Mailgun sends the code (if mail is down, no code is needed) |
| Back up everything | `sudo license-server-admin backup --out /root/license-backup.tar.gz` |
| Change settings | `sudo bash deploy/setup.sh --configure` |
| Update after new code | `sudo bash deploy/setup.sh` (or `push.ps1` again) |
| Live output | `journalctl -u license-server -f` |

## Keys for friends and testers

The owner page (section Licenses, **Make a lifetime key**) and `license-server-admin issue` make
normal server-recorded keys: they can be revoked and work on one computer only. The offline
owner license helper was retired; see [the retired spec](../docs/specs/license-helper.md).

## Signing key

**Back up `signing.key`** (included in `backup`). Without it you cannot issue keys that
existing ScadaBench installs accept. If it ever leaks: `keygen --force`, then run
`configure_release.py --add-key <new key>` so new releases accept both keys.

## Endpoints

`GET /api/v1/health`, `GET /api/v1/challenge`, `POST /api/v1/trial|trial/email-code|activate|verify`,
`POST /api/v1/transfer` (retired: always 410 `moves_disabled`, kept so old 0.1.x apps read a clear text),
`POST /api/v1/checkout/create-order|capture-order`, `POST /api/v1/webhooks/paypal`,
`POST /api/v1/admin/{issue,revoke,reset-trial,reset-trial-parts,reset-trial-ip,allow-email,list,certificate,backup}`
(local + token only),
`GET /buy`, `GET /activate`.

`POST /api/v1/verify` also takes an optional `nonce` (32 lowercase hex) and answers with a signed
`answer` next to the old fields: `{payload, signature}` signed with the certificate key; payload
keys `hwid, kid, license_id, nonce, purpose ("verify-answer"), status, time, version`.

## Development

`go test ./...` runs all tests. `go run ./cmd/license-server serve --data ./data` runs a
local copy (PayPal and Mailgun stay off until configured). Settings live in
`<data>/config.json`; `LICENSE_*`, `PAYPAL_*` and `MAILGUN_*` environment variables
override them without being written to the file.
