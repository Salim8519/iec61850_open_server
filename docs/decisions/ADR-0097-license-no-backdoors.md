# ADR-0097: No backdoors in licensing (client side)

Accepted: 2026-10-21. Owner request after a read-only security review of the license
client: no license moves at all, no owner re-link or reset, the offline owner helper key
is retired, no refunds after activation, and paid licenses keep working offline. Trial
hardening ([0152](../changes/2026/0152-trial-hardening.md), ADR-0046 addendum
2026-10-19) stays exactly as it is. The license server's half of this decision (refusing
transfers, signing verify answers, deleting the owner helper) is written down in its own
change; this record covers the app, the Go engine, the frozen Python backend and the
native guard. The owner approved changing the frozen Python backend for this security
work, so Python and Go were changed together and the Go test vectors were regenerated
from Python.

**Problem**

- A cancelled (refunded or revoked) certificate was only renamed to
  `license.cert.revoked-<time>`. `revoked_reason` was read only when `license.cert` was
  missing, so renaming the file back licensed the PC again. The native guard never knew.
- The answer of `POST /api/v1/verify` was trusted as plain JSON: a hosts-file entry or a
  proxy could answer `valid` forever (keeping a refunded license alive) or `revoked` for
  everyone (a denial of service).
- A packaged app accepted any `https://` server, or `http://127.0.0.1`/`localhost`, from
  `client_config.json` or `PUT /license/config`, so a user could point it at their own
  server that signs nothing but says `valid`.
- The module self-check was optional: deleting `ScadaBench.exe.sig` silently switched it
  off.
- Two keys were accepted. The second (kid `16901336`) belongs to the offline owner
  helper (ADR-0052): a stolen helper or password would license any PC, offline, forever.
- Licenses could be moved between computers (self-service, twice a year).

**Decision**

1. **Cancelled stays cancelled.** When the server's *signed* verdict is `revoked` (or
   `moved`, which no longer arises but may still be answered for old records), the
   license ID goes into the three places the trial clock already uses: the dot-file
   `.revoked_ids`, the string `RevokedLicenses` under `HKCU\Software\ScadaBench`, and the
   field `revoked_ids` of `license_state.json`. Each copy is the IDs sorted and joined by
   `,`. Evaluation refuses any certificate whose `license_id` is in any copy, with the text
   "This license was cancelled. Contact the seller if this is a mistake.", even when
   `license.cert` is present; the copies that were deleted are restored from the others at
   every evaluation. IDs are never removed by the app, and installing a certificate with a
   remembered ID is refused. The native guard (`license_clock.cpp`) reads the same places
   and refuses the same IDs. The owner tool `license_admin.py remove` leaves the record alone.
2. **No moves.** `Activate(key, move)` and the `transfer` client call are gone. The
   activate route answers `move: true` with `400 moves_disabled` ("Moving a license to
   another computer is not possible. Each license works on one computer only."). The
   status still carries `transfer_available`, always `false`, so older screens read it. The
   wrong-computer text no longer offers a move. The move button and flow are removed from
   the UI. `license_admin.py` can no longer free a key on the server.
3. **Signed server answers.** The client sends `nonce` (32 lowercase hex characters from
   the system random source) with every verify call. The server adds
   `answer: {payload, signature}` signed with the certificate key; the payload is
   `{"hwid","kid","license_id","nonce","purpose":"verify-answer","status","time","version":1}`
   with `status` one of `valid|revoked|moved|unknown`. A verdict counts only if the
   signature verifies with the release key, purpose and version match, and the nonce, the
   license ID and the machine ID equal the request (compared exactly as sent). The
   unsigned top-level `status` is ignored. Anything else behaves exactly like an
   unreachable server: nothing changes and `last_verify` is not updated (the trial's
   "Check online now" shows "could not be verified"). The refreshed certificate and the
   signed `server_time` were already signed and are unchanged.
4. **Only the real server.** An enforcing (packaged) build asks only the address baked
   into it (`https://license.scadabench.com`). `client_config.json` is ignored and
   `PUT /license/config` answers `400 bad_url`. The client itself refuses every other
   address. Tests and the license sandbox use an explicit test-only seam (Go
   `Options.AllowTestServer`, Python `allow_test_server=True`) that no release code sets;
   a test walks the Go sources to prove it. The `-tags parity` build may set it from its
   seams file.
5. **The self-check cannot be switched off.** A packaged build (Go `main.packaged=1`,
   Python frozen) whose `ScadaBench.exe.sig` (or `.pyd.sig`) is missing, empty or
   unreadable refuses to start with a plain message (a message box when double-clicked).
   The release build scripts now stop when `module_signing.seed` is missing instead of
   skipping the signature. Development builds are unchanged.
6. **The owner helper key is retired.** Only the license server's key (kid `768dc76e`)
   is accepted by the Python, Go and native clients, so `OWNER-*` certificates are refused
   everywhere. `configure_release.py` now writes the Go key list as well, so the three
   lists cannot drift.

**Compatibility and limits**

- Deploy the server first. An older server does not sign verify answers, so the weekly
  lifetime check changes nothing and a trial's daily date check reports "could not be
  verified" (trials keep running; only the missing-anchor path needs one online check).
- Builds made before this change still trust the owner key and still accept owner helper
  licenses; the helper itself is deleted on the server side. Paid licenses are unaffected.
- A user who deletes all three copies of the cancelled-ID record and puts the certificate
  back is licensed again until the next successful weekly check. The three places are the
  same ones the trial clock uses, and offline PCs stay an accepted limitation.
- Binary patching of the app is still out of scope (ADR-0046). The native workers still
  accept a missing `STUDIO_MODULE_PATH` (development and tests have no packaged module);
  the packaged backend sets it, and now cannot start without the signature that backs it.
- Legacy (pre-v2) machine bindings (ADR-0051) are unchanged.
