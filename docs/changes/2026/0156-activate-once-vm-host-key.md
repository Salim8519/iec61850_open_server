# 0156: Activate once, works forever; VM host key; release 0.1.20

Date: 2026-10-21. Decisions: [ADR-0098](../../decisions/ADR-0098-license-machine-id-v3.md)
(machine ID v3, activate once), [ADR-0099](../../decisions/ADR-0099-license-vm-host-key.md)
(VM host key). Contract: [license binding](../../architecture/license-binding-contract.md).
Spec: [licensing](../../specs/licensing.md) LIC-50 to LIC-54, LIC-70 to LIC-87.

## Why

The owner wants activation without hassle and strong protection without backdoors: one
activation per key for life, offline allowed, no periodic online requirement, VMs on the
licensed PC allowed, a new motherboard needs a new license. A TPM chip binding and a
30-day online lease were built and rejected: the owner's AMD fTPM refused
`ActivateCredential` (TPM_RC_SIZE), chip resets would lock out honest buyers under the
no-reset rule, and leases risked cancelling honest PCs (two Windows users, lost answers,
server outages).

## What changed

- **Machine ID v3** (Go engine and native guard): SMBIOS firmware UUID + baseboard serial
  (MachineGuid only when both are junk); survives a Windows reinstall; v1/v2 removed. The
  frozen Python backend keeps its older ID (Go-engine-only feature, precedent ADR-0091).
- **Activate once**: same key + same machine ID re-issues; any other machine ID is 409;
  the weekly signed verify stays silent and best-effort (never blocks offline) so fraud and
  chargeback keys stop on PCs that go online.
- **VM host key**: an Ed25519 key made at activation, private part DPAPI-protected, public
  part in the certificate (`host_key`); the helper `scadabench_hostkey.exe` (HKCU Run,
  only when the user turns it on) serves proofs only on virtual-switch adapters after
  checking its own machine ID, certificate and cancellation; the VM verifies a signed
  proof over nonce, its machine ID and the host's time (v2, time signed; ±600 s; 24 h;
  rollback anchor); the native guard accepts `vm_proof.json` by the same rules. Firewall
  rule only through the network helper with preview. Activation page accepts the
  `<machine id>.<host key>` activation code.
- **Wording**: "A new computer or a new motherboard needs a new license." on the website,
  buy and activation pages and emails (the earlier TPM wording is gone).
- **Release 0.1.20** also carries the 0.1.19 work merged earlier (point kinds, full view,
  contrast, stop-all, SNMP names) and changes 0152/0153.

Built by agents G (TPM/lease removal) and H (host key) in separate worktrees; merged and
reviewed by the orchestrator, who also sent back one must-fix (the proof time was unsigned).

## Evidence

- Real license server e2e (Go): activate once, reinstall re-issue, other machine 409,
  weekly signed check never blocks offline, cancelled key stops and stays cancelled.
- Host key e2e on this PC: real server, real DPAPI, real helper (test seam on loopback),
  VM licensed in vm-host mode; native `verify-vm` accepts and refuses edited time, 25 h old
  proofs and other machine IDs; cancelled host and stopped helper license nobody.
- Real adapter filter on this PC serves only vEthernet (Default Switch), VMnet1, VMnet8.
- Full merged test run: see the ledger entry of this change.

## Limitations

- No test on a real VM (Hyper-V needs administrator rights on this PC); the firewall rule
  was never applied for real; installer taskkill/uninstall steps only compile-checked.
- A VM switched off past its proof and restarted with its clock set back inside the proof's
  day is not caught (same limit as trials, ADR-0046). Binary patching stays out of scope.
- A skilled user who hand-copies a VM's firmware UUID and board serial defeats the binding.
- Deploy the license server before the app: older servers do not put `host_key` in
  certificates.
