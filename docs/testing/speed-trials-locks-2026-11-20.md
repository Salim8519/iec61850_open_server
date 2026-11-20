# Speed and network trials: Lab-card locks and ownership, N3, N4, N5, N9, N10 (2026-11-20)

Summary rows: [results](speed-trials-results-2026-11-20.md). Change record:
[0222](../changes/2026/0222-lab-addresses-owned-and-kept.md). Decision:
[ADR-0141](../decisions/ADR-0141-lab-addresses-owned-and-kept.md). Harness:
`studio/tools/trials/` (`labgroup.py` and the five scripts named below; README there).

## How these trials ran

- **Today's code** = master at 4b75e777 built to `.local/trials/c/base/studio_server.exe`; **trial**
  = the working tree of change 0222 built to `studio/build/engine/studio_server.exe`. Both without
  `--dev` (packaged behaviour), development native workers, the installed `ScadaBenchLabNet`
  service (0.1.34), own data folder and `LOCALAPPDATA` per engine, ports 24000-24999, Lab
  addresses 198.18.24.0/24, the 8-IED project (and the 20-IED one for N10 timing).
- **Windows is the witness**: every check reads Windows' own address list of the Lab card. Which
  engine removed an address is read from that engine's own `lab_journal.jsonl` (`by_us`).
- **Other agents' engines ran at the same time** (screen trials on ports 22001/22002, the firewall
  group, trial instances whose data folders are deleted afterwards). Engines built before 0222 take
  every Lab address they do not use off the card, ours included, so a check that found an address
  missing *without* a removal in our own journal is not counted against the code under test; each
  result file lists them.

## Finding: copies take each other's addresses off

`lab_cross_removal.py --rounds 10`: copy B runs 8 IEDs (198.18.24.110-117) the whole time; copy A
(same project at .10-.17) does Run + Stop, Use, exit and launch in each round.

| | today | trial |
|---|---|---|
| actions that took B's addresses off | **40 of 40** (Run + Stop 10, Use 10, exit 10, launch 10), all journaled `removed` by A | **0 of 40** |
| A killed while running, then launched again | A's 8 addresses removed, **and B's 8 running addresses too** | A's own 8 removed, B's 8 untouched |
| left on the card after both exit | nothing | nothing (one .10 left by an earlier killed harness engine, see Limitations) |

Earlier attempts of the same script on today's code twice ended with "MMS bind failed:
198.18.24.x:24102": a Run of one copy started while the other copy's launch clean-up took its new
addresses off; this is the failure other agents saw 7 times that night.

## N3: one lock, choice saved before adding

`n3_n5_lock_races.py --idea n3`: one IED starts and stops in a loop (77-189 Stops, each a Release)
and SNMP Stop loops (1653-4001 calls, each a Release) while the main loop works 200 times.

| Loop | today | trial |
|---|---|---|
| `--mode add`: choose the gateway's own address .50, .51, .52, .53 in turn | **80 of 200** addresses missing 0.27-0.87 s later, **80 removed by this engine** (it added the address, then saved the choice; a Release in between read the old choice) | 0 removed by this engine; a typed address no longer goes on the card at all (N9), so the 200 "not on the card" here are by design |
| `--mode start`: Start the gateway on .50, check, Stop | 0 removed by this engine (the address stayed on the card all the time: N9's leftover); 6 Starts refused "Could not listen ... Another program may use this port" right after a Stop while the address was there (the old gateway's port), 1 address missing not by us | **0 of 200** |

Follow-up [0227](../changes/2026/0227-gateway-restart-and-late-lab-adds.md): the six refused Starts were not reproduced on today's code (0 of 800); the gateway's message named every bind failure a busy port, also a Lab address taken off by another engine. It now names the cause.

## N4: launch clean-up before the automatic starts

`n4_launch_order.py --launches 30`: gateway on its own address .50 and one SNMP device at .60,
both started, then 30 × close and launch again on the same data folder.

