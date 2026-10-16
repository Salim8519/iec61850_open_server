# Blueprint: Nuitka Smart-Mode Protection, Standalone Windows Installer & Autonomous Go License Server

Complete, self-contained implementation blueprint for another coding agent to execute end-to-end.

---

## 0. Agent Handover & Non-Negotiable Repository Rules (`AGENTS.md`)

Before modifying any file in this repository, read `docs/llm/START-HERE.md` and `AGENTS.md`:

1. **299-Line Hard Limit**:
   - Every maintained file under `studio/` and `docs/` (plus `AGENTS.md`) MUST be **299 physical lines or fewer** (including comments and blank lines).
   - Verify with `python studio/tools/check_file_lengths.py`.
2. **Domain Purity**:
   - Modules in `studio/backend/studio_app/domain/` MUST NOT import `flask`, `subprocess`, `socket`, `pathlib`, `os`, `api`, `runtime`, `projects`, `networking`, `observability`, or `licensing`.
   - Verify with `python studio/tools/check_architecture.py`.
3. **Dev vs. Production Parity**:
   - Development mode (`start-studio.cmd` / `--dev` / `not getattr(sys, "frozen", False)`) MUST NOT require a license (`licensed: True, mode: "development"`).
   - Packaged production mode (`getattr(sys, "frozen", False) == True`) enforces the 7-day HWID trial or 1-PC lifetime license.
   - **Do NOT build or promote the packaged `.exe` unless the user explicitly asks.** Create and test all source code, build scripts, and unit tests.
4. **Traceability & Verification**:
   - Add an ADR in `docs/decisions/`, a feature spec in `docs/specs/`, and a dated narrative in `docs/changes/2026/`, linked from `docs/INDEX.md`.
   - Append one entry with `python studio/tools/history_log.py append`.
   - Run `python studio/tools/verify.py` (all 5 checks must output `PASS`).

---

## 1. Architecture Overview

```
+-----------------------------------------------------------------------------------+
|                        WINDOWS STANDALONE DESKTOP APP                             |
|                                                                                   |
|  1. Launcher (SubstationStudio.exe)                                               |
|     - Data dir: %LOCALAPPDATA%\SubstationStudio (in packaged mode)                |
|     - Opens chromeless Edge App Window: msedge.exe --app=http://127.0.0.1:PORT    |
|     - Serves UI from encrypted in-memory bundle (frontend.pak -> RAM dict)        |
|                                                                                   |
|  2. Python Backend -> Compiled via Nuitka Smart Mode into single `studio_app.pyd` |
|     - Zero .py or .pyc files for studio_app in distribution                       |
|     - Verifies license.cert (Ed25519 + HWID + clock rollback guard)               |
|                                                                                   |
|  3. Native C++ Workers (studio_ied.exe & studio_live.exe)                         |
|     - Links `studio/native/common/license_guard.cpp`                              |
|     - Compile-time constexpr XOR obfuscated Ed25519 public key & strings          |
|     - Independently verifies license.cert + Win32 HWID + Anti-Debug on start      |
+-----------------------------------------------------------------------------------+
                                        |
                         HTTPS via Cloudflare Tunnel
                                        v
+-----------------------------------------------------------------------------------+
|               LINUX GO LICENSE SERVER (`license-server/` on 127.0.0.1:8090)       |
|                                                                                   |
|  - Single static binary (`CGO_ENABLED=0`), ~10-15 MB RAM, BoltDB (`licenses.db`)  |
|  - Hard-capped rotating JSONL logs: 5 MB x 3 files = 15 MB max disk usage         |
|  - Bot/Abuse Protection:                                                          |
|    * Reads `CF-Connecting-IP` from local cloudflared                              |
|    * Stateless HMAC + SHA-256 Proof-of-Work (PoW) puzzle (~40ms client / 200ns)   |
|    * Bounded 10,000-IP LRU token bucket (~600 KB RAM ceiling) + 30-min auto-ban   |
|  - Licensing & Autonomous Commerce:                                               |
|    * 1-Click 7-Day Trial locked to HWID (`POST /api/v1/trial`)                    |
|    * Strict 1-PC Lifetime License (`POST /api/v1/activate`, admin-only reset)     |
|    * Built-in `/buy` PayPal Checkout Page + Webhook + Mailgun HTTP API emailer    |
|    * 1-Command Lazy Setup (`deploy/setup.sh` / `./license-server configure`)      |
+-----------------------------------------------------------------------------------+
```

