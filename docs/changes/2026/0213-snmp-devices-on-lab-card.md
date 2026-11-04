# 0213 SNMP devices answer on the ScadaBench Lab card

Date: 2026-11-03. Layer: Go engine (`snmp/service`, `app`). Uses the own-address rules of
[ADR-0125](../../decisions/ADR-0125-gateway-own-lab-address.md) and the follow-the-run rule of [ADR-0127](../../decisions/ADR-0127-lab-addresses-follow-the-run.md). No new dependency or contract.

## Problem

With the project on the ScadaBench Lab card, every SNMP device with a lab address failed to start:
`SESW-A: 192.168.235.200 is not an address of this PC.` The Lab card only put IED addresses on
itself at Run (ADR-0127). SNMP lab addresses got onto a card only through Network setup's plan
for a real card (ADR-0092 amendment). So a concentrator on this PC could read the IEDs but none
of the switches or GPS clocks.

## Change

- New file `snmp/service/labcard.go`:
  - `SetLabCard(LabCard{Ensure, Release})` joins the SNMP service to the lab card.
  - `Start` asks `Ensure(labIP, "SNMP · <name>")` for each device before its agent binds. A
    refused address fails only that device, with the card's own reason.
  - `CardAddresses()` lists the lab addresses of running devices.
  - `Stop` calls `Release`.
- `app/wire_snmp_network.go`: `Ensure` is `Lab.EnsureOwn`, and it acts only when the project uses
  the installed Lab card. On a real card the plan's aliases are unchanged.
- `app/wire_lab.go`:
  - `Reserved()` also returns the running devices' addresses. Release after an IED Stop keeps
    them, and the Lab card page treats them as used.
  - `SetLabCard` is wired after the lab service is built.

## Evidence

- `go test ./internal/snmp/... ./internal/labsetup/` pass. The new test
  `TestDevicesPutTheirAddressOnTheLabCard` covers: asked once with its owner name, kept while
  running, released on Stop, and a refused address failing only that device.
  `TestDevicesFollowTheLabNetwork` still passes.
- Owner PC, dev engine (`studio/build/engine/studio_server.exe`) on the installed app's data and
  native workers, Al Kamil 400 kV on the Lab card:
  - 14/14 IEDs running.
  - 24/24 SNMP devices running at their cfg addresses (192.168.235.200–209, .3, 236.3,
    172.23.16.107/109). UDP 161 is open on each.
  - The app's reader made an SNMPv3 walk of SESW-A at 192.168.235.200 with the cfg user and got
    52 values.

## Limitations

- The concentrator was not seen polling SNMP. Its pollers had started before the addresses
  existed, and the concentrator then closed (not by ScadaBench).
- Not in the installed 0.1.32. The EXE was not rebuilt (owner rule). Python engine untouched.
