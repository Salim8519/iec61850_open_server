# 0072: Trustworthy checkout, live price, payment record, sale alerts, hardening

Date: 2026-10-17. The owner's requests:
- make the buy page look legitimate and fully powered by PayPal;
- make the price adjustable at any time;
- check resistance to attacks, bots and outages;
- cap disk, RAM and CPU, with everything under 100 MB;
- record successful payments, and email the owner the buyer's details on every sale.

Follows [0071](0071-license-server-live-release.md).

## Change

- **PayPal order:** `brand_name` "ScadaBench" (the buyer sees ScadaBench in PayPal),
  `NO_SHIPPING` (digital product), `PAY_NOW`. The capture now reads the payer's name, country and
  payer ID.
- **Buy page:**
  - "Pay securely with PayPal" with PayPal's official logo and acceptance mark, both served by
    `paypalobjects.com`;
  - three plain assurances: checkout on PayPal, two emails, encrypted connection;
  - the optional seller line `--seller-name`;
  - a Content-Security-Policy limited to this server and PayPal's documented domains, HSTS and
    a Permissions-Policy;
  - versioned CSS/JS (`?v=<hash>`), so an update never shows a stale stylesheet.
- **Live price:**
  - `set-price.cmd 149` (`deploy/owner.ps1`) changes the server price in seconds;
  - `GET /api/v1/product` (CORS only for scadabench.com) feeds `website/src/scripts/livePrice.ts`,
    which updates every `[data-price]`;
  - `config.NormalizePrice` stores PayPal's two-decimal form ("149" becomes "149.00"); before,
    a price saved as "149" would never have matched a capture;
  - signed invoice IDs (`api/invoice.go`) keep the price a buyer was quoted valid after a price
    change, and still refuse cheap orders forged with the public client ID.
- **Payment record:** `internal/sales` appends one synced JSON line per payment, refund or
  reversal to `sales.jsonl`. It is never rewritten. `license-server payments` and `payments.cmd`
  show it with live totals.
- **Owner alert:** `--notify-email` sends the owner a copy of every sale: buyer name, email,
  country, PayPal payer, order and capture, key, Machine ID and time. Test-mode sales are marked.
- **Hardening:**
  - at most 48 requests in flight; extra requests get an immediate 503;
  - forged PayPal webhooks are dropped before any PayPal API call (required headers plus a
    `paypal.com` certificate host), and the webhook is rate-limited;
  - new free trials pause when the data folder reaches its budget (45 MB by default);
    purchases and checks keep working;
  - systemd: `GOMEMLIMIT=40MiB`, `MemoryHigh=48M`/`MemoryMax=64M`, no swap, `CPUQuota=50%`,
    `TasksMax=64`, `LimitNOFILE=1024`, `Nice=5`; the tunnel is capped at 64 MB and 25% CPU;
  - deploy deletes its upload files from `/tmp`;
  - `status` reports the data and total disk use against the 100 MB target.

## Evidence

- Go: all packages pass. New tests cover:
  - quoted price after a change, and forged or altered invoices;
  - the sale record plus the owner email carrying the buyer details;
  - product CORS and the checkout CSP;
  - the trial pause at the disk budget, the in-flight cap, and the forged-webhook filter;
  - sales append and read.
- Local attack run (8 s each, the production `GOMEMLIMIT`):
  - a buy-page flood from one IP gets 429 after 11 requests;
  - a 50k-IP botnet is served at about 2,000 requests a second;
  - trials without proof of work are refused (400);
  - 1 MB bodies are limited;
  - forged webhooks get 429/400 and no PayPal calls;
  - all 500 slowloris connections are dropped after 5 s;
  - health answers 200 in 3 ms right after;
  - peak working set is 49 MB (Windows).
- Live:
  - the license server uses 2.5 MB of RAM and the tunnel 18 MB, within the limits above
    (`systemctl show`);
  - disk use is 47.4 MB in total, programs included;
  - a burst from one IP through the tunnel is limited after 10 requests; a forged webhook is
    refused; admin answers 403;
  - `/api/v1/product` returns the price with CORS for scadabench.com;
  - `/buy` sends CSP and HSTS;
  - PayPal accepted a live order with the new fields (`201`, order 5MW42689K6898132K, not
    captured);
  - scadabench.com/pricing shows the live price.
- The browser reported no CSP violations while the buttons rendered.

## Limitations

- The card form under the CSP was opened only against a preview with fake credentials. The
  live card path should be watched on the next real purchase.
- Per-IP limits cannot stop a large distributed botnet on their own; Cloudflare's always-on
  DDoS protection in front of the tunnel handles volumetric floods.
- The live price is still the 0.10 test price; the owner restores it with `set-price.cmd 149`.
- The 0.10 order paid before this change is in the database (`list`), not in `sales.jsonl`.
- JSON-LD prices on the website stay at the build-time price until the next site build.
