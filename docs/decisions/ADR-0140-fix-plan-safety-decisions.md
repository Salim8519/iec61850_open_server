# ADR-0140: Safety decisions of the 2026-11-19 fix plan

Accepted: 2026-11-19. Owner, this session: "we allow VMs, but never PCs"; "server backup should
be manual"; "uninstall should fix firewall with option"; "the app should just work".
Change: [0220](../changes/2026/0220-fix-plan-phase-a.md).

## Context

A read-only review found behaviour that could lose work, take a PC off its network, operate a
real device after the operator stopped, or share one licence with other physical PCs. Most fixes
are bugs. These change a contract and are recorded here.

## Decision

1. **Licence sharing is for virtual machines only.** Replaces "Why the VM needs no VM detection"
   in [ADR-0099](ADR-0099-license-vm-host-key.md). The host serves only adapters whose
   *description* names a virtual switch (Hyper-V Virtual Ethernet Adapter, VMware VMnet1/8,
   VirtualBox Host-Only); the friendly name can only exclude (WSL). The asking computer uses a
   host proof only when its SMBIOS system or baseboard maker/product names a hypervisor.
2. **Removal follows ownership.** The app removes a real-card address only when its journal
   records that ScadaBench added it, and never on a DHCP card (ADR-0136 "leave nothing on the
   network" applies to what the app added).
3. **Unknown is not failed.** A real-device command that times out is reported as "result
   unknown" and its control worker is killed; a second command to the same reference waits.
   Gateway Stop refuses commands that have not started.
4. **Unreadable journal does not block start.** It is moved to `recovery\`; the last autosave
   opens with a notice.
5. **Payments.** A Lemon order must match the price like a PayPal one; test and sandbox orders
   give trial keys only. Backup stays a manual command and covers every server file.
6. **Uninstall offers firewall clean-up**, default Yes, through the existing journal and the
   network program; only journaled app-owned items are touched.

## Compatibility

- Project format unchanged. A Modbus point table whose kinds overlap in registers, which loaded
  before, now refuses to start with a message naming both points.
- A Modbus master idle for 60 s is disconnected and must reconnect.
- A VMware guest with `SMBIOS.reflectHost = TRUE` cannot use its host's licence.
- New Go-only event `real_command_unknown`; no Python parity golden changed.
- License server: new bbolt bucket `refunded_orders`; new CLI `restore --from FILE`.
