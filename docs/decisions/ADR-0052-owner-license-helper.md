# ADR-0052: Portable, offline, password-locked owner license helper

Accepted: 2026-10-17. Owner request: "a helper exe that makes the license and removes
the license from the machine, password encrypted, asks for the password, quick
portable exe". Follow-ups: "we avoid scripts", "make sure the tool works offline,
and ultra secured, portable", "no need to edit go" (the license server stays as is).

**Need**

`studio/tools/license_admin.py` does this today, but it is a Python script, needs
the source checkout, and reaches the server over SSH with the deploy key. The owner
wants one file that runs on any test PC, without internet.

**Options considered**

1. *SSH to the server with a restricted key in the vault* (first draft): needs
   internet, so it fails the offline requirement. Dropped.
2. *Carry the server's signing key in the vault:* works with every build already
   installed, but a lost helper plus a guessed password would compromise all
   licensing, and retiring that key means re-keying the server and every build.
   Rejected.
3. **Chosen: a separate owner signing key.** Setup creates a new Ed25519 key,
   locks it in the helper, and saves only the public key. Studio accepts it as one
   more release key (`configure_release.py --add-key`; Python and native stay in
   sync through that tool). The server key never leaves the server, and the owner key
   can be retired alone by dropping it from the next build.

**Decision**

- New Go module `license-server/helper/` (separate `go.mod`, so the server binary and
  its dependencies are unchanged). Builds `ScadaBench-License-Helper-Setup.exe`.
- Vault: Argon2id (256 MiB, t=4, p=4, measured ≈0.3 s per guess on the owner PC) +
  AES-256-GCM with the vault magic as associated data, appended to the program after
  the PE image. Password ≥ 8 characters (owner choice, amended from 12), read with console echo off.
- Certificates mirror `internal/certs` exactly (alphabetical payload JSON, Ed25519,
  base64url without padding), `license_id` = `OWNER-` + 16 random hex, `max_machines`
  1, `issued_at` backdated one hour so a target PC whose clock is slightly behind still
  accepts it. The server answers `unknown` for `OWNER-` ids at the weekly check;
  `service.verify_online_if_due` keeps the certificate (no server change).
- Only the standard `syscall` package is used for Windows APIs (KnownDLLs: kernel32,
  shell32, ole32), so no `golang.org/x/sys/windows` and **no networking package is
  linked**; a test enforces this. Sole third-party code: `golang.org/x/crypto/argon2`.
- `studio/tools/check_specs.py` also recognises Go tests (`func TestX(`).

**Compatibility and limits**

- Builds without the owner public key (0.1.0–0.1.2, the current installers) refuse
  helper licenses with "damaged or was changed". They work from the first build made
  after the key is added. No EXE/installer is built by this change.
- A helper license is a full license for its machine until it expires (trial) or the
  owner key is retired; offline certificates cannot be revoked one by one.
- The helper exe is unsigned (Authenticode is still deferred, ADR-0050). Anyone who
  could swap the helper on the owner's PC could capture the password; keep it on
  media you control.
- Security still rests on the password: with the file and a weak password, an
  attacker can guess offline at ≈0.3 s and 256 MiB per try.

**Evidence**: change record [0077](../changes/2026/0077-owner-license-helper.md).

**Amendment (2026-10-17, owner):** the helper is a two-button window (Make / Remove
for this PC) instead of a console menu, built on plain Win32 through `syscall` (no GUI
dependency, no networking code). The password never expires and is checked on every
click; it is deliberately not stored, so the file alone is useless without it.

**Amendment 2 (2026-10-17):** owner key `1b1bcdb6` is now a release key. Helper updates
reuse the existing vault (setup **Update** button), so the owner's password is permanent
across helper versions. Make never replaces a working license; Copy needs no password.
