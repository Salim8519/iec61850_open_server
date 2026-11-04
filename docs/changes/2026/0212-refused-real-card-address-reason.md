# 0212 Run names the device that already uses a real-card address

Date: 2026-11-03. Layer: Go engine (`labsetup`, `app`). Follows [ADR-0132](../../decisions/ADR-0132-real-card-addresses-follow-the-run.md).

## Problem

The project was on Ethernet with approved addresses (ADR-0132). At Run, the network service
checked the network for each address. Windows found that another device on the cable already
answered at 192.168.236.20–33, so the service took each address off again and recorded why in
`network/lab_journal.jsonl`. The engine ignored that answer, and the preflight then said:

> C04_BCU needs address 192.168.236.20, which is not set up on Ethernet. Open Network > Set up lab network and apply the address plan.

That is wrong: the plan had just been applied. Applying it again cannot help, because the
address belongs to another device. On the owner's bench that device was a second PC
(172.26.1.198) answering at exactly the project's IED addresses.

## Change

- `labsetup.Service.EnsureForRun` now returns an error. `ensureReal` returns the first refused
  address of the IEDs about to start, in project order, with the service's own reason.
- `changeReal` still journals every address, and now also returns the failed ones.
- `Engine.labAddresses` stops the Run with that error before the preflight. Example:
  `C04_BCU cannot start on 192.168.236.20: Another device on “Ethernet” already uses 192.168.236.20; ScadaBench took it off again. Stop the other device that uses this address, or give C04_BCU another address in Network > Set up lab network.`
- The lab card path and the removals are unchanged. A refused real-card removal is still only
  journalled.

## Evidence

- `go test ./internal/labsetup/ ./internal/app/ ./internal/runtime/` pass. The new test
  `TestARefusedRealAddressStopsTheRunWithItsReason` covers a refused and an accepted address.
- Owner PC, 2026-11-03:
  - The journal shows `failed real_address … Another device on “Ethernet” already uses 192.168.236.20` at 14:46.
  - `Get-NetNeighbor` shows one MAC, 9C-69-D3-8B-A6-7A (also 172.26.1.198), answering at 192.168.236.20–33 and 192.168.235.29.

## Limitations

- Not yet run in the installed app. The packaged EXE was not rebuilt (owner rule).
- Python engine untouched. Real-card addresses are Go-only (ADR-0132).
