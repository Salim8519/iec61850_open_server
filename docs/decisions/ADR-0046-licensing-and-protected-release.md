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

**Compatibility**
- Development: no behaviour change; Settings gains a License tab that reads "Development mode".
- Packaged builds now require a license and store data per user by default.
- New dependencies: Go toolchain + `go.etcd.io/bbolt` (server only), vendored Monocypher
  (native), Nuitka 4.3 and Inno Setup 6 (release build machine only).
- libiec61850 is GPL-3.0. Distribution of closed, license-locked binaries requires the
  commercial license, which the owner's team handles separately; nothing here changes that.
