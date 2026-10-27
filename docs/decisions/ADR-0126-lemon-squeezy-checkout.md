# ADR-0126: Lemon Squeezy is the main checkout, PayPal the second

Accepted: 2026-10-28. Owner: "lemon squeezy they approved my store", "the price is 99$ only" and
"Lemon Squeezy is one of two payment gateways, it's the main one. PayPal next". Change
[0200](../changes/2026/0200-lemon-squeezy-checkout.md). Extends ADR-0072 (owner page) and the PayPal
checkout of change 0072.

## Problem

Buyers could pay only through PayPal, with ScadaBench as the seller. A merchant of record takes
cards, Apple Pay and Google Pay, charges and files sales tax and VAT, and sends the receipt.

## Decision

1. **Lemon Squeezy first.** The buy page shows "Pay $99 by card" (Lemon Squeezy overlay) first.
   "or pay with PayPal" and the PayPal buttons follow when PayPal is set up. Either one can be
   switched off by leaving its settings empty.
2. **The license server still issues the key.** Lemon Squeezy's own license keys stay off. A signed
   `order_created` webhook (`POST /api/v1/webhooks/lemonsqueezy`, HMAC-SHA256 `X-Signature`) with
   status `paid` for the configured variant issues exactly one SUBS key, records the sale, alerts the
   owner and emails the key, through the same path as PayPal (`issuePaid`). `order_refunded` with
   status `refunded` revokes the key. Partial refunds keep it.
3. **Key on the page.** The order is stored as `LS-<order UUID>`. The overlay's `Checkout.Success`
   event gives the browser that UUID, and the page asks `GET /api/v1/checkout/lemon-key?order=` until
   the webhook has arrived. Nobody can guess another buyer's UUID. Without the overlay the hosted
   checkout opens and the key arrives by email.
4. **Price (owner: "Admin should control whole price").** The owner-page price is the one price. The
   server stores the Lemon Squeezy API key and makes each buyer's checkout through the API with
   `custom_price`, so card, PayPal, the buy page and scadabench.com change together. Only the variant id
   is checked on the webhook, not the amount, so a price change cannot strand a paid order. Without the
   key the hosted product link (dashboard price) is used.
5. **Test mode.** Test orders issue keys only while `lemon_accept_test_mode` is on
   (`set-lemon.cmd` sets it; `set-lemon.cmd -Live` turns it off).
6. **Secrets.** `set-lemon.cmd` takes the API key (hidden prompt or `LEMON_API_KEY`), finds the product,
   creates or updates the webhook with a fresh 40-character signing secret, and sends both to the
   server over SSH standard input (`lemon_api_key`, `lemon_signing_secret`, never on a command line).

## Compatibility

Without `lemon_*` settings the server and buy page behave exactly as before (PayPal only). The PayPal
fulfilment was moved into the shared `issuePaid` with no change in behaviour. The `/api/v1/product`
`checkout` flag is true when either gateway is ready, and `/api/v1/health` adds `lemon`. The page
CSP adds `https://*.lemonsqueezy.com`.
