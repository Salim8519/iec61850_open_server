# ADR-0099: Virtual machines on a licensed PC use its lifetime license offline (VM host key)

Accepted: 2026-10-21 (agent H; contract: [license-binding-contract](../architecture/license-binding-contract.md),
section "VM host key"). Builds on [ADR-0097](ADR-0097-license-no-backdoors.md) (one computer per
key, no backdoors) and [ADR-0098](ADR-0098-license-binding-tpm-lease.md) (hardware machine ID).
Owner decision: a customer who runs ScadaBench inside virtual machines (Hyper-V, VMware, VirtualBox)
on the one PC that holds the lifetime license must not need a second key, and the VM must work with no
internet, for example on a laboratory laptop. Spec rows [LIC-70 to LIC-85](../specs/licensing.md).
**Go engine only**: the frozen Python backend has no such routes and the frontend hides the screens
when it answers 404 (precedent [ADR-0091](ADR-0091-gateway-own-points.md)).

## Problem

A VM has its own firmware identity, so its machine ID differs from its host's, and a license is
bound to exactly one machine ID (ADR-0097). A virtual TPM cannot be proven (ADR-0098), so a VM can
only ever hold a lease. Customers who test in VMs would need one key per VM, and a VM on an isolated
virtual network can never reach the license server.

## Decision

1. **Host key.** At the first activation of an enforcing build the app creates an Ed25519 key pair.
   The 32-byte private seed is kept in the license home as `host_key.dpapi`, encrypted with Windows
   DPAPI (`CryptProtectData`, current user scope, fixed entropy `scadabench-host-key-v1`), so a copy on
   another PC or another Windows user cannot be opened. The public half goes to the server as
   `"host_key": "<base64url, no padding, 32 bytes>"` in the activation request. Builds without a
   protector (tests, the parity build, development) never make or send a key.
2. **Server.** `host_key` is optional and must be exactly one spelling (base64url of 32 bytes, checked by
   re-encoding, because Go's strict decoder ignores line breaks). The certificate payload (still
   version 1, keys alphabetical) gets the optional field `host_key` between `expires_at` and `hwid`,
   and no field at all when none was sent, so every older certificate is byte-identical. Activating again
   on the same computer replaces the key (a Windows reinstall loses the DPAPI key); another computer
   cannot take over a key by offering one; trial keys never carry one.
3. **Offline activation code.** `<machine ID, 64 hex>.<host key>`, shown by the app (`GET
   /license/activation-code`, "Copy activation code") in "Server & offline options". The offline
   activation page accepts the code and still accepts a bare machine ID, which yields a certificate
   without `host_key`.
4. **Host helper `scadabench_hostkey.exe`** (Go, `studio/engine/cmd/scadabench_hostkey`, windowsgui, no
   window). The Settings switch "Let my virtual machines use this license" writes the per-user value
   `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\ScadaBenchVMHost` and starts the helper; turning
   it off removes the value and the helper (which polls the value) ends itself. Starting the app never
   turns sharing on; with sharing on, the app repairs the value after an update or a move and restarts
   a missing helper. One helper per Windows session (mutex `Local\ScadaBenchVMHost`). The helper serves
   only while this PC's own license allows it: the machine ID equals the certificate's `hwid`, the
   certificate verifies with the release keys, is `lifetime`, has a `host_key`, is not cancelled (the
   three places of ADR-0097), the clock is not before the certificate's issue time, and the DPAPI key
   opens and equals the certificate's `host_key`. It re-checks before every answer, so a cancelled
   license stops being served although `license.cert` is still there. Its log is `vm_host.log` (reset
   above 256 KB); it has no inputs from the environment or the command line.
5. **Where it listens.** TCP port **47862**, only on the host side of virtual switches that are up, have an
   IPv4 address and have **no default gateway**: Hyper-V "vEthernet", VMware VMnet1 and VMnet8,
   VirtualBox "Host-Only". Never physical adapters, WSL, or an adapter with a gateway (a Hyper-V
   External switch shares the real network). Listeners follow the adapters that appear and disappear
   (rescan every 10 s). A peer outside the subnet of the adapter it reached gets no answer. Rate limit: 20
   requests per minute per peer address and 60 per minute in total, answered with a plain "try again in a
   minute". Requests are one JSON line of at most 4096 bytes.
6. **Protocol.** Request `{"nonce": 32 hex, "vm_hwid": 64 hex}`; answer one JSON line `{"certificate": <the
   host's license.cert object>, "signature": base64url(Ed25519 over the bytes "scadabench-vm-proof-v1|" +
   nonce + "|" + vm_hwid)}`, or `{"error": "<plain words>"}` that reveals nothing else.
7. **Firewall.** The one inbound allow rule (program-scoped to the helper, TCP 47862, `-RemoteAddress
   LocalSubnet`) is added only through the existing network helper after the usual preview (item kind
   `hostkey_firewall`, fingerprint, "Yes, apply"), journaled and removed by Settings > Network > Clean up
   like every other app-owned rule; a rule that already existed is never owned, and the elevated helper
   refuses any other rule. Routes: `GET/POST /license/host-key/firewall`.
