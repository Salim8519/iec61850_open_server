# 0200: Lemon Squeezy checkout on the license server

Date: 2026-10-28. Decision: [ADR-0126](../../decisions/ADR-0126-lemon-squeezy-checkout.md).

## Change

| File | Why |
| --- | --- |
| `license-server/internal/lemon/webhook.go` | signature check and order event reading |
| `license-server/internal/api/lemon.go` (+ `lemon_test.go`) | webhook, checkout link and key-for-this-order endpoints |
| `license-server/internal/api/fulfil.go`, `checkout.go`, `sales.go` | one key, sale record, owner alert and email path for both gateways |
| `license-server/internal/api/server.go`, `pages.go`, `guard.go` | routes, `lemon` in health, page data, CSP |
| `license-server/internal/api/pages/buy.html`, `lemon.js`, `style.css` | card checkout first, "or pay with PayPal" second |
| `license-server/internal/config/config.go`, `cmd/license-server/set.go` | `lemon_*` settings and `set` flags |
| `license-server/deploy/lemon.ps1`, `set-lemon.cmd` | one command: hidden API key, find product, webhook, server settings, price |

## Lemon Squeezy dashboard (2026-10-28)

- Store currency changed from OMR to USD, so the price is exactly $99. The save said "saved", but a
  reload could not be read back because the browser pane stopped drawing, so it is not verified.
- The product is not made yet. The browser pane stops while the app window is hidden.

## Evidence

- `internal/api` tests pass, including:
  - a paid order gives one key, shown on the page and emailed once, even when the webhook is retried;
  - a refund revokes the key;
  - forged signatures are refused;
  - other products, unpaid orders and test orders are ignored, and a test order is accepted when allowed;
  - the checkout link carries the computer id and the news choice;
  - the buy page shows the card button first and PayPal second.
- `lemon.ps1` parses. It has not been run against Lemon Squeezy yet; it needs the owner's API key.

## Limitations

- Not deployed. The live license server and buy page are unchanged until the owner approves a deploy.
- The overlay's `Checkout.Success` payload was not seen live. The page searches it for the order
  `identifier`; if it is missing, the buyer is told the key comes by email.
- The website still shows the server price. It becomes $99 when `set-lemon.cmd` copies the Lemon
  Squeezy price, or with `set-price.cmd 99`.
