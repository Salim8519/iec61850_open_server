# 0221 Network trials N1, N2, N6, N7, N8: every address failure named, lost answers settled, cable out refused, gone gateway address shown

Date: 2026-11-20. Owner: the SCADA network "works fine sometimes, sometimes we don't understand
what happened"; adding and removing IP addresses is "not that good"; "do all of them, once i
wake, i should see all ready and no coding left, just testing".
Plan: [speed and network trials](../../testing/speed-trials-plan-2026-11-20.md), ideas N1, N2,
N6, N7, N8. Results: [trial results](../../testing/speed-trials-results-2026-11-20.md), [network details](../../testing/speed-trials-network-2026-11-20.md).
Decision: [ADR-0132](../../decisions/ADR-0132-real-card-addresses-follow-the-run.md) amended
(item 3: card without a link, still-checked addresses). Layers: Go engine (labsetup, app,
gateway service, netcfg), Go lab network service (labnet), frontend (gateway status).

## Change

- **N1, every failure named.** `EnsureForRun` (Run) returns every lab-card address the service
  refused, each with the IED, the address and the service's or Windows' reason, in project order
  (before: the refusals were journaled and Run went on). A real card names every refused address
  (before: only the first). A service that is off or does not answer while starting IEDs'
  addresses are not on the card stops the Run naming them (before: silence, then the preflight's
  "not set up" for LAN addresses). `Release` (after Stop, at launch and exit) now returns the
  addresses that stayed with their reason, including while the service is off (it reads the card
  from Windows' own list); the app shows it in Activity as `lab_address_left`, journals each as
  `failed`, and the lab page's problem names them while the service is off. The advice after the
  reasons fits them: a clash gets "Stop the other device ... or give ... another address"; a
  reason with its own advice gets none; anything else gets "Press Start again. If it keeps
  failing, press Repair". Files: `labsetup/failures.go` (new), `apply.go`, `batch.go`,
  `realcards.go`, `release.go`, `service.go`; `app/wire_runtime.go`, `startup.go`,
  `wire_snmp_network.go`.
- **N2, lost answers settled.** After a pipe error or timeout the card itself is read (the
  service's status, else Windows' list) and every address is journaled as it really is. A
  real-card add is journaled `requested` before the request; `requested` counts as added, so Stop
  takes off an address the service added after the app gave up (removal still needs the address
  on the card, never-source and approved, fix plan A6). Files: `labsetup/lost.go` (new),
  `real_journal.go`, `netcfg/journal.go` (accepts `requested`; `Outstanding` ignores it, so
  `journal.jsonl` reads as Python's). The lab card path adds no journal write.
- **N6, cable out.** The service adds no address to a real card without a link: "“Ethernet” is
  disconnected (no cable, or its network is switched off), so ScadaBench did not add <address>.
  Plug the cable in, then press Start again." An address Windows is still checking when the 6 s
  wait ends is taken off and reported ("Windows was still checking ... after 6 seconds, so
  ScadaBench took it off again. Check the cable and the network switch, then press Start again").
  The wait loop moved to a portable `labnet/settle.go` behind a small interface so it is tested
  with a fake card and clock; it never sleeps when every address is usable at once.
- **N7 dropped.** See Limitations; no code.
- **N8, gateway address gone.** A gateway that listens on one address of this PC asks Windows every
  2 s whether the address is still there (`netcfg.HasAddress`: one `GetUnicastIpAddressTable`
  read, no adapter listing, no DNS Client work). While it is gone the status has
  `address_gone: true` (the key exists only then, so the usual status stays byte-identical to
  Flask: `TestRoutesAgainstFlask` caught the first version) and the message "Address gone: <address> is no longer on this PC (it was
  taken off its network card), so SCADA cannot connect to the gateway." plus what to do (own
  lab address: "Press Stop, then Start: ScadaBench puts the address back on the lab card"); a
  traffic line and a `gateway_address_gone` / `gateway_address_back` event are written. The page
  shows the pill "Running, address gone" and the message as an alert (words, not colour alone).
  All networks and 127.0.0.1 are not watched. Files: `gateway/service/address_watch.go` (new),
  `manager.go`, `start.go`, `stop.go`, `app/wire_tools.go`, `netcfg/has_address_windows.go`
  (new), `iphelper_other.go`; frontend `GatewaySettings.tsx`, `copy/gateway.ts`, `api/gateway.ts`.
- Specs: lab-network NET-19..21, gateway GW-18; API: gateway status `address_gone`.

## Evidence

Measured on the owner's PC against the installed `ScadaBenchLabNet` service (release 0.1.34),
Windows' own address list as witness (`studio/tools/trials/n1_n2_lab_failures.ps1`,
`n8_gateway_address_gone.ps1`). The service refuses loopback 127.x, so the refused address was
127.67.21.x and the accepted ones 198.18.21.x (benchmark range, never routed).

