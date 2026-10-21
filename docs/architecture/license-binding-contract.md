# License binding contract (phases 2 and 3, ADR-0098)

Shared by the app (Go engine) and the license server. No customers exist yet, so there is
no migration from older machine IDs: v1 and v2 are removed.

## Machine ID v3 (phase 2)
`hwid = SHA256_HEX("scadabench-hwid-v3|" + firmware + "|" + board + "|" + extra)` where
- `firmware` = SMBIOS type 1 UUID as today's lowercase raw hex, or "" when junk
  (placeholders/all-same bytes, same rules as trial_parts.go).
- `board` = SMBIOS type 2 (baseboard) serial number, trimmed; "" when junk
  ("", "Default string", "To be filled by O.E.M.", "None", "N/A", "0", all one repeated
  character, case-insensitive).
- `extra` = "" normally. Only when BOTH firmware and board are "" it is the lowercased
  MachineGuid (then a Windows reinstall changes the ID; documented).
Same formula in Go (`studio/engine/internal/licensing`) and the native guard
(`studio/native/common`). The Python backend is frozen and is not a shipped engine: it
keeps working but these phase 2-4 features are "Go engine only" (precedent ADR-0091).

## TPM binding (phase 3), library `github.com/google/go-attestation` on both sides
Activation (`POST /api/v1/activate`, body as today plus optional `tpm`):
`"tpm": {"ek_cert": b64 DER or "", "ek_public": b64 PKIX DER, "ak": {"public": b64,
"create_data": b64, "create_attestation": b64, "create_signature": b64}}` (go-attestation
`AttestationParameters`, TPM 2.0 only).
- Server accepts the TPM only when `ek_cert` chains to a bundled TPM vendor root
  (license-server/internal/tpmroots) and its public key equals `ek_public`; then it answers
  200 `{"challenge": {"session": "<32 hex>", "credential": b64, "secret": b64}}` from
  `attest.ActivationParameters.Generate()` (session kept in memory 5 minutes, single use).
- App runs `ak.ActivateCredential` and calls `POST /api/v1/activate/tpm`
  `{"session": ..., "secret": b64}`; on match the server binds the key to
  `ek = SHA256_HEX(ek_public)` and `hwid`, and answers `{"certificate": ...,
  "server_time": ...}`.
- A TPM that cannot be proven (no/unknown EK cert, virtual TPM, TPM 1.2, any error) is
  NOT an error: the app activates without `tpm` and gets a lease license (below).
Certificate payload (still version 1, keys alphabetical as today) gains optional fields:
`ak` (b64 AK public, TPM licenses only), `ek` (hex, TPM only), `lease_days` (30, lease
licenses only). Old fields unchanged.
Offline proof at every evaluation for TPM licenses: the app loads its AK (blob kept in the
license home as `ak.blob`), signs a fresh random 32-byte nonce, verifies the signature
with `ak` from the signed certificate. Failure -> unlicensed "This license is tied to this
computer's security chip, which could not confirm it. ..." Cached like other status.
Re-activation: same key + same `ek` -> new AK allowed (TPM was cleared); same key and
neither `ek` nor (lease) `hwid` match -> 409 `bound_elsewhere`.

## Lease licenses (no provable TPM)
- Signed verify answer payload gains `lease_token` (keys alphabetical: hwid, kid,
  lease_token, license_id, nonce, purpose, status, time, version); "" for TPM/trial keys.
- Verify request adds `"lease_token"` (the one the app holds; "" first time). Server keeps
  the current token per key. Matching (or first) token -> status valid + NEW random
  token (32 hex). A stale token -> status `moved`, token not rotated (a copy elsewhere).
- Activation answer of a lease license also carries `lease_token`.
- App keeps the token and the time of the last valid signed answer in the three clock
  places (file, HKCU, license_state.json). It checks quietly in the background every
  start and every 30 minutes when online. If now > last_valid + lease_days*86400 ->
  unlicensed with online_check_required and the text "Connect this computer to the
  internet once so ScadaBench can confirm the license. Computers with a security chip
  (TPM) do not need this." Server time (signed) guards against clock rollback.
- A `moved` verdict cancels the license on that copy (cancelled stays cancelled).

## Trials
Unchanged rules, but sent with the v3 hwid.
