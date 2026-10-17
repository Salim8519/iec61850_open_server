# ADR-0050: Harden the published release against casual reverse engineering

Accepted: 2026-10-17. Owner report: the first published build was reviewed for
reverse-engineering resistance. The compiled module and packed UI are a reasonable
speed bump, but five gaps remained: the installed app still accepted developer mode
(with a fixed session secret), there was no app version anywhere, the debugger check
was minimal, the hardware ID rested on two easily changed values, and nothing bound
the native workers to the exact compiled backend they load. Owner decision: close the
four code gaps (developer-mode lockout, version stamp, stronger anti-debug, stronger
HWID) now; Authenticode signing stays out of scope (a later purchase).

**Decision**

- **Developer mode is refused in the installed app.** `launcher/entrypoint.py` exits
  when `--dev` is combined with `sys.frozen`, so the packaged build can never run the
  reloader or the fixed `studio-dev-secret`/dev token. Development from source
  (`run_dev.py`, not frozen) is unchanged, preserving dev/production parity.

- **Single version stamp.** New `studio_app/version.py` holds `__version__`
  (`0.0.0+dev` in development). The protected build overwrites it in the sealed copy
  from `compile_protected.py --version` (`protected_source.version_source`). It is
  surfaced in `/api/v1/license/status` and `/api/v1/health` as `app_version`, and shown
  as a "Version" row in Settings › License. No network update check or auto-updater.

- **Stronger anti-debug.** `common/integrity.cpp` (`integrity::check_no_debugger`)
  keeps `IsDebuggerPresent`/`CheckRemoteDebuggerPresent` and adds the PEB
  `BeingDebugged` and `NtGlobalFlag` reads. Only signals that Windows sets exclusively
  while a debugger is attached are used — deliberately **no timing heuristics**, which
  could falsely block a legitimate user on a slow or virtual machine. Mirrored lightly
  in `licensing/integrity.py`. The check stays inside `require_valid_license()`.

- **HWID v2 (compatibility break).** `compose_hwid`/`compute_hwid` (native) and
  `hwid.py` (Python) add the SMBIOS system UUID (`GetSystemFirmwareTable 'RSMB'`,
  Type 1) as a third factor and bump the prefix `substation-studio-hwid-v1` → `v2`,
  byte-identical across both languages. The raw 16 UUID bytes are hashed as hex to
  avoid endianness disagreement; an unreadable UUID falls back to an empty field so
  the ID stays deterministic. **This invalidates every already-issued license** — each
  machine must re-activate. Accepted because the product has only just launched. The
  server is unaffected: the client computes the HWID and the server binds whatever it
  is given.

- **Compiled-module signature (Fix 2): implemented.** The native workers verify the
  exact compiled module (`.pyd`) before serving. A dedicated Ed25519 **module key**
  (separate from the license key, so the build needs no license-server round trip) is
  created once by `tools/module_signing_key.py`: the secret seed stays on the build
  host (git-ignored), the public key is compiled into the workers
  (`native/release/module_key.hpp`). `compile_protected.py` signs the SHA-256 of the
  `.pyd` and ships the raw 64-byte signature beside it; `integrity::verify_module`
  reads the module and signature itself and checks them (`crypto_ed25519_check`).

  *Why it is safe to ship now:* enforcement is **opt-in by presence of the signature**.
  `entrypoint._announce_signed_module()` points the workers at the module only when a
  `.sig` is actually next to it, and the workers verify only when the backend provides
  that path. An unsigned build therefore sets nothing and can never lock itself out —
  the failure mode is "no extra protection", never "app won't start". The check lives
  in the one place every worker already passes (`require_valid_license`).

  *Known limitation (recorded intentionally):* the worker trusts the module path the
  backend hands it. That path is the backend's own loaded module (correct by
  construction), but an attacker who can already patch the compiled backend could also
  stop it from pointing the workers at the module. This is a real raise-the-bar speed
  bump, not an unbreakable guarantee — the same honest framing as the anti-debug check.
  Verified here by a signer/verifier round-trip (valid passes, tamper and wrong key
  fail); the packaged end-to-end run is confirmed on the next authorized EXE build.

**Compatibility**

- Development behaviour is unchanged except that a frozen `--dev` is refused (frozen is
  never set from source). Existing licenses stop matching after HWID v2 and need
  re-activation. No packaged EXE/installer was built for this change.

**Evidence**: see change record [0075](../changes/2026/0075-release-hardening.md).
