# 0148: SNMP devices on the network, select-and-delete for points, menus never cut off

Date: 2026-10-21. Owner request with answers to questions; decisions in the
[ADR-0092 amendment](../../decisions/ADR-0092-snmp-devices.md#amendment-2026-10-21-snmp-devices-follow-the-ieds-onto-the-network).

## Change

- **SNMP devices follow the IEDs onto the network.** Settings › Network lists them next to the
  IEDs ("SNMP · Network switch 1 → 192.168.10.51") on the same card, finds free addresses (ARP
  check, never an IED's address), keeps each device's saved address while it is free, accepts
  typed addresses and offers a free alternative for a taken one. The plan adds their aliases and
  one narrow UDP firewall rule for `studio_snmp.exe`; Clean up removes them. Go:
  `netcfg/snmp_plan.go`, `snmp_rules.go`; SNMP service `network.go`; agent twin socket.
- In lab mode a device answers at its lab address and its own 127.x address; the device list,
  header and settings say where it answers. A device without a lab address does not start and
  explains how to get one. Back to "this PC only" moves running devices back by themselves.
- Settings › Network counts SNMP devices ("Put my devices on a network"; "Your 1 SNMP device
  follows the IEDs…"), also when the project has SNMP devices but no simulated IED.
- **Select and delete** in SNMP values and in the gateway point list: Select, tick rows or
  "All N shown" (follows search and filters, all pages), Delete N. Gateway removals still need
  Save point list (Discard changes undoes them); the description column steps aside while
  selecting so the table still fits 1366x768.
- **Menus open above the page** (shared `Menu`): never cut off by a narrow panel, sized to their
  text, kept inside the window (flip up/left near an edge), closed on scroll or resize.
- Test helper `testutil.FreeUDPPort`: SNMP tests no longer pick ports in Windows' reserved UDP
  ranges (they failed about 1 run in 2 with WinError 10013).

## Evidence

- Go: new `netcfg` tests (rows on the IED card, saved address replaced when busy, typed IED
  address refused with an alternative, SNMP-only plans, plans without SNMP unchanged, helper
  accepts the SNMP rule only for the SNMP worker, UDP command) and all existing netcfg parity
  cases pass; `TestDevicesFollowTheLabNetwork` with the real `studio_snmp.exe` on 127.66.31.x
  (answers at lab and own address, moves back); snmp manager/service 5/5 runs; app and
  platformapi pass.
- Windows check (this PC, nothing sent out): a socket on a non-loopback address cannot send to
  127.0.0.1 (WinError 10049), hence the 127.x twin.
- Frontend 262/262 (new `menuPlace`, `pickRows`, `snmpNetwork` tests), `tsc` clean.
- Browser at 1366x768 on a scratch Go server: lab mode with test addresses 127.66.40.10/127.66.40.1
  (device answers at the lab address, list says "On the network", back to this PC moves it);
  Network setup with a canned preview shows the SNMP rows, saved address, UDP 161 sentence and
  "Put 2 SNMP devices on “Ethernet 2”"; gateway Measured filter → All 4 shown → 4 removed,
  15 → 11 rows with the Save/Discard note; menus checked at 1093-1920 px wide, no cut-off text.

## Limitations

- Not run against a real network card: applying a real plan needs Windows approval and changes
  the adapter, which tests must not do; the plan, helper validation and binding were tested
  with fakes and loopback test addresses.
- The trap receiver has no firewall rule of its own yet; v3 users still have full access.
- Python engine: no SNMP rows (it has no SNMP tool); its plans are unchanged.
