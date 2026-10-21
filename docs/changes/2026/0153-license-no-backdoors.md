# 0153: Licensing without backdoors, one computer per key, `.scadabench` files

Date: 2026-10-21. Decisions: [ADR-0097](../../decisions/ADR-0097-license-no-backdoors.md)
(licensing), [ADR-0096](../../decisions/ADR-0096-scadabench-project-file-extension.md)
(project files). Spec: [licensing](../../specs/licensing.md) LIC-08, LIC-33 to LIC-38,
LIC-40 to LIC-43; [owner page](../../specs/owner-page.md) OWN-08;
[license helper](../../specs/license-helper.md) (retired).

## Why

A security review found policy holes around the (sound) Ed25519 certificates: a cancelled
certificate came back by renaming the file, the weekly check failed open and trusted any
answer, a packaged app accepted a localhost server, a missing `.sig` silently skipped the
module check, moves let one key run on three PCs, and owner tools could re-link keys or
mint uncancellable licenses. The owner decided: one computer per key for good, no moves,
no resets, no owner backdoors (friend keys stay), no refund after activation, paid licenses
keep working offline, and project files become `.scadabench`.

## What changed

- **App (Go engine, frozen Python by owner approval, native guard)**: cancelled license IDs
  are kept in three places (`.revoked_ids`, HKCU `RevokedLicenses`, `license_state.json`)
  and refused even when `license.cert` is present; server verdicts count only from a signed
  `answer` echoing a fresh 32-hex nonce, license ID and machine ID (otherwise treated as
  unreachable); packaged builds use only their built-in server (test seam
  `AllowTestServer` / `allow_test_server` never set by release code); no move path
  (`move: true` answers `moves_disabled`); packaged start refused without a readable
  `.sig`; owner-helper key `16901336` removed from Go, Python and native key lists; release
  scripts stop without `module_signing.seed`; `license_admin.py` never frees a key.
- **License server**: `/api/v1/transfer` answers 410 `moves_disabled`; move settings removed;
  `reset-hwid` gone from store, admin API, CLI and owner page ("Free for a new PC");
  `certificate` only re-signs for the bound machine; `license-server/helper` deleted; every
  verify answer adds a signed `answer` (`verify-answer` payload); customer pages and emails
  state the one-computer, reinstall, new-hardware/TPM and refund rules.
- **Website**: moves removed; FAQ, pricing, license agreement ("One computer only"),
  privacy, tutorial text; refund only within 7 days and only before activation.
- **Project files**: `.scadabench` for download, chooser and Settings; a picked `.subsim` is
  refused with a plain message (UI and `POST /project/open` 400 `old_project_file`); the
  app's own `autosave.subsim` and `recovery/*.subsim` are renamed in place at startup by
  both engines, never deleted or overwritten, journaled in `logs/project-file-migrations.jsonl`.

Built by three agents in separate worktrees, reviewed and merged by the orchestrator.

## Evidence

- Real Go license server + Python client e2e: 5 passed, including signed answers for the
  request and a revoked key staying dead after the file is renamed back.
- Go engine service against the real new server: trial, "Check online now" through the
  signed answer, licensed.
- Server: build, vet, tests pass (Windows temp-folder cleanup flake on random tests passes
  on rerun); agent live check 39/39 (410 transfer, 409 bound elsewhere, signed answers,
  bad nonce 400, reset-hwid gone).
- Go engine: build, vet, `-tags parity` build, tests pass; antivirus-blocked test binaries
  pass when built under another name. Python license and native enforcement: 151 passed
  (agent: 142 with enforced and dev native builds incl. gateway). Frontend 275 passed,
  `tsc` clean. Project-file tests 13 passed; Go vs Python migration identical on 5 folder
  shapes; agent browser run renamed old data, refused `.subsim`, round-tripped `.scadabench`.
- Key check: removed kid `16901336` is the owner helper key (change 0077); kept
  `768dc76e` is the live server key.

## Limitations

- **Deploy the license server first**: older servers send no signed answer, so the weekly
  check changes nothing and a trial's "Check online now" cannot confirm the date.
- Owner PCs licensed by the retired helper need a normal key.
- A PC that deletes all three cancelled-ID copies works again until its next successful
  online check (same limit as the trial clock); binary patching stays out of scope.
- The real HKCU `RevokedLicenses` value was tested through fakes only; the shared HKCU code
  was proven Go↔Python in 0152.
- Native workers still skip the module check when `STUDIO_MODULE_PATH` is unset (the
  packaged backend always sets it and cannot start without `.sig`).
- The published activation tutorial video still narrates "Move the key here".
- Phase 2 (hardware-only ID), TPM binding and VM host key are not part of this change.
- No EXE built; website and server not deployed.
