# Release 0.1.35: what the owner checks before publishing (2026-11-20)

Everything below was built and tested overnight on the owner's PC without an administrator
prompt. These checks need the owner: a Windows (UAC) prompt, a cable, a real card, a VM, or a
decision. The candidate build is in `studio/dist-go/` (`ScadaBench-Setup-0.1.35.exe`,
`ScadaBench-Portable-0.1.35.exe`); it is **not published**.

What is in it: fix plan 0220 (proven live: [0226](../changes/2026/0226-fixplan-windows-proof.md)),
the speed and network trials ([results](speed-trials-results-2026-11-20.md)): changes
[0221](../changes/2026/0221-network-failures-named.md), [0222](../changes/2026/0222-lab-addresses-owned-and-kept.md),
[0223](../changes/2026/0223-run-stop-trials.md), [0224](../changes/2026/0224-screen-trials-u1-u5.md),
[0225](../changes/2026/0225-fixed-firewall-rule-dropped.md) (dropped), and
[0227](../changes/2026/0227-gateway-restart-and-late-lab-adds.md).

## Candidate build (2026-11-20, from master after all merges)

- `compile_go_release.py --version 0.1.35`: installer compiled (with the B-3 uninstall step),
  antivirus gate 19 programs, 0 findings. `ScadaBench-Setup-0.1.35.exe` 20,248,032 bytes (SHA-256
  `cee23fad…`), `ScadaBench-Portable-0.1.35.exe` 54,877,184 bytes (`2d27ad6d…`).
- Before the build, on this PC with no other agent running: native workers rebuilt (VS 2019);
  Go engine `go vet` and `go test ./...`; frontend `tsc` and 511/511 tests (Node 24);
  `verify.py` all 8 checks PASS; protocol suite 95 passed, 9 skipped, 1 failed
  (`test_adapter_listing`, see "Known").
- Packaged folder on a throwaway data folder (your licence read from `%LOCALAPPDATA%\ScadaBench`
  like the installed app): `GET /app/update` current 0.1.35; sample IED Feeder01 added; 10
  Run/Stop rounds: the IED listens 0.02 s after Run answers (pre-R3 build: 0.06-0.08 s), every
  independent read of `Feeder01CTRL/XCBR1.Pos.stVal` with `studio_probe` answered `01`; Stop 0.01 s.

## Two decisions

1. **IED addresses stay after Stop (N10, ADR-0141 item 5).** Until now Stop took every IED address
   off (your words in ADR-0136: leftovers look "like trash"). Now they stay while the project is
   open and come off at Close, another project, Use, exit, or the lab page's **Take them off now**.
   Measured: Stop's address step 38 % → 0, Windows address changes 160 → 7 per 10 Run/Stop
   rounds, Run 13-14 % quicker from the second Run. The page names the kept addresses. If you
   prefer the old way, say so: it is one rule in `labsetup` (ADR-0141 item 5) to switch back.
2. **Each copy removes only its own addresses (ADR-0141 item 1).** This was the fix plan's Part B
   item "installed and portable copies remove each other's addresses" (not decided). Tonight it
   broke Runs ("MMS bind failed") 7 times between test engines, so it was fixed: 40/40 → 0/40.

## Checks that need you

Do them with the 0.1.35 candidate installed (or the dev app on master for 1-3).

1. **Undo after a drag (screen).** Drag a breaker on the diagram, press Undo once: it goes back.
2. **Second tab.** Open a second tab, switch auto save off in the first, bring the second to the
   front: it shows auto save off.
3. **Kept addresses (decision 1).** Run and Stop a Lab-card project: the lab page says "N addresses
   stay on the lab card until you close this project…"; **Take them off now** empties the card.
4. **Network service update (needed for 5 and 6).** Network › This PC lab network › **Repair**
   (one UAC prompt) installs the new network service.
5. **Cable out (N6).** Project on Ethernet with approved addresses, cable unplugged, press Run:
   "“Ethernet” is disconnected … Plug the cable in, then press Start again."; no IED address in
   `Get-NetIPAddress -InterfaceAlias Ethernet`. Plug in, Run works.
6. **Service stopped (N1).** Admin PowerShell `Stop-Service ScadaBenchLabNet`, press Stop: Activity
   and the lab page name each address left. `Start-Service ScadaBenchLabNet`; Run and Stop work.
   With the service stopped, Run asks Windows once; accept it and Run works.
7. **Real card Run/Stop (N2, N10, A6).** On Ethernet with approved addresses:
   `python studio/tools/trials/card_watch.py --card Ethernet --seconds 300`, then Run, Stop, Run,
   Stop: the second Run adds no address. Close the project: no IED address left on Ethernet. Set
   one address as the card's own static address in Windows, open ScadaBench, Run, Stop, Quit: it
   stays. A DHCP card's own address is never removed.
8. **Gateway address gone (N8).** Gateway on an own Lab address, Start; admin PowerShell
   `Remove-NetIPAddress -IPAddress <address> -Confirm:$false`: within about 2 s "Running, address
   gone"; Stop, Start: the address comes back.
9. **SCADA master own address (N9).** Set an own Lab address for the master, Connect: the address is
   on the card; Disconnect: it is gone.
10. **Modbus gateway Stop (0227).** Modbus gateway Start, connect a Modbus master and leave it idle,
    Stop: it stops at once.
11. **Two copies (ADR-0141).** Installed 0.1.35 and the dev app on the Lab card together: Stop one,
    the other's IEDs keep running and stay reachable.
12. **Faster Run (R3).** Packaged app, projects of about 7 and 20 IEDs: wait 10 s after opening,
    Run/Stop three times; while stopped Task Manager shows one idle `studio_ied` per simulated IED.
13. **Real IED switched off (R6).** Run, then Stop within 5 s: Stop ends at once.
14. **SCADA master to a real RTU (R7).** Connect answers within a second; points fill in by
    themselves; a converter started right after Connect serves every point.
15. **Licence sharing with a real VM (B-1).** Licensed host, ScadaBench in a Hyper-V or VMware
    guest: licensed by its host. Rename the host's Ethernet card "vEthernet (LAN)": another
    physical PC never gets the licence and never asks.
16. **Uninstall firewall option (B-3).** After any firewall or network-helper use, uninstall: it
    asks "Also remove the firewall rules ScadaBench added and switch Windows' own rules back on?"
    (Yes by default), one UAC prompt; `wf.msc` shows ScadaBench rules gone and Windows' Block rules
    on. No keeps the rules; refusing the prompt shows a failure.
17. **License server (A21-A23, B-2).** Not part of the app: deploy the license server only when you
    decide; the Lemon dashboard price and currency must equal the server's price.

## Known, not changed

- Two Go tests can fail only when the whole suite runs on a busy PC: licensing
  `cancelled_stays_cancelled` (Windows briefly refuses a rename while the scanner holds the file; the
  revoked IDs are also kept in the registry and the state file) and control
  `TestTypedInterlockAndSynchroBlocks` (two packages' tests can pick the same free port). Both pass
  10/10 alone. `test_adapter_listing` (frozen Python lister vs this PC's cards) fails as before.
- The installed 0.1.34 app still removes every Lab address it does not use until it is updated.
- N7 (Windows' duplicate check instead of the 0.7 s ARP wait), N11 (one firewall rule), U3 (lazy
  pages), R1, R2 and R5 were measured and dropped; reasons in the results.
