# 0206: Ethernet addresses follow Start and Stop without a prompt; Repair updates a running service

Owner: "the problem is the annoyance of the IPs", "lets make best user experience and smoothest".
Decision: [ADR-0132](../../decisions/ADR-0132-real-card-addresses-follow-the-run.md).

## Change

| Area | Files |
| --- | --- |
| Service (protocol 3) | `labnet/protocol.go` (`card`, `ApprovedKey`, `ApprovalName`, `RealCards`), `handler.go`, `handler_real.go` (new: approved only, no DHCP card, no other card's address, ask first), `realaddr_windows.go` (new: cards, approvals, 0.7 s ARP question, ready-at-once add, wait until usable), `adapters_windows.go` (DHCP flag), `service_windows.go`, `labnet_other.go` |
| Repair | `labnet/install_windows.go`: wait for the old service process, retry the copy, restart the old service on failure |
| Apply writes approvals | `netcfg/approval.go` (new), `service_plan.go`, `service.go` (`RealCards`), `commands.go`, `describe.go`, `item_fields.go`, `helper_validation.go`, `service_cleanup.go` |
| Start / Stop / exit | `labsetup/realcards.go` (new), `apply.go` (`EnsureForRun`), `release.go`, `service.go` (`Approved`, Repair offered below protocol 3), `app/wire_lab.go`, `app/wire_platform.go` |
| Page | `frontend/src/features/network/setupWords.ts`, `ReviewStep.tsx`, `api/network.ts`, `copy/labNetwork.ts` |
| Release package | `tools/go_release_steps.py`: `scadabench_labsvc.exe` is shipped (the first 0.1.31 build left it out: Repair would have reported it missing); `tests/unit/test_go_release.py` |
| Tests | `labnet/handler_real_test.go`, `labsetup/realcards_test.go`, `netcfg/approval_test.go`, `labsetup/batch_test.go` |

## Evidence (owner's PC, Kaspersky Total Security 21.3)

- Go tests for `labnet`, `labsetup`, `netcfg`, `app` pass; `tsc --noEmit` clean. Both lab programs
  scanned clean by the release gate each build and placed in the installed 0.1.30 folder.
- First Repair to the new service failed: "the service program could not be copied … being used",
  service left stopped. After the fix Repair succeeded in 10 s (owner approved), and again later.
- Test engine on a copy of the owner's data, real workers from the installed folder:
  - Clean up of the owner's 19 leftover changes (7 Ethernet IED addresses from an apply the
    installed 0.1.30 made in 70 s, SNMP addresses, firewall rules): one prompt, 34 s from request
    to result including the click; Ethernet kept only the owner's 192.168.40.111.
  - Preview on VMware VMnet8 (connected, 192.168.230.1/24): 7 `approval` items, SNMP alias, two
    firewall rules; Apply with one prompt: 10/10 ok, 7 values in `Parameters\Approved`, no IED
    address on VMnet8.
  - Start with no prompt: all 7 IEDs running, addresses Preferred, SkipAsSource. Stop removed them
    (0.17 s). Service call alone, 7 addresses: add 0.74–0.99 s, all usable at once; remove 0.07–0.08 s
    (once 3.3 s).
  - Start of the same project: VMnet8 11.2–13.1 s, ScadaBench Lab card 11.0–11.9 s.
- Before the ARP question, Windows' own duplicate check made the address usable only after
  3.3–3.5 s, and the first Start failed with WinError 10049 (the service then waited 3 s only).

## Limitations

- Not released: the installed app is still 0.1.30. Each PC needs one Repair after the update.
- The 0.7 s ARP question replaces Windows' duplicate check where the card has an address on that
  network; a device that answers ARP later than 0.7 s is not seen.
- SNMP device addresses on a real card still stay until Clean up (their own Start/Stop is not
  wired yet).
- Start of this 7-IED project takes 11–13 s on this laptop on every card; Stop varied 0.2–8.8 s
  on both cards. Neither comes from this change and neither was investigated here.
- Ethernet itself (no cable here) was not used for Start; VMnet8 stood in as the connected card.
