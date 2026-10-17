# 0077: Owner license helper — portable, offline, password-locked EXE

Date: 2026-10-17. Decision: [ADR-0052](../../decisions/ADR-0052-owner-license-helper.md).
Spec: [license helper](../../specs/license-helper.md).

## Owner report

- Wanted: one portable `.exe` that makes and removes a license on a PC, asks for a
  password, no scripts, works offline, "ultra secured". The license server is not edited.

## Change

- New `license-server/helper/` Go module (`substation-license-helper`):
  `main.go` (setup vs. sealed), `setup.go` (owner key, password, seal to Desktop),
  `vault.go` (Argon2id + AES-GCM vault appended to the exe), `signer.go` (certificates in
  the server's format), `menu.go` (owner menu, 10-minute idle lock), `local.go` (license
  folder: install, move aside), `hwid.go` + `hwid_windows.go` (Machine ID, Studio v2
  formula), `console.go` + `console_windows.go` (prompts, hidden password),
  `desktop_windows.go` (Desktop folder), `validate.go` (Machine ID input).
- Tests: `vault_test.go`, `hwid_test.go`, `signer_test.go`, `local_test.go`, `e2e_test.go`.
- `studio/tools/check_specs.py`: recognises Go test functions.
- `docs/specs/license-helper.md` (LHL-01…08) and indexes.

## Menu (sealed helper)

1 license this PC forever · 2 license this PC for N days · 3 license file for another PC
(paste its Machine ID; written beside the helper) · 4 remove the license from this PC
(files kept in `removed-<time>`) · 5 show this PC's Machine ID · 0 exit.

## Evidence / limitations

- `go vet` clean; 10 Go tests pass, including the end-to-end run of a real sealed `.exe`
  (3 wrong passwords refused; licensed this PC with a Studio-readable lifetime cert bound
  to this PC; 5-day file for another PC; removal kept files aside) with `LOCALAPPDATA`
  redirected, so the owner's real license was not touched.
- The Studio Python verifier and the built native `studio_license_check verify` accept a
  helper certificate on its machine and refuse it on another.
- `go list -deps`: no `net`, TLS or SSH package; third-party code only `x/crypto/argon2`
  (+ `blake2b`, `x/sys/cpu`).
- Machine ID from the built exe equals `hwid.machine_id()` on the owner PC.
- Argon2id unlock measured at ≈0.31 s and 256 MiB (16-thread owner PC).
- Built `license-server/bin/ScadaBench-License-Helper-Setup.exe` (3.0 MB, git-ignored).
- **Pending:** the owner runs setup (chooses the password); then the owner public key is
  added with `configure_release.py --add-key` and native rebuilt. Until a build carries
  that key, installed ScadaBench copies (0.1.0–0.1.2) refuse helper licenses. No
  ScadaBench EXE/installer built.

## Amendment (2026-10-17)

- Owner request: minimum helper password lowered from 12 to 8 characters
  (`minPasswordLength`, setup wording, `vault_test.go`). Go tests pass; setup exe rebuilt.

## Amendment 2: two-button window (2026-10-17)

- Owner request: "two buttons, one remove license from machine, another for making one,
  simple; the password I enter should be there forever".
- The console menu is replaced by a native Win32 window (standard `syscall` only, no GUI
  library, still no networking code): password box + **Make license for this PC** +
  **Remove license from this PC**; setup is a window too (password, again, create).
  Trial, other-PC file and Machine ID screens were dropped for simplicity.
- Password: never expires; checked on every click; key decrypted per click and wiped.
  It is not remembered on disk (a remembered password would let anyone with the file
  on that PC use it).
- Files: new `actions.go`, `win32_windows.go`, `window_windows.go`, `screens_windows.go`,
  `actions_test.go`, `gui_test.go`; removed `menu.go`, `console*.go`, `e2e_test.go`.
- Evidence: 13 Go tests pass, including the real windowed `.exe` driven through its
  password box and buttons (wrong password refused, Make licenses this PC, Remove moves
  it aside, window closes); screenshots of both windows checked (modern controls, sharp
  at the owner's 150 % scaling). Setup exe rebuilt with `-H=windowsgui` (no console).

## Amendment 3: readable result, Copy, safe Make, Update, owner key trusted (2026-10-17)

- Owner report (screenshot): the result text was cut off; asked for "copy license text".
  Owner had pressed Make several times; each press moved the previous license aside,
  including the bought `SUBS-…` license (kept in `removed-20261017-150446`, not lost).
- Window wider with a taller message area and short messages; new **Copy license text**
  button (`clipboard_windows.go`). **Make** now refuses to replace a license that still
  works on this PC. Setup gained **Update my helper (keeps my password)**.
- The owner's Desktop helper was updated with that button: vault identical byte for byte,
  program = new build, old file in `ScadaBench-deploy\helper-backups`.
- Owner public key added to Studio: `configure_release.py --add-key` (kid `1b1bcdb6`);
  `release_key.py` and `native/release/release_key.hpp` now list 2 keys. The Studio
  verifier accepts the owner's active `OWNER-…` license on this PC. Installed 0.1.x
  copies still refuse it until a new build; native workers pick the key up on that build.
- Evidence: 14 Go tests pass (the window test now only drives the window of the process
  it started: the first run found the owner's own open helper and stopped before acting);
  52 Studio licensing/release/protected tests pass; screenshot after Make is readable.

## Amendment 4: installed app showed "unlicensed" (2026-10-17)

- Owner report: the installed app says the machine is unlicensed with the helper license.
- Cause: (1) installed 0.1.2 trusts only the server key (kid 768dc76e), so no helper
  license can work there before a new build; (2) setup **Create** was pressed again at
  15:15, making a new owner key (kid 16901336) and replacing the Desktop helper, so the
  earlier key (1b1bcdb6) was stale.
- Fix now: the bought license `SUBS-RN46-BY6P-YGS8-6BSS` was restored as active (copied
  back from `removed-20261017-150446`; the helper license moved aside, nothing deleted);
  it passes a server-key-only check like 0.1.2.
- Release keys are now the server key + current owner key 16901336 (1b1bcdb6 dropped; it
  was never in a shipped build). The next build accepts `OWNER-802B2107BC66863E`.
- **Create** now refuses when a helper is already on the Desktop and points to **Update**
  (keeps password and key); test extended. Setup exe rebuilt.
