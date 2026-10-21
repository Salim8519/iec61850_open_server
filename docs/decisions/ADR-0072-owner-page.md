# ADR-0072: Owner page behind a Cloudflare gateway

Accepted: 2026-10-19. Change: [0107](../changes/2026/0107-owner-page.md). Spec: [owner page](../specs/owner-page.md).
New contract: owner API under the configured owner path (gateway only), public `GET/POST /unsubscribe`,
`news` flag on `POST /api/v1/checkout/create-order`, CLI `admin-setup` / `admin-status`. New Cloudflare
Worker `scadabench-owner-gateway`. Go language version 1.22 → 1.24 (standard-library PBKDF2).

**Problem**

The owner manages keys, sales and problems only over SSH commands and asked for one secure web page,
usable on a phone, at `scadabench.com/<owner path>`, to watch purchases, make keys for friends, trace
issues, free disk space and send news, with two-step sign-in and a way back in when something is lost.

**Decision**

1. **Where it runs.** The Go license server serves the page and its API under `admin_path` (config,
   not code). scadabench.com is static (Pages), so a Worker on route `scadabench.com/<owner path>*`
   forwards to `license.scadabench.com` and adds `X-Admin-Gateway: <secret>` plus the visitor IP. The
   server answers 404 to every owner address without that exact secret (constant-time check), and
   when `admin_path` or a ≥32-character secret is missing the routes do not exist. The route and the
   secret are set at deploy time; neither is in the repository.
2. **Sign-in.** Password (PBKDF2-HMAC-SHA256, 600 000 iterations, 16-byte salt; Go standard library,
   chosen over argon2 because the service is capped at 64 MB) **and** a 6-digit RFC 6238 code
   (SHA-1, 30 s, ±1 step, a used step is refused). Ten one-time recovery codes (hashed) replace the
   code when the phone is lost. One combined check, so a failure never says which part was wrong.
3. **Limits.** Per address: 6 sign-in tries a minute, banned 1 h after 5 failures. Everyone: sign-in
   locked 1 h after 10 failures in an hour (email to the owner). Wrong confirmation codes count too.
   At most 25 hand-made keys per 24 h. Only one password check runs at a time (CPU bound).
4. **Sessions.** Memory only (restart signs out), cookie `__Secure-owner`: random 32 bytes, stored as
   SHA-256, `Secure; HttpOnly; SameSite=Strict; Path=<owner path>`, 30 min idle, 12 h maximum.
   Every change also needs the session's CSRF header and `Origin` equal to `admin_origin`.
5. **Confirmation.** Deleting a key, removing logs and sending news need a fresh authenticator code
   (`X-Confirm-Code`).
6. **Recovery without email.** No email reset: a stolen mailbox must not open the page.
   `reset-admin.cmd` (SSH deploy key) writes a new password, authenticator secret and recovery codes
   and raises the account epoch, which ends every session.
7. **Records.** `admin-audit.jsonl` is append-only and never edited or deleted by the app; sign-ins
   send the owner an email. Owner responses are `no-store` or `private` so Cloudflare never caches
   anything behind the secret; a strict CSP allows only the page's own files.
8. **Disk.** The server is shared with another project, so the page removes only rotated
   `server.log.N` files in the app's log folder, after a preview. The database, keys, settings,
   payment and audit records are never removed. Whole-disk numbers are shown read-only.
9. **News.** Only people who ticked "Email me about new versions" at checkout (carried as an `N` in the
   signed invoice nonce, no extra storage) or whom the owner confirms agreed. A person who unsubscribed
   can be re-added only by their own new checkout consent. Every email has a one-click unsubscribe
   link and RFC 8058 headers; GET on the link only shows a button (mail scanners), POST unsubscribes.
   `marketing_domain` / `marketing_from` let news use its own Mailgun domain.
10. **Cloudflare Access** (owner's email one-time code) in front of the path is recommended and is
    set up by the owner in the dashboard; the page is secure without it.

**Compatibility**

Existing endpoints, keys, certificates and the admin CLI are unchanged. `License` gains an optional
`note`; the database gains a `subscribers` bucket (created on open). Invoice ids may now start
`SB-N`; old ids still verify. The owner page is off until `admin-setup.cmd` has run. Building needs
Go ≥ 1.24 (`setup.sh` fallback updated).
