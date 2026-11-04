# 0214 Leave nothing on the network

Date: 2026-11-04. Decision: [ADR-0136](../../decisions/ADR-0136-leave-nothing-on-the-network.md).
Layers: Go engine (`netcfg`, `labsetup`, `labnet`, `app`, `platformapi`), frontend.

## Problem

On the owner's PC another program set the network service to manual start. After a restart,
Apply on Ethernet silently used the old helper path: 38 addresses plus a firewall rule, which Stop
does not remove. SNMP device addresses on real cards always took that path. Nothing told the user
anything was left behind.

## Change

- **No fallback.** `netcfg/helper_guard.go`: on a real card, while the network service cannot add
  addresses (missing, stopped or older than protocol 3), each new address row is marked
  `helper_off` with a plain hint. The plan is not ready, so no `alias` item can be applied. The
  frontend shows the status "Network helper is off".
- **Repair turns the helper on.** `labnet.pointAt` sets the service back to automatic start when
  another program set it to manual or disabled. The lab page already says "not running. Press
  Repair".
- **SNMP through approvals.** `useApprovals` turns SNMP rows into approvals too. The SNMP service's
  lab-card hook (change 0213) calls `labsetup.EnsureRealOwn` on a real card before a device
  starts. An address already on the card (an older plan's) is used as it is. `releaseReal` keeps
  the addresses of running devices (`Reserved`), so they leave with Stop, exit and the next
  launch.
- **Reopen asks.** `GET /network/leftovers` and `POST /network/leftovers/remove` (Go only) list
  and remove journaled `alias` addresses still on a card. Approvals, firewall rules and the
  network mode are untouched; the full Clean up still covers those. `LeftoverNotice` asks once
  above the page when the app opens and after Stop: "ScadaBench left N addresses on your network
  from an earlier run. Remove them? Windows asks once." It offers Show addresses, Remove them and
  Keep for now. `CompactNotice` gains an optional action button.

## Evidence

- Go: `go test ./internal/netcfg/ ./internal/labsetup/ ./internal/labnet/ ./internal/api/...
  ./internal/app/ ./internal/snmp/service/` pass. New tests:
  - `TestHelperOffBlocksNewAddressesOnARealCard`
  - `TestIEDAndSNMPAddressesBecomeApprovals`
  - `TestSNMPAddressFollowsItsDeviceOnARealCard` (added at start, kept through IED Stop, removed
    with the device, unapproved refused)
  - `TestLeftoverAddressesAreTheJournaledAliasesStillThere` (the user's own address and a gone
    alias are not listed; removal needs the confirmed list)
- Frontend: `tsc --noEmit` clean; `node --test tests/frontend/*.test.ts` 483 pass.
- Owner's PC, built engine on a scratch data folder holding a copy of the owner's network journal,
  at 1366x768: the notice reads "ScadaBench left 38 addresses…" on one line, and Show addresses
  lists all 38 ("Remove address 192.168.230.41 that ScadaBench added to 'Ethernet'"). Nothing
  was removed: the owner confirms the Windows prompt himself.
- Windows event log: 7040 at 01:02 changed the service from auto to demand start. ScadaBench
  never writes that.

## Limitations

- Remove them and Repair were not clicked through here: both need the owner's Windows prompt.
- Firewall rules per plan still pile up (separate decision, ADR-0136 Compatibility).
- Not in the installed 0.1.32. The EXE was not rebuilt (owner rule). Python engine untouched.
