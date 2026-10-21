# ADR-0098: One computer for good: hardware machine ID v3, activate once (app side)

Accepted: 2026-10-21. Revised the same day: the first version also tied licenses to the
security chip (TPM 2.0) and added a 30-day online lease; the owner replaced both by the plan
below before anything shipped, so no customer ever held such a license and nothing is migrated.
The contract is [license-binding-contract](../architecture/license-binding-contract.md);
the license server's half is written by its own agent from the same contract. Builds on
[ADR-0097](ADR-0097-license-no-backdoors.md). Spec rows [LIC-50 to LIC-54](../specs/licensing.md).
**Go engine and native guard only**: the frozen Python backend keeps its older machine ID and is
not a shipped engine (precedent [ADR-0091](ADR-0091-gateway-own-points.md)). The VM host key (a
virtual machine on a licensed PC) is a separate record.

## Problem

The machine ID v2 mixed the Windows MachineGuid and the system drive's volume serial with the
firmware UUID. Reinstalling Windows changes both, so a customer who reinstalled lost the license,
and the old self-service move that covered this is gone (ADR-0097).

## Decision

1. **Machine ID v3** (Go `licensing/hwid.go`, native `common/machine_id.cpp`, identical):
   `SHA256_HEX("scadabench-hwid-v3|" + firmware + "|" + board + "|" + extra)`. `firmware` is the
   SMBIOS type 1 UUID as lowercase raw hex, `board` the SMBIOS type 2 (baseboard) serial number,
   trimmed; each is `""` when it is a placeholder (UUID: empty, malformed, one repeated byte or the
   known placeholders of `trial_parts.go`; serial: empty, "Default string", "To be filled by O.E.M.",
   "None", "N/A", "0" in any case, or one repeated character). `extra` is `""`, except when both are
   `""`: then the lowercased MachineGuid (such a PC gets a new ID after a Windows reinstall; a computer
   that reveals nothing at all gets no ID rather than one shared by every such computer). Both SMBIOS
   values come from the same raw firmware table. A Windows reinstall keeps the ID; a new motherboard
   changes it and needs a new license. v1 and v2 IDs are not accepted anywhere in Go or native code
   (`BoundHWID`, `LegacyMachineID`, `ComposeLegacy`, `Options.LegacyHWID` are gone).
2. **Activate once, works forever.** A key is activated one time for life, online in the app
   (`POST /api/v1/activate`) or offline through the activation page. The same key on the same machine
   ID issues the certificate again (a reinstall, a wiped license folder); any other machine ID gets
   409 `bound_elsewhere`. No moves, no resets, no chip, no lease and no periodic requirement: a PC
   that never sees the internet again keeps its license.
3. **Weekly silent verify, unchanged from ADR-0097.** Once a week, and only when the PC is online,
   the app asks the server whether the license is still valid. The answer counts only when it is a
   genuine signed answer for the request just sent (payload keys `hwid`, `kid`, `license_id`,
   `nonce`, `purpose`, `status`, `time`, `version`). An unreachable server, a bad answer or a refused
   connection is ignored (fail open): the check never blocks an offline PC. A refunded or revoked key
   stops where the PC goes online, and cancelled stays cancelled.
4. **Trials use the same machine ID v3** with their rules unchanged (hardware parts, signed server
   time, grace and online check of ADR-0046 and its addenda).
5. **Native guard.** It checks the same certificate, the v3 ID, the clock and the cancelled IDs as
   before; the Go backend starts the workers only for a licensed computer.

## Considered and rejected

- **Security-chip (TPM 2.0) binding**: an attestation key bound to the chip's endorsement key,
  proven offline at every start (the first version of this record, with the dependency
  `github.com/google/go-attestation`). Rejected because (a) on the owner's PC (AMD firmware TPM 3.57,
  Windows 11 build 26300, standard user) `ActivateCredential` fails with `0x80280095`
  (`TPM_RC_SIZE`) for every credential, also one the chip built itself, so the proof cannot be
  established there; Windows also hides the EK certificate from standard users and the registry copy
  has to be read from HKLM; (b) a chip reset, a firmware or BIOS update or "clear TPM" would lock out
  an honest buyer who did nothing wrong, and only support could free them; (c) it adds a third-party
  dependency, a bigger packaged exe and a server-side maker-root registry to a standard-library-only
  engine for protection the weekly verify and the machine ID already give.
- **Online lease**: a certificate valid for 30 days at a time, renewed by a rotating signed token
  (a stale token cancelling the copy). Rejected because (a) two Windows users on one PC, a restored
  backup or one lost answer would look like a clone and cancel an honest license for good; (b) a
  server outage, or a laboratory PC with no internet (common for substation work), would stop paying
  customers, against "paid licenses keep working offline" (ADR-0097); (c) every extra moving part
  fails closed on honest users while a cheat only needs a second copy for a few weeks.

## Parity with the frozen Python backend

The Python vectors stay the oracle for everything else. Intentional differences, none in a shipped
engine: the machine ID (Go v3, Python v2), `Options.LegacyHWID` (Go: removed; the one recorded
scenario "legacy machine" is skipped in `TestScenariosMatchPython`), and the v1/v2 native tests
(replaced by `test_worker_accepts_only_the_machine_id_v3`). The status, the wire formats and every
other behaviour are exactly those of ADR-0097; `go.mod` is back to the standard library only.

## Evidence and limits

- Machine ID: `hwid_test.go` (shared vectors in `studio/tests/fixtures/licensing/hwid_v3.json` from an
  independent Python reference `studio/tests/unit/hwid_v3.py`; the same vectors run the native guard
  through `test_native_license.py`), `test_native_enforcement.py::test_worker_accepts_only_the_machine_id_v3`.
- Activate once and the weekly verify run against the real license-server binary:
  `e2e_server_test.go::TestRealServerActivatesOnceAndKeepsTheSameIDForLife`,
  `TestRealServerWeeklyCheckIsSignedAndNeverBlocksAnOfflineComputer` and
  `TestRealServerCancelledKeyStopsAtTheNextWeeklyCheckAndStaysCancelled`.
- A copy of `license.cert` on a PC with another motherboard is refused; a disk image restored on the
  same hardware (or a virtual machine cloned with the same firmware UUID) cannot be told apart offline.
  The first stays a deliberate limit, the second is what the VM host key record addresses.
- A PC without any firmware identity gets a new ID after a Windows reinstall (its MachineGuid
  changes) and needs a new license; this is rare and accepted.
- No EXE was built and nothing was deployed. Binary patching of the app stays out of scope (ADR-0046).
