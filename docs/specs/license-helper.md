# Spec: Owner license helper (portable, offline, password-locked)

Decision: [ADR-0052](../decisions/ADR-0052-owner-license-helper.md).
Change: [0077](../changes/2026/0077-owner-license-helper.md).
Source: `license-server/helper/` (own Go module; the license server is unchanged).

## Purpose

One portable Windows `.exe` the owner double-clicks: a small window with one password
box and two buttons, **Make license for this PC** and **Remove license from this PC**.
No internet, no scripts, no console. Amended 2026-10-17 from a console menu (owner:
"two buttons, simple"). It is an owner tool and is never shipped to customers.

## Behaviour contract

- **Two states of one program.** As built (`ScadaBench-License-Helper-Setup.exe`) it
  opens first-time setup only (password, type it again, **Create my helper**). Setup
  writes the sealed `ScadaBench-License-Helper.exe` to the Desktop: the same program
  with an encrypted vault appended. The password chosen there never expires.
- **Vault:** holds only the owner Ed25519 signing seed and its key id. Argon2id
  (256 MiB, 4 passes, 4 lanes) derives the AES-256-GCM key from the password; the
  password is at least 8 characters, is read without echo, and is never stored.
- **Owner key, not the server key.** The license server's signing key never leaves the
  server. Setup saves only the owner *public* key to
  `%LOCALAPPDATA%\ScadaBench-deploy\owner-signing-key.txt`; a Studio release accepts
  helper licenses only after that key is added with `configure_release.py --add-key`.
- **Offline:** no networking code is linked. Licenses are `license.cert` files in the
  server's exact format (`license_id` `OWNER-…`), which the weekly online check
  reports as `unknown` and leaves in place.
- **Remove** moves `license.cert`, `license_state.json` and `.time_anchor` into a
  `removed-<time>` folder; nothing is deleted. Unrelated files are untouched.
- **Every click checks the password.** The owner key is decrypted only for that click
  and wiped right after; nothing stays unlocked. A wrong or empty password changes
  nothing. The typed password stays in the box while the window is open, and is never
  saved to disk.
- **Make** writes a lifetime license bound to this PC, but never replaces a license that
  still works here (press **Remove** first). **Remove** moves it aside. **Copy license
  text** puts this PC's `license.cert` text on the clipboard (no password: a license is
  bound to one PC and holds no secret). The top line says the license state in words.
- **Update my helper** (setup window) moves the helper on the Desktop onto the new
  program version and keeps its vault byte for byte: same password, same owner key.
  The old file is kept in `%LOCALAPPDATA%\ScadaBench-deploy\helper-backups`.

| ID | Rule | Enforced by |
| --- | --- | --- |
| LHL-01 | The vault opens only with the right password; an altered vault or a short password is refused; no secret is stored in plain text. | `license-server/helper/vault_test.go::TestHelperVaultOpensOnlyWithThePassword` |
| LHL-02 | The vault rides on the program file and re-sealing replaces it, so the result is one portable file. | `license-server/helper/vault_test.go::TestHelperVaultRidesOnTheProgramAndCanBeReplaced` |
| LHL-03 | The helper's Machine ID is byte-identical to the Studio formula and, on a real PC, to the Studio app's value. | `license-server/helper/hwid_test.go::TestHelperMachineIDMatchesStudioFormula`, `license-server/helper/hwid_test.go::TestHelperMachineIDEqualsPythonOnThisPC` |
| LHL-04 | Helper certificates use the server's canonical payload and encoding; bad Machine IDs are refused. | `license-server/helper/signer_test.go::TestHelperCertificateMatchesServerFormat` |
| LHL-05 | The Studio backend and the native guard accept a helper certificate on its own machine and refuse it on another. | `license-server/helper/signer_test.go::TestHelperCertificateIsAcceptedByStudio` |
| LHL-06 | The helper links no networking code (no `net`, TLS or SSH packages). | `license-server/helper/signer_test.go::TestHelperHasNoNetworkCode` |
| LHL-07 | Removal moves license files aside, never deletes, and leaves unrelated files alone; installing keeps the old certificate aside and the clock anchor in place. | `license-server/helper/local_test.go::TestHelperRemoveMovesLicenseAsideNeverDeletes`, `license-server/helper/local_test.go::TestHelperInstallWritesStudioCertificateAndKeepsTheOldOne` |
| LHL-08 | The real sealed `.exe` window, driven through its password box and two buttons, refuses a wrong password, licenses this PC and removes the license. | `license-server/helper/gui_test.go::TestHelperWindowMakesAndRemovesWithTwoButtons` |
| LHL-09 | Studio keeps a helper license when the weekly online check answers `unknown`. | `studio/tests/unit/test_licensing.py::test_online_check_keeps_owner_helper_license_the_server_does_not_know` |
| LHL-10 | Both buttons need the right password on every click; a wrong or empty password writes and removes nothing; Make then Remove works on this PC. | `license-server/helper/actions_test.go::TestHelperButtonsNeedThePasswordEveryTime`, `license-server/helper/actions_test.go::TestHelperMakeThenRemoveOnThisPC` |
| LHL-11 | Setup refuses a password under 8 characters, two different entries, or replacing an existing helper; it writes a helper that opens with that password, and saves only the public key. | `license-server/helper/actions_test.go::TestHelperSetupCreatesAPortableHelperWithAPermanentPassword` |
| LHL-12 | Make never replaces a license that still works on this PC; Copy gives the exact license text. | `license-server/helper/actions_test.go::TestHelperMakeThenRemoveOnThisPC`, `license-server/helper/gui_test.go::TestHelperWindowMakesAndRemovesWithTwoButtons` |
| LHL-13 | Update keeps the vault (same password and key), swaps in the new program, keeps the old file, and refuses non-helpers. | `license-server/helper/actions_test.go::TestHelperUpdateKeepsThePasswordAndKey` |
