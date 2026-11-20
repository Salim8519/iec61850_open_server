# Speed and network trials: network group N1, N2, N6-N8 (2026-11-20)

Summary rows: [results](speed-trials-results-2026-11-20.md). Change record:
[0221](../changes/2026/0221-network-failures-named.md). Harnesses: `studio/tools/trials/`.

## No slowdown on the happy path (N1, N2)

Run's address step and Stop's release against the installed service, before and after, rounds
interleaved in both orders (other agents' Runs shared the Lab card, so single runs vary widely):

| Addresses | Run median before → after | Run worst before / after | Stop median before → after | Stop worst before / after | Runs each |
|---|---|---|---|---|---|
| 7 | 60.5 → 88 ms | 381 / 463 ms | 163 → 224 ms | 484 / 554 ms | 60 |
| 20 | 220.5 → 193 ms | 986 / 1128 ms | 330.5 → 274.5 ms | 1421 / 856 ms | 60 / 50 |

The changed code sends exactly the same service requests and journal writes on this path
(asserted by `TestRunAndStopSendOneRequestToANewService`); a fake-Windows benchmark of one Run
plus Stop is about 2 ms on both sides. The differences are Windows and machine load, not the
change. (One "after" round of 20 failed its own check once: another engine's Stop took the
trial's addresses off. Its lab journal, `%TEMP%ixplan-lab-*\data
etwork\lab_journal.jsonl`,
records "removed" for 198.18.21.100 and up at 09:51 local, during that round. It did not repeat
in 2 more runs. Engines on one PC take each other's Lab addresses off at launch and Stop; a later
group fixes that, so the trials do not.)

## N8 runs where the address did not come back

In 7 of 31 runs the trial's re-added 198.18.21.50 was gone again within 10 s. Windows' table and
the service's list both lacked it, so the status ("Address gone") was right. No engine journal on
this PC (`%TEMP%`, every worktree's and the main `.local`) was written in those minutes, so no
engine's Stop or launch removed it. A probe without the gateway status (10 cycles: add, remove,
wait 2 s, add, watch 6 s every 50 ms) never lost the address. The cause is outside the watch and
not proven; another agent's direct call to the shared service is the likely one.

## Owner checks (N1, N2, N6, N8)

1. **N1, service stopped at Stop.** Run a project on the ScadaBench Lab card, then in an
   administrator PowerShell `Stop-Service ScadaBenchLabNet`, press Stop. Expect in Activity
   "lab address left: ScadaBench could not take these addresses off their network card: <address>
   on ScadaBench Lab (the lab network service is off) ...", and on Network › This PC lab network
   "The lab network service is off, so ScadaBench cannot take <addresses> off the card ...".
   `Start-Service ScadaBenchLabNet`, press Run and Stop: the addresses go.
2. **N1, service stopped at Run.** With the service stopped, press Run and refuse the Windows
   prompt: "Windows did not get permission ..."; accept it: Run works.
3. **N6, cable out (after Repair installs the new service).** Network › This PC lab network ›
   Repair (one prompt). Project on Ethernet with approved addresses; unplug the cable; press Run.
   Expect "<IED> cannot start on <address>: “Ethernet” is disconnected (no cable, or its network
   is switched off), so ScadaBench did not add <address>. Plug the cable in, then press Start
   again." and `Get-NetIPAddress -InterfaceAlias Ethernet` shows no IED address. Plug the cable in,
   press Run: it starts.
4. **N2 on a real card.** Not reproducible by hand; covered by `lost_test.go`. After any Run and
   Stop on Ethernet, `Get-NetIPAddress -InterfaceAlias Ethernet` must show no IED address.
5. **N8 in the app.** Gateway on an own Lab-card address, Start; in an administrator PowerShell
   `Remove-NetIPAddress -IPAddress <address> -Confirm:$false`. Within about 2 s the gateway shows
   "Running, address gone" with the message; Stop, then Start puts the address back and shows
   "Running".