---

## 2. Shared Cryptographic & HWID Specification

Both the Windows App (Python + C++) and the Linux Go Server MUST implement this exact contract:

### 2.1 Deterministic Machine Hardware ID (`HWID`)
To ensure Python (`studio_app/licensing/hwid.py`) and C++ (`studio/native/common/license_guard.cpp`) compute the exact same 64-character hex HWID in `<1ms` without slow WMI queries:
1. Read Windows Registry `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography` value `MachineGuid` (using `KEY_WOW64_64KEY`), trimmed and lowercased.
2. Read the system drive root (`C:\` or `%SystemDrive%\`) volume serial number via Win32 `GetVolumeInformationW`, formatted as 8-character uppercase hex (e.g. `A1B2C3D4`).
3. Compute:
   ```
   HWID = SHA256_HEX("substation-studio-hwid-v1|" + machine_guid + "|" + volume_serial_hex)
   ```

### 2.2 License Certificate Format (`license.cert`)
Stored at `%LOCALAPPDATA%\SubstationStudio\license.cert` as JSON:
```json
{
  "payload": "{\"email\":\"user@domain.com\",\"expires_at\":0,\"hwid\":\"<64-hex>\",\"issued_at\":1760000000,\"license_id\":\"SUBS-ABCD-1234-EFGH-5678\",\"max_machines\":1,\"type\":\"lifetime\",\"version\":1}",
  "signature": "<base64url-encoded-64-byte-Ed25519-signature>"
}
```
- `payload` is the exact UTF-8 JSON string signed by the Go server's Ed25519 private key.
- Signing and verifying the raw `payload` string avoids any JSON key-ordering discrepancies across Go, Python, and C++.
- `type`:
  - `"trial"` (`expires_at = issued_at + 7 * 86400`)
  - `"lifetime"` (`expires_at = 0`)

### 2.3 Clock-Rollback Protection (Anti-Trial Tampering)
For `type == "trial"`:
- Current UTC timestamp `now` must satisfy `now >= issued_at - 300` and `now < expires_at`.
- Also verify `now + 300 >= max_observed_timestamp` (checked against `%LOCALAPPDATA%\SubstationStudio\.time_anchor` and the latest run timestamp in `data/logs/runs.jsonl`).
- Update `.time_anchor` whenever `now > anchor`. If the user rolls back the Windows clock, trial validation fails with `"System clock rollback detected"`.

---

## 3. Phase 1: Studio Backend Licensing & Domain Layer

All files must stay under 299 lines:

1. **`studio/backend/studio_app/domain/license_policy.py` (NEW, pure domain)**:
   - Pure functions and dataclasses only (no `os`, `pathlib`, `flask`, etc.):
   - `evaluate_certificate(payload_dict, current_hwid, now_ts, max_seen_ts, is_dev_mode) -> LicenseStatus`:
     - If `is_dev_mode`: returns `LicenseStatus(licensed=True, mode="development", plan="dev", days_remaining=None, reason=None)`.
     - Checks `version == 1`, `hwid == current_hwid`, clock rollback (`now_ts + 300 < max_seen_ts` or `now_ts + 300 < issued_at`), and expiry (`expires_at > 0 and now_ts >= expires_at`).
2. **`studio/backend/studio_app/licensing/__init__.py` (NEW)**
3. **`studio/backend/studio_app/licensing/hwid.py` (NEW)**:
   - Computes the deterministic Windows HWID using `winreg` (`KEY_READ | KEY_WOW64_64KEY`) and `ctypes.windll.kernel32.GetVolumeInformationW`.
4. **`studio/backend/studio_app/licensing/ed25519_verify.py` (NEW)**:
   - Compact Ed25519 signature verifier + XOR-obfuscated 32-byte public key reconstruction so no extra third-party wheel is required.
5. **`studio/backend/studio_app/licensing/client.py` (NEW)**:
   - Reads/writes `client_config.json` in the app data directory (`{"server_url": "https://..."}`).
   - Implements Proof-of-Work solver: fetches `GET {server_url}/api/v1/challenge`, finds `counter` where `SHA256(f"{nonce}:{counter}".encode())` has `difficulty` leading zero bits (~40ms), and calls:
     - `POST {server_url}/api/v1/trial` with `{"hwid": ..., "pow_nonce": ..., "pow_ts": ..., "pow_mac": ..., "pow_counter": ...}`
     - `POST {server_url}/api/v1/activate` with `{"license_key": ..., "hwid": ..., ...}`
     - `POST {server_url}/api/v1/verify`
6. **`studio/backend/studio_app/licensing/service.py` (NEW)**:
   - Loads `license.cert`, tracks `.time_anchor`, evaluates status via `domain.license_policy`, saves activated certificates, and sets `STUDIO_LICENSE_CERT_PATH` environment variable for child C++ workers (`studio_ied.exe` and `studio_live.exe`).
7. **`studio/backend/studio_app/api/license.py` (NEW)**:
   - `GET /api/v1/license/status` — returns `{licensed, mode, plan, hwid, expires_at, days_remaining, server_url, reason, masked_key}`.
   - `POST /api/v1/license/trial` — starts 7-day trial with Go server and saves `license.cert`.
   - `POST /api/v1/license/activate` — activates lifetime key online and saves `license.cert`.
   - `POST /api/v1/license/activate-offline` — validates and saves a pasted `license.cert` JSON string offline.
   - `PUT /api/v1/license/config` — updates `server_url` in `client_config.json`.
8. **`studio/backend/studio_app/api/security.py` (MODIFY)**:
   - In `protect_request()`, after session authentication, if `getattr(sys, "frozen", False)` is True and `request.path.startswith("/api/v1/")` and not `request.path.startswith("/api/v1/license")` and `request.path != "/api/v1/session"`:
     - Check `current_app.extensions["license"].is_licensed()`. If False, return `jsonify(error="License required", code="license_required"), 402`.
9. **`studio/tools/check_architecture.py` (MODIFY)**:
   - Add `"licensing"` to `FORBIDDEN` set so `domain/` can never import `licensing`.

---

## 4. Phase 2: Native C++ Hardware & Signature Lock (`studio_ied.exe` & `studio_live.exe`)

1. **`studio/native/common/obfuscate.hpp` (NEW)**:
   - C++17 `constexpr` XOR string/byte array obfuscator (`OBF_BYTES` / `OBF_STR`) that encrypts string literals and the 32-byte Ed25519 public key at compile time and decrypts on the stack with `volatile` cleanup.
2. **`studio/native/common/ed25519_verify.hpp` & `ed25519_verify.cpp` (NEW)**:
   - Compact, self-contained C++ Ed25519 signature verifier (~180 lines, zero external dependencies).
3. **`studio/native/common/license_guard.hpp` & `license_guard.cpp` (NEW)**:
   - `check_debugger()`: calls `IsDebuggerPresent()` and `CheckRemoteDebuggerPresent(GetCurrentProcess(), &dbg)`.
   - `compute_hwid()`: calls `RegGetValueW(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Microsoft\\Cryptography", L"MachineGuid", RRF_RT_REG_SZ | RRF_SUBKEY_WOW6464KEY, ...)` + `GetVolumeInformationW(L"C:\\", ...)` + Windows `BCryptHash` (`BCRYPT_SHA256_ALGORITHM` — already linked via `bcrypt` in `CMakeLists.txt`).
   - `require_valid_license()`:
     - In development builds (when `STUDIO_ENFORCE_LICENSE` env var is not `"1"` and executable is inside `native/build/`), returns immediately.
     - When packaged (`STUDIO_ENFORCE_LICENSE=1` or executable is in `_internal/native/`):
       1. Throws runtime error if debugger is attached.
       2. Reads certificate file from `STUDIO_LICENSE_CERT_PATH` (or `%LOCALAPPDATA%\SubstationStudio\license.cert`).
       3. Verifies Ed25519 signature over `payload` using the compile-time XOR-obfuscated public key.
       4. Parses `payload` JSON, verifies `hwid == compute_hwid()`, and checks `expires_at == 0 || current_unix_time < expires_at`.
4. **`studio/native/worker/main.cpp` & `studio/native/live/main.cpp` (MODIFY)**:
   - Call `require_valid_license()` before `server.start(message)` in `studio_ied` and before `connect` / `goose_start` / `sv_start` in `studio_live`.
5. **`studio/native/CMakeLists.txt` (MODIFY)**:
   - Add `common/ed25519_verify.cpp` and `common/license_guard.cpp` to `studio_ied` and `studio_live`.

---

## 5. Phase 3: Frontend Activation UI (`LicenseGate.tsx` & Settings)

1. **`studio/frontend/src/api/license.ts` (NEW)**:
   - Typed API calls for `getLicenseStatus()`, `startTrial()`, `activateLicense(key)`, `activateOffline(certJson)`, and `updateLicenseServerUrl(url)`.
2. **`studio/frontend/src/features/license/LicenseGate.tsx` & `license-gate.css` (NEW)**:
   - Displayed automatically when `licenseStatus.licensed === false` (packaged production mode only).
   - Features:
     - **1-Click "Start 7-Day Free Trial"** button.
     - **Lifetime License Key** input (`SUBS-XXXX-XXXX-XXXX-XXXX`) + **Activate Machine** button.
     - **Buy Lifetime License** button (opens `{server_url}/buy?hwid={hwid}`).
     - Expandable **Server & Offline Options** drawer:
       - Copyable **Machine HWID**
       - Editable **License Server URL** input + Save button (dynamically adapts when migrating domains)
       - **Offline Activation**: textarea to paste a signed `license.cert` JSON for air-gapped substation PCs.
3. **`studio/frontend/src/features/settings/LicenseSection.tsx` (NEW)**:
   - Displays current license status (`Development Bypass`, `7-Day Trial (X days left)`, or `Machine Lifetime License`), Machine HWID, editable Server URL, and Upgrade input if on Trial.
4. **`studio/frontend/src/app/App.tsx` (MODIFY)**:
   - Fetch `/api/v1/license/status` after session auth; if `!status.licensed`, render `<LicenseGate />`.

---

## 6. Phase 4: Nuitka Smart Mode, Encrypted Frontend & Standalone Installer

1. **`studio/tools/pack_frontend.py` (NEW)**:
   - Walks `studio/frontend/dist`, compresses each file with `zlib`, encrypts with a stream key embedded in `frontend_assets.py`, and writes a single `frontend.pak` file.
2. **`studio/backend/studio_app/launcher/frontend_assets.py` (NEW)**:
   - If `frontend.pak` exists, decrypts and decompresses all frontend files into a RAM dict `dict[str, tuple[bytes, str]]` at startup (`<15ms`, `~1 MB RAM`) and serves `/` and `/assets/<path>` directly from memory. Falls back to `send_from_directory` in dev mode.
3. **`studio/tools/compile_protected.py` (NEW)**:
   - Automates the protected build pipeline (run only when user requests a build):
     1. Builds frontend (`npm run build`, `sourcemap: false`) and packs `frontend/dist` into `studio/build/frontend.pak`.
     2. Runs **Nuitka Smart Mode**:
        ```
        python -m nuitka --module studio_app --include-package=studio_app --output-dir=studio/build/nuitka --msvc=latest --lto=yes --no-pyi-file
        ```
        Uses `NUITKA_CACHE_DIR=studio/build/nuitka-cache` so initial build takes ~30s and incremental builds take ~5–10s. Produces a single monolithic native `studio_app.<tag>.pyd`.
     3. Runs PyInstaller using `studio/packaging/studio_protected.spec` bundling `studio_app.<tag>.pyd`, `frontend.pak`, `studio_ied.exe`, and `studio_live.exe` with **zero `.py` or `.pyc` files from `studio_app`** and **zero loose `.js`/`.html` files**.
4. **`studio/backend/studio_app/settings.py` & `studio/backend/studio_app/launcher/entrypoint.py` (MODIFY)**:
   - In frozen mode (`getattr(sys, "frozen", False)`), default data directory resolves to `Path(os.environ.get("LOCALAPPDATA", root)) / "SubstationStudio" / "data"` (unless a local `data/` folder exists beside the exe).
   - Launches dedicated chromeless app window (`msedge.exe --app=http://127.0.0.1:{port}/#session={token}`) with fallback to Chrome `--app` or default browser.
5. **`studio/packaging/installer.iss` (NEW)**:
   - Inno Setup script creating `SubstationStudio-Setup.exe` (installs to `{autopf}\Substation Studio`, creates Desktop & Start Menu shortcuts, preserves `%LOCALAPPDATA%\SubstationStudio` user data on uninstall).
6. **`studio/tools/configure_release.py` (NEW)**:
   - Interactive lazy CLI wizard for Windows (`python studio/tools/configure_release.py`): prompts `[current_value]` for Default License Server URL and Ed25519 Public Key, updating both Python and C++ XOR-obfuscated public key constants in one command.

---

## 7. Phase 5: Ultra-Lightweight Go License, PayPal & Mailgun Server (`license-server/`)

Create a self-contained Go module in `license-server/` (keep all `.go` and `.md` files under 299 lines):

### Directory & File Responsibilities
- **`license-server/go.mod`**:
  - `module substation-license-server` (Go 1.22+, dependency: `go.etcd.io/bbolt v1.3.11`).
- **`license-server/cmd/license-server/main.go`**:
  - Subcommands:
    - `serve` — starts the HTTP server (default `127.0.0.1:8090` for Cloudflare Tunnel).
    - `configure` — interactive lazy setup prompt (`[current_value]` -> Enter to keep, or type new value).
    - `keygen` — generates Ed25519 keypair and prints the public key hex.
    - `issue --email <email> [--type lifetime|trial] [--days 7]` — generates a `SUBS-XXXX-XXXX-XXXX-XXXX` key or offline `license.cert`.
    - `revoke --key <SUBS-...>` — revokes a license key.
    - `reset-hwid --key <SUBS-...>` — clears the locked HWID on a key (admin-only machine reset).
    - `list` — lists issued licenses and trial HWID count.
- **`license-server/internal/config/config.go`**:
  - Loads/saves `config.json` (with env var overrides):
    - `listen_addr` (`"127.0.0.1:8090"`), `public_url`, `trust_cloudflare` (`true`)
    - `trial_days` (`7`), `license_price` (`"149.00"`), `license_currency` (`"USD"`), `product_name` (`"Substation Studio Lifetime License"`)
    - `paypal_mode` (`"sandbox"` | `"live"`), `paypal_client_id`, `paypal_client_secret`, `paypal_webhook_id`
    - `mailgun_domain`, `mailgun_api_key`, `mailgun_region` (`"us"` | `"eu"`), `mailgun_from`
    - `admin_token` (auto-generated 32-byte hex), `pow_secret`, `pow_difficulty` (`18`)
    - `log_dir`, `log_max_megabytes` (`5`), `log_max_files` (`3`)
- **`license-server/internal/log/capped.go`**:
  - Thread-safe size-capped rotating JSONL logger: tracks bytes written, rotates `server.log` -> `server.log.1` -> `server.log.2` at 5 MB, deleting older files so total log footprint never exceeds **15 MB**.
- **`license-server/internal/security/pow.go`**:
  - `IssueChallenge(secret, difficulty)` -> returns `{nonce, timestamp, difficulty, mac}`.
  - `VerifySolution(secret, nonce, timestamp, difficulty, mac, counter)` -> checks HMAC and timestamp window (5 min), then verifies `SHA256(nonce + ":" + counter)` has `difficulty` leading zero bits in 1 hash operation (`~200ns`).
- **`license-server/internal/security/ratelimit.go`**:
  - Extracts real client IP (`CF-Connecting-IP` when request comes from loopback `127.0.0.1`/`::1` and `trust_cloudflare` is true).
  - Bounded fixed-size LRU map (max 10,000 IPs ≈ 600 KB max RAM ceiling).
  - Per-IP token bucket (10 req/min general, 5 req/min activation/trial) + automatic 30-minute ban after 8 failed key attempts.
- **`license-server/internal/crypto/sign.go`**:
  - Ed25519 key generation, `SignCertificate(privKey, payload) -> CertBundle`, and formatted key generator (`SUBS-XXXX-XXXX-XXXX-XXXX`).
- **`license-server/internal/store/store.go`**:
  - BoltDB (`bbolt`) buckets:
    - `trials`: `hwid -> {hwid, issued_at, expires_at, ip}` (strictly 1 trial per HWID ever).
    - `licenses`: `key -> {key, email, type, hwid, activated_at, created_at, revoked, order_id}`.
    - `orders`: `paypal_order_id -> {order_id, email, amount, currency, license_key, created_at}`.
- **`license-server/internal/mail/mailgun.go`**:
  - Sends transactional license delivery emails using Go's `net/http` `POST https://api(.eu).mailgun.net/v3/{domain}/messages` with a 10s timeout.
- **`license-server/internal/api/server.go`, `paypal.go`, `checkout.go`**:
  - `GET /api/v1/health`
  - `GET /api/v1/challenge`
  - `POST /api/v1/trial` (PoW verified -> issues 7-day trial cert if HWID never seen)
  - `POST /api/v1/activate` (PoW verified -> locks key to single HWID if unbound or returns cert if same HWID; rejects if bound to another HWID)
  - `POST /api/v1/verify` (checks revocation & returns refreshed cert)
  - `GET /buy` (clean, self-contained HTML/JS checkout page with PayPal buttons, instant license display on capture, and offline activation helper)
  - `POST /api/v1/checkout/create-order` & `POST /api/v1/checkout/capture-order` (PayPal Orders v2 REST API integration; idempotently issues key + triggers Mailgun email)
  - `POST /api/v1/webhooks/paypal` (verifies PayPal webhook signature via PayPal API and ensures license is issued & emailed even if user closed browser tab)
  - `POST /api/v1/admin/*` (`issue`, `revoke`, `reset-hwid`, `list` protected by `subtle.ConstantTimeCompare` Bearer token)
- **`license-server/deploy/setup.sh` & `license-server/deploy/license-server.service`**:
  - 1-command lazy Linux installer script: builds static Go binary (`CGO_ENABLED=0`), runs `./license-server configure` interactively, installs/restarts `systemd` service (`MemoryMax=64M`, `CPUQuota=50%`, `NoNewPrivileges=true`), and prints the Windows sync command.

---

## 8. Final Verification Checklist

- [ ] `python studio/tools/verify.py` passes all 5 checks (`File lengths`, `Architecture`, `Documentation`, `Engineering history`, `Specifications`).
- [ ] `pytest` passes all existing tests plus new unit/integration tests in `studio/tests/unit/test_licensing.py` (dev bypass, 7-day trial, trial expiry, clock rollback detection, lifetime activation, wrong HWID rejection, Ed25519 signature tamper rejection).
- [ ] `go test ./...` inside `license-server/` passes tests for PoW verification, bounded LRU rate limiter + auto-ban, capped log rotation, 1-trial-per-HWID enforcement, strict 1-PC lifetime key lock + admin reset, and PayPal capture idempotency.
- [ ] `npm run check` in `studio/frontend` passes TypeScript validation with zero errors.
