# Spec: Owner page

Decision: [ADR-0072](../decisions/ADR-0072-owner-page.md). Change: [0107](../changes/2026/0107-owner-page.md).

## Purpose

One private web page for the owner, comfortable on a phone, at `https://scadabench.com/<owner path>/`.
The path is chosen once and kept only in the server settings and in
`%LOCALAPPDATA%\ScadaBench-deploy\owner-page.json` on the owner's PC, never in this repository.

## Everyday use

| Task | How |
| --- | --- |
| First setup (once) | `deploy-server.cmd`, then `admin-setup.cmd`; type a password twice, add the shown key to Google Authenticator ("Enter a setup key", time based), write the 10 recovery codes on paper |
| Sign in | open the page, type the password and the 6-digit code |
| Lost the phone | type a recovery code in the code box, then run `reset-admin.cmd` to get a new key and codes |
| Forgot the password / locked out | `reset-admin.cmd` on the PC (needs the SSH deploy key); every browser is signed out |
| Check it | `powershell -File license-server\deploy\admin.ps1 -Action status` |

## Sections

- **Overview**: live sales today / month / all (sandbox excluded, refunds subtracted), keys in use,
  hand-made keys, trials, news subscribers, storage against the 45 MB data limit, latest activity.
- **Licenses**: make a lifetime key (optional email, note, "email the key"), search, copy, email
  again, free for a new PC, cancel (revoke), edit note, delete (hand-made keys only, needs a code).
- **Purchases**: every PayPal payment, refund and chargeback with buyer details.
- **Issues**: problem events from the rotated server logs in plain words (keys are masked in logs),
  search, "show every event".
- **Disk**: whole disk (read-only), the data folder file by file, remove older server logs after a
  preview and a code.
- **Email**: write plain-text news, send a test to the owner address, send to subscribers (needs a
  code), progress; add a person who agreed (consent tick required), remove a person completely.
- **Activity**: the append-only owner record; sign out all browsers.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| OWN-01 | Without the gateway secret every owner address is a plain 404; without a configured path and secret the routes do not exist. | `license-server/internal/api/owner_test.go::TestOwnerPageIsInvisibleWithoutGateway` |
| OWN-02 | Sign-in needs password and code together; the cookie is Secure, HttpOnly, SameSite=Strict and limited to the owner path; changes need the CSRF token and the owner origin. | `license-server/internal/api/owner_test.go::TestOwnerLoginNeedsPasswordAndCodeAndSetsStrictCookie` |
| OWN-03 | Ten wrong tries in an hour lock sign-in for everyone; five from one address ban it; a reset ends every session. | `license-server/internal/api/owner_test.go::TestOwnerLockoutStopsSpreadGuessesAndResetEndsSessions`, `license-server/internal/api/owner_test.go::TestOwnerAddressBanAfterFiveWrongTries` |
| OWN-04 | Codes follow RFC 6238, work once, and recovery codes work once. | `license-server/internal/adminauth/adminauth_test.go::TestTOTPMatchesRFC6238`, `license-server/internal/adminauth/adminauth_test.go::TestVerifyCodeAllowsDriftAndRefusesReplay`, `license-server/internal/adminauth/adminauth_test.go::TestRecoveryCodesWorkOnceAndResetRaisesEpoch` |
| OWN-05 | Delete, disk cleanup and news sending need a fresh code; at most 25 hand-made keys a day. | `license-server/internal/api/owner_actions_test.go::TestOwnerMakesFriendKeyAndConfirmsDelete`, `license-server/internal/api/owner_actions_test.go::TestOwnerDailyKeyCap` |
| OWN-06 | News reaches only people who agreed; unsubscribe is one click (POST) and GET never unsubscribes. | `license-server/internal/api/owner_actions_test.go::TestCheckoutOptInSubscribesAndOneClickUnsubscribe`, `license-server/internal/api/owner_actions_test.go::TestOwnerNewsGoesOnlyToSubscribersWithUnsubscribeLink`, `license-server/internal/store/subscribers_test.go::TestSubscribersNeedConsentAndCanLeave` |
| OWN-07 | Cleanup removes only rotated server logs; the database is never removed. | `license-server/internal/api/owner_actions_test.go::TestOwnerCleanRemovesOnlyOldLogs` |

## Cloudflare Access (recommended extra lock, about 5 minutes, once)

With this, Cloudflare emails a one-time code to the owner's address before anyone even sees the
sign-in page. It is free for one person.

1. Open <https://one.dash.cloudflare.com> and sign in to the ScadaBench Cloudflare account. The first
   time it asks for a team name (any name, e.g. `scadabench`) and a plan: choose **Free**.
2. Go to **Access controls › Applications** (older menus: **Access › Applications**) and press
   **Add an application**, then **Self-hosted**.
3. Application name `ScadaBench owner`. Session duration `24 hours`.
4. Under **Public hostname / Destinations** add: subdomain empty, domain `scadabench.com`, path the
   owner path without the first slash (for example `yourname`).
5. Add a policy: name `Only me`, action **Allow**, rule **Include › Emails** = the owner's address.
6. Login methods: keep **One-time PIN** on. Save the application.
7. Test in a private browser window: the page now asks for the email and sends a code, then shows
   the ScadaBench sign-in. `admin.ps1 -Action status` then reports "behind Cloudflare Access".

To remove it, delete the application; the page keeps its own password and code.

## Limitations

- Sessions live in memory: a server restart or deploy signs the owner out.
- News uses the license Mailgun domain until `marketing_domain` / `marketing_from` are set
  (`license-server-admin set --marketing-domain news.scadabench.com --marketing-from '...'` after
  that domain is verified in Mailgun).
- Whole-disk cleanup is out of scope: the server is shared, so only ScadaBench log files are removed.