- N1, before: one accepted and two refused addresses gave Run error `<nil>`; after: "TRIAL-01
  cannot start on 127.67.21.12: use an IPv4 address such as 192.168.216.1; loopback (127.x) ...
  cannot be used. TRIAL-02 cannot start on 127.67.21.13: ... Press Start again. If it keeps
  failing, press Repair ...". Nothing left on the card after Stop in both.
- N2, before: two adds whose answers were lost left no journal record at all (the addresses were
  on the card); after: both journaled `applied` from the service's status, then `removed`. The
  lab card's Stop removes by the service's list, so nothing was left there even before; the
  leftover is on real cards (unit test, Owner check).
- N8, before: the status kept "Listening on" for 10 s after the address left (never changed);
  after: "Address gone" in 1.71–2.06 s (31 runs), back to "Listening on" in 1.79–2.01 s in the 24
  runs where the address stayed back. In 7 runs the re-added address was gone again within 10 s;
  each time Windows' table and the service's list both lacked it, so "Address gone" was right.
  The cause is outside the watch and not proven: no engine journal on this PC recorded a removal
  then, and a probe without the gateway status (10 cycles: add, remove, wait 2 s, add, watch 6 s
  at 50 ms) never lost the address; another agent's direct call to the shared service is the
  likely remover. One Windows read costs 0.16–0.92 ms (mean of 1000), once every 2 s.
- No slowdown: Run's address step and Stop's release, 7 and 20 addresses, before and after
  interleaved in both orders, 50–60 runs each: 7 IEDs Run median 60.5 → 88 ms, Stop 163 → 224 ms;
  20 IEDs Run 220.5 → 193 ms, Stop 330.5 → 274.5 ms (worst runs 0.4–1.4 s on both sides; other
  agents' Runs shared the card). The code sends the same service requests and journal writes as
  before on this path (`TestRunAndStopSendOneRequestToANewService` asserts the exact sequence), and
  a fake-Windows benchmark of one Run plus Stop is about 2 ms on both sides (journal writes).
- Go: `go vet` (Windows, Linux, `-tags trial`) clean for labnet, labsetup, netcfg,
  gateway/service, app; `go test` passes for all five. New tests: 6 in `failures_test.go`, 3 in
  `lost_test.go`, 4 in `labnet/settle_test.go`, 2 in `address_watch_test.go`. `-race` needs cgo,
  which this PC lacks.
- Frontend: `tsc --noEmit` clean; frontend tests pass under Node 24 (new
  `gatewayAddressGone.test.ts`), 501 of 501.
- `verify.py`: every check PASS (the first run failed `TestRoutesAgainstFlask` while the gateway
  status always carried `address_gone`; fixed as above).

## Limitations

- Service-side changes (N6, `labnet`) reach the PC only with the next installation (Repair asks
  Windows once); they are unit-tested only. The engine shows any service's reason, so N1 works with
  the installed service today.
- N7 was dropped: today a device that answers ARP after 0.7 s is not seen (ADR-0132 accepted
  this). Letting Windows' duplicate check run after a late answer adds 3.3–3.5 s instead of 0.7 s
  to every real-card Start, and LAN devices answer ARP in milliseconds.
- A real-card address whose add answer was lost and that appears only after the next Release
  already ran stays until a Windows restart (the client timeout is 10 s; the service finishes
  within about 7 s).
- Not run: Stop with the service stopped (needs an administrator to stop it), Run with the
  Ethernet cable out, a real card at all (Wi-Fi is the owner's DHCP card, Ethernet unplugged).
  See Owner checks in the trial results.
- Two ScadaBench sessions on one PC take each other's lab addresses off at Stop and launch
  (`Release` keeps only what its own project uses). The first trial run took another agent's
  10.67.0.10–29 off for about a second and put them straight back; the harness now keeps every
  address that is not its own.
