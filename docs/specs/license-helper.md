# Spec: Owner license helper (RETIRED)

Retired 2026-10-21 by `docs/decisions/ADR-0094-license-no-backdoors.md` (owner decision: no
license moves and no backdoors). Original decision: [ADR-0052](../decisions/ADR-0052-owner-license-helper.md);
original change: [0077](../changes/2026/0077-owner-license-helper.md).

## What was removed

The portable, offline, password-locked helper EXE (`license-server/helper/`, its own Go module)
that signed `OWNER-` licenses with a separate owner key is gone: the source, its tests and its
build are deleted from the repository. No new helper builds exist.

- A helper license is a certificate signed outside the server, so it could not be revoked, was
  never recorded on the server and could be made for any computer. That is a backdoor around
  "one license, one computer", so the owner chose to remove it.
- The license server still answers `unknown` for `OWNER-` ids at the weekly check, as before.
  This change does not touch the server answer or the app: certificates the helper already made
  and the owner public key inside released builds are the app change's concern, not this one's.

## What to use instead

- A key for a friend, tester or a buyer who paid another way: the owner page, section Licenses,
  **Make a lifetime key** (spec: [owner page](owner-page.md)), or `license-server-admin issue`.
  These are normal server-recorded keys: they can be revoked and work on one computer only.
- A key for an offline computer: `license-server-admin issue --email E --hwid <Machine ID>`
  prints a `license.cert` bound to that Machine ID for good.
- A customer on an offline computer: `https://license.scadabench.com/activate`.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| LHL-01 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-02 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-03 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-04 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-05 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-06 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-07 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-08 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-09 | Retired: it protected helper licenses at the weekly check; the helper is gone (ADR-0094). | — |
| LHL-10 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-11 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-12 | Retired: the helper no longer exists (ADR-0094). | — |
| LHL-13 | Retired: the helper no longer exists (ADR-0094). | — |
