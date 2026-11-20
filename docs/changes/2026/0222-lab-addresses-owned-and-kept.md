# 0222 Lab addresses: only this copy's are removed, device addresses follow their device, IED addresses stay for the session (trials N3, N4, N5, N9, N10)

Date: 2026-11-20. Owner: the SCADA network "works fine sometimes, sometimes we don't understand
what happened"; adding and removing IP addresses is "not that good"; "do all of them, once i
wake, i should see all ready and no coding left, just testing".
Plan: [speed and network trials](../../testing/speed-trials-plan-2026-11-20.md), ideas N3, N4,
N5, N9, N10, and the Lab-card finding of the same night. Results:
[trial results](../../testing/speed-trials-results-2026-11-20.md),
[locks and ownership details](../../testing/speed-trials-locks-2026-11-20.md).
Decision: [ADR-0141](../../decisions/ADR-0141-lab-addresses-owned-and-kept.md) (amends ADR-0125,
ADR-0127, ADR-0132; touches ADR-0136; settles fix plan Part B "copies remove each other's
addresses"). Layers: Go engine (labsetup, gateway service, SNMP service, app, labapi), frontend
(lab page line, words).

## Change

- **Only this copy's addresses come off (finding).** `Release` takes off a Lab-card address only
  when this copy's `lab_journal.jsonl` says it added it (newest `lab_address` record `applied` or
  `requested`), as fix plan A6 did for real cards. A journaled address that is gone is journaled
  `skipped` once, so a later copy's address at the same place is never taken for ours. Files:
  `labsetup/release.go`, `real_journal.go` (`outstanding(kind)`), `service.go` (`leftOn` names only
  ours), `own.go` (`RemoveOwn` removes only ours).
- **N3, one lock.** `EnsureOwn` and `RemoveOwn` take the same `card` lock as Run (`EnsureForRun`)
  and `Release` (before: `change`, so a gateway address could be added while a Release that had
  read the old state removed it). `AddOwn` is gone: typing an own address only checks it
  (`CheckOwn`) and stores the choice (N9), so the choice is always saved before anything is added.
- **N9, device addresses follow the device.** The gateway (`gateway/service/own_claim.go`, `start.go`,
  `stop.go`, `gateway_own.go`) and the master (`master_connect.go`, `master_own.go`) hold their own
  address from Start/Connect (claimed *before* `EnsureOwn`, reported in `CardAddresses`) until
  Stop/Disconnect/exit or a failed start, which take it off. `labsetup.Projects` gains `InUse()`
  (what devices hold now; Release keeps it) beside `Reserved()` (what they are set to use; the
  clash checks); `app/wire_lab.go` wires both. Choosing another network no longer touches the card.
- **N5, starting SNMP devices count.** A device's lab address is reported as held from just before
  its start puts it on (`snmp/service/labcard.go` `markStarting`, `devices.go`) until the start
  ends, not only once it runs.
- **N4, launch order.** The launch clean-up (`releaseLeftLab`) finishes before the master,
  gateway and SNMP devices start by themselves (`app/startup.go`).
- **N10, IED addresses kept for the session.** Addresses Run put on (Lab card and approved real
  cards) are remembered for the session (`labsetup/session.go`) and kept by Release while the
  project stays open; the next Run finds them and adds nothing. They come off when the project's
  addresses change while stopped (`app/lab_session.go` watches `project_changed`: another project,
  New, Close, This PC only, a move or a removed IED), at Use, after a move (`SetAddress`), at exit
  (`EndSession` in `app/shutdown.go`), from the lab page's **Take them off now**
  (`POST /network/lab/cleanup`, `labsetup.CleanUp`, refused while IEDs run), and after a kill at
  the next launch (empty session, journal says ours). `GET /network/lab` gains `kept`.
- **Words.** Lab page: "N addresses stay on the lab card until you close this project or
  ScadaBench, so the next Start is quicker. Take them off now"; the Use and ping hints and the
  summary say when addresses go on and come off (`copy/labNetwork.ts`, `LabNetwork.tsx`,
  `api/lab.ts`).
- Specs: lab-network NET-17 (amended), NET-22..24; gateway GW-19 and own-address paragraph;
  scada-master own address. API: gateway/master `network/own`, lab `kept`, `cleanup`, address move.