| | today | trial |
|---|---|---|
| exit = the app's own exit (Ctrl+Break) | **1 of 60** launches (two sets of 30): this engine's launch clean-up took .50 and .60 off after the gateway had started on .50 ("Address gone"; the clean-up had read the card before the project was loaded, so the gateway's address was not kept) | **0 of 30** |
| exit = End task (kill) | not measured | **0 of 120 by this engine** (four sets of 30). 9 launches had an address gone with no removal in this engine's journal after its own add (another engine on the PC; the journal timeline is in the result files), and in 1 the gateway's bind failed ("Could not listen on 198.18.24.50:24404 ...") and its failed start took its own address off, as it should |

The first set on today's code read the wrong status field (`run_state`) and waited 30 s per
launch; its launch 1 is the fault above, the other 29 were clean. The second set (fixed) was clean.

## N5: starting SNMP devices count as kept

`n3_n5_lock_races.py --idea n5`: one IED starts and stops in a loop (146-196 Stops) while the SNMP
device at .60 starts and stops 200 times.

| | today | trial |
|---|---|---|
| starts that failed or lost the address | **47 of 200**, all 47 removed by this engine: 44 "Network switch 1: 198.18.24.60 is not an address of this PC", 3 running without its address | **0 of 200** |

## N9: the gateway's own address only while it runs

`n9_gateway_address.py --cycles 10`.

| | today | trial |
|---|---|---|
| on the card right after typing it | yes | no |
| Start succeeded / IEC 104 client connected | 10/10, 10/10 | 10/10, 10/10 |
| address left after Stop | **10 of 10** | **0 of 10** |
| address left after exit | yes | no |
| Start answer, median (worst) | 0.130 s (0.274) | 0.238 s (1.271): Start now adds the address |
| Stop answer, median (worst) | 0.042 s (0.073) | 0.056 s (0.111) |

## N10: IED addresses kept for the session

`n10_session_addresses.py --rounds 10`: Run/Stop rounds cut into steps as Step 0 does (round =
Run, 2 s, Stop, 3 s), the Windows DNS Client's CPU over the rounds and Windows' own address changes,
then what stays after Stop, File › Close, exit and a kill + launch. Four sets per side; "counted"
leaves out rounds in which another engine took our addresses off while running (`disturbed`).

| Set (counted rounds, machine busy) | Run median / worst s | Stop median / worst s | Run "addresses" step | Stop "address release" step | Windows address changes | DNS Client CPU s |
|---|---|---|---|---|---|---|
| small, today, 1 (10, 64 %) | 1.077 / 1.825 | 0.174 / 0.410 | 0.154 (14 %) | 0.066 (38 %) | 80 added, 80 removed | 50.4 |
| small, trial, 1 (10, 28 %) | 0.679 / 0.931 | 0.104 / 0.126 | 0.000 | 0.000 | 7 added (round 1), 0 removed | 6.3 |
| small, today, 2 (8, 64 %) | 1.170 / 1.892 | 0.229 / 4.614 | 0.149 | 0.065 | 80 / 64 | 60.6 |
| small, trial, 2 (6, 58 %) | 1.125 / 1.782 | 0.157 / 0.343 | 0.115 | 0.000 | 48 / 0 | 45.0 |
| large, today (7, 74 %) | 2.281 / 3.257 | 0.310 / 2.220 | 0.300 (13 %) | 0.118 (38 %) | 160 / 140 | 103.0 |
| large, trial (6, 80 %) | 2.374 / 2.774 | 0.328 / 2.011 | 0.321 | 0.000 | 80 / 0 | 68.8 |

- **Stop**: the address release is gone in every trial round (0.000 s in all 26 counted rounds;
  38 % of Stop's median today). The trial never removed an address at Stop.
- **Run**: the address step is gone when nothing else touches the card (set 1: only round 1 added,
  the other rounds 0.000 s). In the later sets the other agents' engines (built before 0222, e.g. a
  renamed `studio_server_r1.exe` on port 26000 that the harness's busy check does not see) took our
  kept addresses off between rounds, so the trial added them again 5-9 times per set (48/80 adds,
  0 removes); their Runs and Stops also loaded the machine to 58-80 %, which moves "model load"
  more than any address step. Set 1 ran at a different load than today's set 1, so its 37 %/40 %
  are not the trial's gain; the step columns are.
- **DNS Client**: CPU follows Windows' address changes: 160 changes cost 50-61 CPU s per 10 small
  rounds today; 7 changes cost 6.3 CPU s (about the idle baseline of 1.1 CPU s per 30 s).
- **Nothing left behind** (Windows' list after each step, trial; today has 0 everywhere because Stop
  removes them):

  | | after Stop | after File › Close | after exit | after a kill | after the launch that follows |
  |---|---|---|---|---|---|
  | trial, set 1 | 8 (kept) | 0 own (+1) | 0 own (+1) | 8 | 0 own (+1) |
  | trial, set 2 | 0 (taken off by another engine) | 0 | 0 | 8 | 0 |

  (+1) is 198.18.24.10, left at 11:27:48 by a harness engine of the N3 trial that was killed and
  whose data folder was deleted (Windows' creation time); not this copy's, so correctly left alone.

## Group B's re-added gateway address that vanished (7 of 31 N8 runs)

The races fixed here (N3, N5) are inside one engine, between its own Release and its own device
start; group B's N8 trial added the address straight through the service from a Go test, with no
engine Release in that process, so N3/N5 do not explain it. The cross-removal finding does: every
engine built before 0222 that ran a Release (a Stop, Use, launch or exit) at that time took the
address off as "not mine", and journaled it only in its own data folder. The trial harnesses of
this night delete their data folders after each run (`.local/trials/instances/<id>`), so such a
removal leaves no journal anyone can read later. During these trials the same thing happened to
our addresses many times (9 N4 launches, N10 sets 2-4, N3 start loop 1) with no record in any journal still on disk; one remover running then was a renamed engine (`studio_server_r1.exe`) of another agent, built before 0222, whose data folders are deleted after each run.
This fits all 7 cases but is not proven for them: their journals are gone.

## Limitations

- An address a killed engine kept for its session stays on the card until that same copy (same
  data folder) launches again or Windows restarts; a copy whose data folder is deleted never
  removes it, and since ADR-0141 no other copy does either. One harness engine of these trials was
  killed that way (198.18.24.10, removed afterwards with `labgroup.clean_card`).
- A lab-card add whose answer was lost and that the service finishes after the app read the card
  is journaled `failed` (change 0221) and so is not this copy's; it stays until a Windows restart.
- Engines built before 0222 (the installed 0.1.34 app, other agents' engines) still take every
  address they do not use off the card; only updated copies leave each other alone.
- Real cards (N10 on Ethernet: no ARP question and no DNS Client work at the second Start) are
  Owner checks; Wi-Fi is the owner's DHCP card and Ethernet is unplugged.