8. **VM side.** A computer first evaluates its own license exactly as before; a VM with a valid license of
   its own never asks anyone. Only when it has none (and the integrity check passed, the build enforces
   and has a host search) a background single-flight search asks the default gateways and the `.1` of every
   local IPv4 subnet (2.5 s each, 30 s before the next search after a failure) with a fresh nonce. It
   accepts an answer only when the certificate verifies with the release keys, is `lifetime`, is not
   cancelled in this VM's own records, has a `host_key`, and the signature verifies for this nonce and
   this VM's machine ID. The status is then `licensed: true`, `plan: "lifetime"`, `mode: "vm-host"`,
   `license_id` of the host's license and `host_key` (masked). The proof lives 24 h, is renewed in the
   background after 6 h, and ends 24 h after the host disappears. The VM never reports the host's license to
   the license server (the server would see one key from two machines).
9. **Native guard.** The Go backend writes `vm_proof.json` (certificate, nonce, vm_hwid, signature, `time`
   in seconds) into the license home. The native workers (`common/vm_proof.cpp`, tested through
   `studio_license_check verify-vm`) accept it only when there is no valid license of their own, `vm_hwid`
   is this machine's ID, `time` is at most 24 h old (a clock up to 300 s ahead is fine), the host
   certificate is genuine, lifetime, names a host key and is not cancelled in any of the three places, and the
   signature verifies. A damaged file never crashes anything and never licenses. An own valid license
   wins whatever the proof says.
10. **Screens (Settings > License).** "Let my virtual machines use this license" with a plain explanation, the
    masked key, the firewall review, and the reason in words when sharing is impossible (a trial, no
    `host_key` because the license was activated before this existed, a different computer, a cancelled
    license). A VM shows "Licensed through your host PC" and its host key row. The activation screen of an
    unlicensed computer keeps asking quietly while a VM may be finding its host.
11. **Release.** `scadabench_hostkey.exe` is built by `build_hostkey_helper.py` (windowsgui, trimmed, **no build
    tags**) into the native folder, listed in `go_release_steps.WORKERS` (laid out, leak-checked, code signed
    with the workers), rebuilt by the development launcher when stale (dev and release run the same
    program), stopped by `stop_all.py`, and the installer ends it before an update and, at uninstall, ends it
    and removes the Run value. It is not a sealed worker (the seal names the programs the engine starts for
    its own use).

## Security reasoning

- **What it protects.** A VM can be licensed only by a host that holds a genuine lifetime certificate bound
  to its own machine ID, with the matching DPAPI key, right now. Copying `license.cert` to another PC does
  nothing (machine ID); copying `host_key.dpapi` does nothing (DPAPI); a stolen certificate cannot answer
  for another VM or another nonce (the signature covers both); a recorded answer is useless for another
  request (fresh nonce per request, machine ID inside the signature).
- **Why the VM needs no VM detection.** A physical PC behind a licensed host on a virtual switch could use a
  host proof too, which is harmless: the host is the license owner's own PC.
- **A cancelled license** stops being answered by the host as soon as the host learns of it (its own online
  check, ADR-0097), so a VM loses it within 24 h at the latest. The VM learns nothing from the server.
- **Known limits (written down, not hidden).** `time` in `vm_proof.json` is not signed (the contract fixes the
  signed bytes as nonce and machine ID only), so someone who edits that file in a VM can keep an old proof alive
  after the host is gone; the 24 h rule is a convenience bound, not a cryptographic one, like the trial clock
  (ADR-0046). Signing `time` into the proof would close this and is a possible later protocol change. A VM
  image cloned with the same firmware identity has the same machine ID and is the same machine to the
  license, like any cloned disk. The port is reachable from every VM on the host's virtual switches, which is
  the point of the feature; anything on those switches can ask, and learns only the host's certificate (which
  holds the license ID and e-mail) and a signature. No admin rights are needed to ask.

## What this does not touch

The key format, machine ID v3, trial rules, the server's verify and activate answers for computers
without a `host_key`, the chip and lease bindings of ADR-0098 (a VM's own lease or chip license is
evaluated first and wins), and the Python backend.

## Findings on this PC (Windows 11 build 26300, standard user, Hyper-V role enabled)

- The adapter list is read with `GetAdaptersAddresses` (no PowerShell). On this PC the helper would serve
  `vEthernet (Default Switch)` 172.25.96.1/20, `VMware Network Adapter VMnet8` 192.168.230.1/24 and
  `VMware Network Adapter VMnet1` (a link-local 169.254.x.x address) and nothing else: the Wi-Fi card, the
  ZeroTier and VPN adapters have a gateway or are not virtual switches (`TestTheRealAdapterListOfThisPC...`).
- Hyper-V is installed and its services run (`vmms`, `vmcompute`, hypervisor present), but a standard user
  cannot list or start VMs (`Get-VM`: "You do not have the required permission", the account is not in
  Hyper-V Administrators and the feature query needs elevation). No VM was created or started and no system
  setting was changed, so the real VM-to-host path (a VM asking its gateway) was **not** run on a real
  hypervisor; it is covered by the loopback end-to-end test with the real helper, real DPAPI and the real
  license server, and by unit tests with fake adapters and a fake network.

## Risks

- A host that sleeps or is off cannot renew a VM's proof; the VM keeps working up to 24 h from the last proof,
  then asks again.
- The firewall rule needs one UAC approval, as the gateway and SNMP rules do.
- A customer who reinstalls Windows on the host loses `host_key.dpapi`: activating again (same key, same
  computer) makes a new host key and the certificate is replaced.
- No EXE was built and nothing deployed (the installer script was only syntax-checked with `ISCC /O-`).
