# ScadaBench license server

One small Go program (about 10-15 MB RAM) that gives out 7-day trials, sells lifetime
keys through PayPal, emails them with Mailgun and activates ScadaBench on one PC per key.
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
| Free a key for a new PC | `sudo license-server-admin reset-hwid --key SUBS-...` |
| Cancel a key | `sudo license-server-admin revoke --key SUBS-... --reason refund` |
| Re-send a license file | `sudo license-server-admin certificate --key SUBS-...` |
| Back up everything | `sudo license-server-admin backup --out /root/license-backup.tar.gz` |
| Change settings | `sudo bash deploy/setup.sh --configure` |
| Update after new code | `sudo bash deploy/setup.sh` (or `push.ps1` again) |
| Live output | `journalctl -u license-server -f` |

## Owner license helper (Windows, offline)

`helper/` builds `ScadaBench-License-Helper-Setup.exe`. Run it once on your PC: it makes
a separate owner signing key, locks it with your password and writes the portable
`ScadaBench-License-Helper.exe` to the Desktop. That one file opens a window with a
password box and two buttons (make / remove the license on this PC), with no internet. This server's
`signing.key` is never copied. See [the spec](../docs/specs/license-helper.md).

**Back up `signing.key`** (included in `backup`). Without it you cannot issue keys that
existing ScadaBench installs accept. If it ever leaks: `keygen --force`, then run
`configure_release.py --add-key <new key>` so new releases accept both keys.

## Endpoints

`GET /api/v1/health`, `GET /api/v1/challenge`, `POST /api/v1/trial|activate|transfer|verify`,
`POST /api/v1/checkout/create-order|capture-order`, `POST /api/v1/webhooks/paypal`,
`POST /api/v1/admin/{issue,revoke,reset-hwid,list,certificate,backup}` (local + token only),
`GET /buy`, `GET /activate`.

## Development

`go test ./...` runs all tests. `go run ./cmd/license-server serve --data ./data` runs a
local copy (PayPal and Mailgun stay off until configured). Settings live in
`<data>/config.json`; `LICENSE_*`, `PAYPAL_*` and `MAILGUN_*` environment variables
override them without being written to the file.
