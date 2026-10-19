# ADR-0046: Machine licensing and the protected Windows release

Accepted: 2026-10-16. Owner request: a solid production line (7-day trial, 1-PC lifetime
license, self-hosted Go license server with PayPal and Mailgun, protected installer) while
development keeps working exactly as before. Blueprint: `PROTECTION_AND_LICENSING_PLAN.md`.

**Problem**
- The packaged app has no license, so it cannot be sold.
- The plan's native bypass (an exe inside `native/build/` or `STUDIO_ENFORCE_LICENSE` not "1")
  would let any user switch licensing off by moving a file or omitting a variable.
- A self-written Ed25519 verifier in two languages is a real forgery risk.
- Certificates had no key id, so a lost or leaked signing key could not be rotated.

**Decision**
- **One certificate contract.** The Go server signs the exact UTF-8 `payload` string
  (alphabetical keys, `kid`, `version: 1`) with Ed25519; `signature` is unpadded base64url.
  Python, C++ and Go verify the raw string, so key order never matters.
- **HWID** = `SHA256_HEX("substation-studio-hwid-v1|" + MachineGuid + "|" + %08X system volume serial)`,
  computed identically by `licensing/hwid.py` and `native/common/license_guard.cpp`.
- **Development is untouched.** The backend enforces only when `sys.frozen`; the native
  workers enforce only when CMake is configured with `-DSTUDIO_ENFORCE_LICENSE=ON`, which
  only `compile_protected.py` does. There is no runtime switch, path rule or variable that
  turns enforcement off in a release build (the plan's runtime bypass is not implemented).
- **Packaged gate.** After session authentication every `/api/v1/*` call except
  `/api/v1/session` and `/api/v1/license/*` answers 402 `license_required` until licensed.
  The UI shows the activation screen; projects stay in the data folder (hard block chosen
  by the owner instead of read-only mode).
- **Verified crypto.** Python uses the RFC 8032 reference verifier (no new wheel), pinned by
  RFC and Go-signed vectors. Native uses vendored, unmodified Monocypher 4.0.2
  `crypto_ed25519_check` (CC0/BSD-2). Windows CNG provides SHA-256.
- **Rotation.** Clients accept a list of public keys and select by `kid`.
  `configure_release.py` rewrites the Python and native key files together.
- **Clock rollback.** `%LOCALAPPDATA%\SubstationStudio\.time_anchor` plus the latest
  `runs.jsonl` time; 300 s tolerance. The native guard also reads the anchor.
- **Moves and revocation.** The server allows a configurable number of self-service moves
  per key per year (default 2). Packaged apps check `/api/v1/verify` at most weekly when
  online; `revoked`/`moved` renames the certificate aside (kept, never deleted). Offline
  PCs keep working (an accepted limitation).
- **Protected release.** Nuitka compiles `studio_app` to one `.pyd`; the UI ships as a
  masked `frontend.pak` served from memory; PyInstaller bundles both with licensed workers;
  a leak check refuses any `studio_app` source/bytecode or loose UI file; Inno Setup builds
  the installer. Packaged data defaults to `%LOCALAPPDATA%\SubstationStudio\data` unless a
  `data` folder exists beside the exe (existing portable users keep their data).
- **Packaged app window.** Only packaged launches open a chromeless Edge/Chrome `--app`
  window with its own profile; closing it stops Studio. Development still never opens
  windows (ADR-0012).

**Addendum (2026-10-16, owner: "make the distributed app anti-reverse", test locally)**
- The protected build compiles a *copy* of `studio_app` (`tools/protected_source.py`); the
  repository source is never edited. The copy carries an integrity seal: SHA-256 of
  `studio_ied/live/probe.exe`. A packaged app with a swapped or patched worker, or with a
  debugger attached, reports "unlicensed" with a reinstall/debugger reason.
- Nuitka runs with `--python-flag=no_docstrings`; asserts are kept because the packaged
  self-test relies on them. The leak check also refuses `.pdb` files.
- `compile_protected.py --sandbox` builds the same protected app trusting only the local
  sandbox server (`http://127.0.0.1:8090`) into `studio/dist-sandbox`; real releases use
  `studio/dist-protected` and the configured key. The installer is per-user by default
  (no administrator prompt) and installs a one-folder app (fast start, no temp extraction).
- Owner tooling: `license-server reset-trial|delete`, `tools/license_admin.py`
  (free keys, instant activation of this PC, remove/re-activate, reset trial; local
  sandbox or `--ssh` to the real server). Removed licenses are moved aside, not deleted.
- Limits stay honest: these raise the cost of reverse engineering and patching; they do not
  make it impossible.

**Addendum (2026-10-19, owner: stop trial cheaters, never block honest users — "smart, not
soft"; change [0106](../changes/2026/0106-trial-hardening.md))**
- **Problem.** Trials were keyed only on the combined machine ID, so changing one
  identifier (or cloning a VM) gave a new trial. Deleting `.time_anchor` and setting the
  clock back stretched a running trial.
- **Rule.** Only clear proof refuses; a suspicion asks for one extra step; errors and
  outages never count against a user. Paid licenses are unchanged.
- **Hardware parts.** `/api/v1/trial` also takes salted part hashes
  (`SHA256_HEX("scadabench-trial-part-v1|" + guid|volume|firmware + "|" + value)`, the HWID
  normalisation). Placeholder firmware UUIDs (empty, one repeated byte such as all 0 / all F,
  `03000200-0400-0500-0006-000700080009` and similar, both byte orders) and volume serials
  0 / FFFFFFFF are never sent or compared. Every usable part (at least two) matching an
  ended trial = refused. Two of three = suspicious. One alone = nothing. Hardware that matches a
  *running* trial continues that trial under the new machine ID. Old clients without parts
  keep the machine-ID rule.
- **Network.** IP never refuses. More than `trial_ip_email_after` (3) new trials from one
  IP within `trial_ip_window_days` (30) = suspicious.
- **Suspicious = confirmed email.** `POST /api/v1/trial/email-code` mails a 6-digit code
  (Mailgun, 15 min, 5 tries, HMAC-bound to machine ID and address). One trial per
  normalised address (lowercase, no `+tag`, Gmail dots removed), stored only as a salted
  hash. A short list of throwaway-mail domains is refused. Mailgun down or not set up →
  no code needed (logged), the address is still one-per-trial.
- **Signed server time.** Trial, activate, transfer and verify answers carry `server_time`
  `{kid, purpose: "server-time", time, version: 1}`, signed with the certificate key; it
  has no `type`/`hwid`, so a stamp and a license can never stand in for each other.
- **Three copies.** The anchor, the newest genuine stamp and the missing-anchor mark live
  in a dot-file, `HKCU\Software\ScadaBench` (REG_SZ) and `license_state.json`. Write
  failures are ignored.
- **Trial clock.** Rollback = the clock more than 48 h behind the latest anchor copy or the
  signed time (lifetime keeps 300 s). Signed time past `expires_at` ends the trial. A quiet
  online check runs daily; failures never stop a trial. Only when every anchor copy and the
  stamp are gone is one online check asked for within 3 days ("Check online now"), never
  an instant block.
- **Unchanged.** Lifetime licenses (no daily check, 300 s rule, offline-friendly), the
  weekly revocation check, development mode. Python and the native guard
  (`license_clock.cpp`) apply identical rules; a parity test runs both on the same files.
- **Logs and overrides.** Every refusal or extra step is logged with its reason and signals,
  never a full machine ID, part hash, key or email. Owner: `license-server reset-trial`
  (also a refused machine's hardware), `reset-trial-parts`, `reset-trial-ip`, `allow-email`;
  `license_admin.py reset-trial [--machine-id]`, `reset-trial-ip`, `allow-email`; `list`
  shows busy networks.
- **Deploy order.** Deploy the server before a client release with this change (an older
  server sends no stamp and has no email step; trials still work, rollback relies on the
  anchors alone).
- **Accepted limits.** An offline PC whose owner deletes every copy, the run log, and resets
  the clock each time can still stretch a trial in 3-day steps; with the mail service down,
  the email step accepts any unique address; binary patching stays out of scope.

**Compatibility**
- Development: no behaviour change; Settings gains a License tab that reads "Development mode".
- Packaged builds now require a license and store data per user by default.
- New dependencies: Go toolchain + `go.etcd.io/bbolt` (server only), vendored Monocypher
  (native), Nuitka 4.3 and Inno Setup 6 (release build machine only).
- libiec61850 is GPL-3.0. Distribution of closed, license-locked binaries requires the
  commercial license, which the owner's team handles separately; nothing here changes that.