- Trial harness: `studio/tools/trials/labgroup.py`, `lab_cross_removal.py`, `n3_n5_lock_races.py`,
  `n4_launch_order.py`, `n9_gateway_address.py`, `n10_session_addresses.py` (README updated).

## Evidence

Measured on the owner's PC against the installed `ScadaBenchLabNet` service (0.1.34): today's code
= master 4b75e777 built to `.local/trials/c/base/`, trial = this change, both packaged behaviour,
Windows' own Lab-card list as witness, our own lab journal telling which engine removed what.
Ports 24000-24999, addresses 198.18.24.0/24. Full tables:
[locks and ownership trials](../../testing/speed-trials-locks-2026-11-20.md).

- **Cross-removal**: one copy's Run + Stop, Use, exit and launch took the other copy's 8 running
  addresses off in 40 of 40 actions today (all journaled by that copy), 0 of 40 with the trial; a
  kill + launch removed the other copy's addresses too today, only its own with the trial.
- **N3**: choosing the gateway's own address 200 times while IED and SNMP Stops looped: 80 of 200
  taken off by the same engine today; trial 0 removed by the engine, and gateway Start/Stop 200
  times under the same loops: 0 faults.
- **N4**: 1 of 60 launches today lost the gateway's and SNMP device's new addresses to the launch
  clean-up; trial 0 of 150 launches by this engine (30 exit, 120 End task).
- **N5**: SNMP start 200 times while an IED started and stopped: 47 of 200 failed or lost the address
  today (all removed by this engine), 0 of 200 with the trial.
- **N9**: address left after gateway Stop 10/10 and after exit today, 0/10 and none with the trial;
  10/10 starts and IEC 104 client connections on both; Start answer 0.13 → 0.24 s median.
- **N10**: Stop's address release 0.000 s in all 26 trial rounds (0.066 s / 0.118 s, 38 % of Stop
  today, 8 and 20 IEDs); Run's address step 0.000 s when no older engine touched the card (0.154 s /
  0.300 s, 13-14 % today); Windows address changes per 10 rounds 160 → 7 and DNS Client 50.4 → 6.3
  CPU s in the quiet set; after File › Close, exit and a kill + launch none of this copy's addresses
  stayed.
- Go: `go vet` (Windows, Linux, `-tags trial`) clean for labsetup, gateway/service, snmp/service,
  app, api/labapi; `go test` passes for labsetup, api/labapi, netaddr, snmp/service,
  gateway/service, app. New or changed tests: `labsetup/ownership_test.go` (5 new), the lab-card
  tests whose contract changed (Stop keeps, exit removes), `gateway_network_test.go` and
  `master_network_test.go` (held before added, off at Stop), `snmp/service/labcard_test.go` (N5;
  fails without `markStarting`, checked), `labapi/routes_test.go` (`cleanup`).
- Frontend: `tsc --noEmit` clean; 502 of 502 frontend tests under Node 24 (new test in
  `labNetwork.test.ts`).
- `verify.py`: every check PASS (file lengths, architecture, documentation, history, specifications, offline interface, GOOSE engine and Go engine tests).

## Limitations

- Engines built before this change (the installed 0.1.34 app, other agents' engines) still take
  every Lab address they do not use off the card; during these trials they took our kept addresses
  off many times (no record in any journal still on disk), which shows in the N10 sets 2-4 and in
  9 N4 launches. Only updated copies leave each other alone.
- An address a killed copy kept for its session stays until that copy launches again (same data
  folder) or Windows restarts; nobody else removes it any more.
- Addresses an older ScadaBench put on the card without a journal record in this data folder are no
  longer removed by this copy (Windows clears them at restart).
- A lab add whose answer was lost and that the service finishes later is journaled `failed`
  (change 0221) and so is not removed by this copy.
- N10 keeps addresses visible after Stop (ping answers until the project closes); the owner said in
  ADR-0136 that addresses left after Stop "looks like a trash", so this is on the owner's list; the
  lab page names them and offers "Take them off now".
- Gateway Start is about 0.1 s slower (it puts the address on); a gateway Start right after a Stop
  was refused 6 times in 200 on today's code with "Could not listen ... Another program may use
  this port" while the address was there (the previous gateway's port), not changed here.
- Not run: real cards (N10 on Ethernet: no ARP question and no DNS Client work at the second Start),
  the master's own address live (unit tests only).
