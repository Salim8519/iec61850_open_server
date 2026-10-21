# ADR-0098: One computer for good: hardware machine ID, security-chip binding, online lease (app side)

Accepted: 2026-10-21 (draft by the app-side agent; the license server half is written by its own
agent from the same contract, [license-binding-contract](../architecture/license-binding-contract.md)).
Builds on [ADR-0097](ADR-0097-license-no-backdoors.md). Owner decisions: there are no customers yet,
so nothing is migrated; a license must survive a Windows reinstall; a PC with a real,
maker-certified TPM 2.0 works fully offline for good; a PC without one (or with a virtual TPM, or a
VM) works offline for up to 30 days and quietly re-checks online; a cloned copy gets caught; a new
motherboard or a TPM reset needs a new license; no backdoors. Spec rows
[LIC-50 to LIC-59](../specs/licensing.md). **Go engine only** (the frozen Python backend keeps
its older machine ID and is not a shipped engine; precedent
[ADR-0091](ADR-0091-gateway-own-points.md)).

## Problem

The machine ID v2 mixed the Windows MachineGuid and the system drive's volume serial with the
firmware UUID. Reinstalling Windows changes both, so a customer who reinstalled lost the license
(the old self-service move that covered this is gone, ADR-0097). A copy of `license.cert` plus the
license folder on a second computer with the same ID (a disk clone, a VM clone) worked as long as
the server was never asked.

## Decision

1. **Machine ID v3** (Go `licensing/hwid.go`, native `common/machine_id.cpp`, identical):
   `SHA256_HEX("scadabench-hwid-v3|" + firmware + "|" + board + "|" + extra)`. `firmware` is the
   SMBIOS type 1 UUID as lowercase raw hex, `board` the SMBIOS type 2 (baseboard) serial number,
   trimmed; each is `""` when it is a placeholder (UUID: empty, malformed, one repeated byte or the
   known placeholders of `trial_parts.go`; serial: empty, "Default string", "To be filled by O.E.M.",
   "None", "N/A", "0" in any case, or one repeated character). `extra` is `""`, except when both are
   `""`: then the lowercased MachineGuid (such a PC gets a new ID after a Windows reinstall; a computer
   that reveals nothing at all gets no ID rather than one shared by every such computer). Both
   SMBIOS values come from the same raw firmware table. v1 and v2 IDs are not accepted anywhere in
   Go or native code (`BoundHWID`, `LegacyMachineID`, `ComposeLegacy`, `Options.LegacyHWID` are gone).
2. **Chip binding.** At activation the app offers the TPM 2.0's endorsement key (EK), the maker's
   EK certificate and a new attestation key (AK) created by the chip. The server accepts only a
   certificate that chains to a bundled TPM maker root, sends a credential challenge only the chip
   holding both keys can open (`POST /api/v1/activate`, then `POST /api/v1/activate/tpm`), and issues
   a certificate with the extra signed fields `ak` and `ek`. The AK's storage blob is kept as
   `ak.blob` in the license home. At every evaluation (status cached 60 s, proofs reused 60 s) the
   chip signs a fresh random 32-byte nonce with the AK (a TPM quote over PCR 0) and software checks
   the signature with the certificate's `ak`: no server, no clock, no limit. A copy of the license on
   another computer has no such key; it shows "This license is tied to this computer's security chip,
   which could not confirm it. ..." A chip that worked within 10 minutes and is momentarily busy still
   counts, one that never proved the license never does. Re-activating (same key, same EK, e.g. after a
   Windows reinstall that removed the AK) creates a new AK; the old key is deleted from the chip store.
3. **Lease.** Anything that stops the chip proof (no EK certificate, virtual TPM, TPM 1.2, a chip that
   cannot open the challenge, an older server) is not an error: the app activates again without the
   chip and gets a lease certificate (`lease_days` 30). The server keeps one lease token per key and
   passes a new one on with every valid signed verify answer (`lease_token` inside the signed payload).
   The app sends its token with each verify request. A stale token answers `moved`: this is a copy, and
   the license is cancelled on it for good (the cancelled-ID record of ADR-0097). The app stores the
   token (`<server time>:<token>`, newest copy wins) and the last valid signed answer in the three
   places of the trial clock (dot-files `.lease_token` and `.lease_answer`, HKCU `LeaseToken` and
   `LeaseAnswer`, `license_state.json`). The signed answer, not a plain number, is the proof of when the
   license was last confirmed, so editing a file cannot extend a lease; its server time also guards
   against rolling the clock back. The lease ends 30 days after the later of the certificate's issue time
   and the newest valid answer, with `online_check_required` and the text "Connect this computer to the
   internet once so ScadaBench can confirm the license. Computers with a security chip (TPM) do not need
   this." and the existing "Check online now" button. The background loop checks once at start and on
   each 30-minute tick, but only when the last confirmation is older than 6 hours (or the lease ran
   out), and checks never overlap, because every valid answer replaces the token.
