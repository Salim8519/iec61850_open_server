# Speed and network trials: results (2026-11-20)

Plan: [speed-trials-plan-2026-11-20.md](speed-trials-plan-2026-11-20.md). One row per idea:
today's number, the trial's number, runs, verdict and reason. Harnesses: `studio/tools/trials/`.

| Idea | Today's number | Trial's number | Runs | Verdict | Reason |
|---|---|---|---|---|---|
| N1 | 2 of 3 addresses refused by the service: Run error none (Run went on); Stop with the service off: silent | Run stops naming both IEDs, addresses and the service's reason; Stop names each address left with its reason (Activity `lab_address_left`, lab page) | 3 live (refused addresses), unit tests for service off and refused removal | Consider (implemented, change 0221) | Faults went from silent to named with the reason; nothing saved changes; Run and Stop timing within noise (below) |
| N2 | Lost answers on 2 adds: 0 journal records (addresses on the card); real card: address never removed (unit test) | Both journaled from the card (`applied`, then `removed`); real-card add journaled `requested`, removed at Stop | 3 live (lab card), unit tests (real card) | Consider (implemented, change 0221) | Real-card leftover from 1 to 0 in the unit test; the lab card had no leftover even before (its Stop removes by the service's list) |
| N6 | Cable out: address added, reported OK, left tentative; still-checking after 6 s reported OK | Refused with "“Ethernet” is disconnected ... Plug the cable in, then press Start again"; still-checking taken off and reported | Unit tests only (service side; cable needs the owner) | Needs the owner's eye (implemented, change 0221) | Fault removed in the service's logic; live proof needs a reinstall (UAC) and the cable out |
| N7 | Device answering ARP after 0.7 s is not seen (ADR-0132 accepted) | Not built | 0 | Drop | Would add Windows' 3.3–3.5 s duplicate check instead of 0.7 s to every real-card Start; LAN devices answer ARP in milliseconds |
| N8 | Gateway status unchanged 10 s after its address left the card (never changes) | "Address gone" in 1.77–2.06 s; cleared in 1.84–2.01 s; one Windows read 0.16–0.92 ms every 2 s | 1 before, 21 after | Consider (implemented, change 0221) | Status changes within a few seconds; negligible CPU; Running gateway untouched |

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
change. (One "after" round of 20 failed its own check once, most likely another session's Stop
taking the trial's addresses off; it did not repeat in 2 more runs.)

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
