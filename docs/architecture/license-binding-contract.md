# License binding contract (owner decision 2026-10-21, ADR-0098 revised)

"Activate once, works forever." No customers exist yet; no migration.

## What a license is bound to
- Machine ID v3 (unchanged from ADR-0098): `SHA256_HEX("scadabench-hwid-v3|" + firmware +
  "|" + board + "|" + extra)` — SMBIOS type 1 UUID, SMBIOS type 2 serial, MachineGuid only
  when both are junk. Survives a Windows reinstall; a new motherboard needs a new license.
- One activation per key, for life (online in the app, or offline through the activation
  page). Re-activating the same key on the same machine ID re-issues the certificate.
  Any other machine ID: 409 `bound_elsewhere`. No moves, no resets.
- No TPM, no lease, no periodic requirement. The existing weekly verify stays as a
  silent best-effort check (only when online; never blocks an offline PC) so fraud /
  chargeback keys stop on PCs that go online.

## VM host key (VMs on the licensed PC, offline)
Activation: the app creates an Ed25519 host key pair at first activation. The private key
is kept in the license home as `host_key.dpapi`, encrypted with Windows DPAPI (current
user scope), so a copy on another PC/user cannot be decrypted. The public key is sent in
the activation request as `"host_key": "<base64url, no padding, 32 bytes>"` and the server
puts it in the certificate payload as optional field `host_key` (payload still version 1,
keys alphabetical). Offline activation: the app shows one "activation code" =
`<machine id 64 hex>.<host_key base64url>`; the activation page accepts this code (and
still a bare 64-hex machine ID, which gives a certificate without `host_key`).

Host helper `scadabench_hostkey.exe` (Go, studio/engine/cmd/scadabench_hostkey), started
at Windows logon via HKCU\Software\Microsoft\Windows\CurrentVersion\Run only when the
user turns on "Let my virtual machines use this license" (Settings > License). It:
- refuses to serve unless this PC's machine ID equals the certificate's `hwid`, the
  certificate verifies with the release keys and is not cancelled (revoked_ids), and the
  DPAPI key decrypts and matches the certificate's `host_key`;
- listens on TCP port 47862 only on virtual-switch host adapters (Hyper-V "vEthernet",
  VMware "VMnet1/VMnet8", VirtualBox "Host-Only"), never on physical adapters;
- request (one JSON line): `{"nonce": 32 hex, "vm_hwid": 64 hex}`; answer (one JSON
  line): `{"certificate": <host license.cert object>, "signature": base64url Ed25519 over
  the bytes "scadabench-vm-proof-v1|" + nonce + "|" + vm_hwid}`; rate-limited.
The inbound firewall rule for port 47862 on those adapters is added only through the
existing network helper with its preview (journaled, removable), per AGENTS.md.

VM side (inside a VM without its own valid license): find candidate hosts (default
gateway and the `.1` of each local IPv4 subnet), send a fresh nonce, accept only when the
certificate verifies with the release keys, is `lifetime`, not cancelled locally, has a
`host_key`, and the signature verifies for this nonce and this VM's machine ID. Then the
status is licensed with plan "lifetime", mode "vm-host" and the masked host key. Proof is
re-done at every start; the Go backend writes `vm_proof.json` (certificate, nonce,
vm_hwid, signature, time) for the native guard, which accepts it only for its own machine
ID and when `time` is within 24 h. A VM with its own license works as any computer.
VM detection is not needed for security (a physical PC can also use a host proof only if
it sits behind a licensed host on a virtual switch, which is harmless).