4. **Status.** `GET /license/status` adds `binding` (`"tpm"` or `"lease"`) for such licenses only; it is
   absent for trials and older certificates, so every Python-shaped answer is byte-identical. The UI
   shows "This computer's security chip" or "Confirms online every 30 days".
5. **Native guard.** It checks the same certificate, v3 ID, clock and cancelled IDs as before. It cannot
   do the TPM proof or the lease; the Go backend does both and starts the workers only for a licensed
   computer.

## The new dependency: `github.com/google/go-attestation` v0.6.4

The engine so far was standard library only (ADR-0074). TPM access on Windows needs the Platform Crypto
Provider (credential activation and key creation) and the TPM 2.0 structures (AK parameters, credential
challenge, quotes). Writing that ourselves would be a long, risky copy of the same code, and the license
server needs the same library for the server half of the challenge (`ActivationParameters.Generate`).

- **Why this library:** Apache-2.0, maintained by Google, the reference for TPM 2.0 attestation in Go,
  pure Go on Windows (no cgo; the release build keeps `CGO_ENABLED=0`), works as a standard user.
  The engine now requires `github.com/google/go-attestation` v0.6.4, `github.com/google/go-tpm` v0.9.8
  (TPM structures and the Windows `tbs` package; we import it for the quote check) and
  `golang.org/x/sys` v0.41.0 (already needed for the registry); `go-cmp` and `go-tpm-tools` appear in
  `go.sum` only. Everything is pinned by hash in `studio/engine/go.sum`.
- **Alternatives rejected:** raw TBS commands (Windows blocks `TPM2_ActivateCredential` for standard
  users: `TBS_E_COMMAND_BLOCKED` 0x80280400, tested on this PC); our own NCrypt calls (the same code
  without the review); a user-installed tool (breaks "usable without development tools").
- **What stays ours:** a seam, `licensing/tpm` (`Chip` interface), around it: tests use a software
  `Fake`; `tpm.None` is the chip of tests and the parity build (lease only); only the app composition
  (`wire_platform.go`) passes `tpm.System()`. The AK signature is verified by our own code
  (`tpm/proof.go`, signature plus quote structure plus nonce) without reading PCRs back.
- **Compatibility:** `go 1.24.0` in go.mod (was `1.24`); packaged `ScadaBench.exe` grows with the
  library; nothing in the wire formats of the other engines changes.

## Parity with the frozen Python backend

The Python vectors stay the oracle for everything else. Intentional differences, none of them in a
shipped engine: the machine ID (Go v3 vs Python v2), `binding` in the status, the lease and chip files,
`Options.LegacyHWID` (Go: removed; the one recorded scenario "legacy machine" is skipped in
`TestScenariosMatchPython`), and the v2/v1 native tests (replaced by `test_worker_accepts_only_the_machine_id_v3`).

## Findings on the real chip (this PC: AMD firmware TPM 3.57, Windows 11 build 26300, standard user)

- Works without administrator rights: opening the chip, reading the EK, creating AKs (stored under
  `%LOCALAPPDATA%\Microsoft\Crypto\PCPKSP`), signing quotes, and software verification (a proof takes about 65 ms).
- The Platform Crypto Provider returns no EK certificate to a standard user, although Windows keeps one in
  `HKLM\SYSTEM\CurrentControlSet\Services\TPM\WMI\Endorsement\EKCertStore`. `tpm/ekcert_windows.go` reads
  that store (read-only) and keeps the certificate whose public key equals the chip's EK. Its issuer is
  "PRG-RN" (AMD); the server's bundled roots must cover AMD's intermediate/root for this chip to be accepted.
- `ActivateCredential` through the provider fails with `0x80280095` (TPM_RC_SIZE) for every credential,
  also one built by the chip itself with `TPM2_MakeCredential`, so the failure is in this
  chip/Windows build, not in our blob. The license then falls back to the lease (by design). Whether an
  elevated process or another build behaves differently is not known; no administrator rights were available.

## Risks and limits

- **A lost answer** (the server rotated the token, the app never stored it) makes the next check read
  `moved` and cancels an honest license for good. The app keeps this rare (one check per 6 hours, never
  two at once); the server should also accept the immediately previous token once (see the report).
- An offline PC that deletes every copy of its state and sets its clock back can stretch a lease 30 days at a
  time (same accepted limit as the trial clock). A TPM license cannot be cloned; a lease can be run on two
  computers for at most 30 days before one of them is caught.
- A TPM reset that keeps the EK (Clear) only needs the key entered again; a reset that changes the EK, or
  a new board, needs a new license (website wording unchanged).
- The attestation keys of failed or replaced activations are deleted; a crash between creating and deleting
  leaves a 2 KB file in the provider's folder.
- No EXE was built and nothing deployed. Binary patching of the app stays out of scope (ADR-0046).
