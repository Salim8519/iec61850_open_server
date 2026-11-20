# 0226 Fix plan 0220 proven live on Windows; three gaps found and fixed

Date: 2026-11-20. Owner: "do all of them, once i wake, i should see all ready and no coding left,
just testing, make sure all works fine". Lead: run the fix plan's laptop checklist against real
processes before release 0.1.35.
Plan: [fix plan, laptop checklist](../../testing/fix-plan-2026-11-19.md). Earlier change:
[0220](0220-fix-plan-phase-a.md). Decision: [ADR-0140](../../decisions/ADR-0140-fix-plan-safety-decisions.md)
(decision 3, "Unknown is not failed", now also covers gateway commands).
Layers: Go engine (acquisition, gateway service, project service); trial scripts.

## Change

- **Frozen real device goes stale (A18).** A real IED that stops answering while its connection
  stays open (a frozen relay, a pulled cable) kept "Receiving live values by report" for 100 s
  and more, with frozen values shown as fresh: every read answered per point with a timeout, which
  the engine took as an answer. A read where no point got a reply (libiec61850 timeout, not
  connected, connection lost) is now an error, so the live loop ends, the values are marked stale
  ("The device did not answer a read (no reply in time)") and the session reconnects
  (`acquisition/silent_read.go`, used by the mode refresh, the poll and "Read these now").
- **Gateway command whose control program froze (A8 in the gateway).** Its command-log line stayed
  empty forever and SCADA got "Native operate request timed out". The line and the answer now say
  "result unknown: <IED> did not answer in time; result unknown — check the position before
  trying again", like Devices; any other control failure is logged as "failed: <reason>"
  (`gateway/service/commands.go`).
- **Failed Save in words (A5).** A read-only, locked or full project file reached the page as
  "The server encountered an internal error ... the server is overloaded". Save now answers 503
  "ScadaBench could not write the project file (Access is denied). Check that the data folder is
  not read-only or full and that no other program has the project file open, then press Save
  again." (`project/service/save_error.go`); the unsaved mark stays.
- **Trial scripts** `studio/tools/trials/fixplan_*.py` (not in `verify.py`): each starts its own
  engine (ports 23000-23999, IEDs on 127.67.23.x, temp data folder, native workers of the main
  checkout), drives real processes, freezes programs with `NtSuspendProcess` like Task Manager,
  and writes its results to `.local/fixplan/`. The UI check drives the real page in headless
  Chrome over the DevTools protocol.

## Evidence (this PC, Windows 11, release build of the native workers from 2026-11-20)

| Item | Live check | Result |
| --- | --- | --- |
| A1 | 10 rounds: drag, Ctrl+S, drag again during that save and its project read | engine drawing = page drawing every round, "All changes saved" |
| A2 | every diagram PUT failed (CDP); then the engine process ended | "Not saved", moved breaker kept 2 s, nothing reverted; Ctrl+S saved it once the engine answered |
| A3 | 10 rounds: edits back to back, engine ended by TerminateProcess after 0.3-1.8 s | every acknowledged edit opened again (the in-flight one too in 4 of 10), project open in 3-152 ms |
| A4 | journal line 4 of 8 damaged | started; journal and autosave copy in `recovery\`; last autosave opened with the notice |
| A5 | autosave read-only, then held open without sharing | Save 503 with the plain reason both times (before: generic 500); Save 200 once free |
| A6 | Lab card: Run, Stop, Quit, new launch; Clean up preview | the IED's own address on while running, off after Stop and Quit; Clean up lists nothing |
| A7 | 4 SCADA commands, the first stuck in a frozen studio_control | Stop 0.02 s; 3 logged "refused: the gateway stopped before this command ran" |
| A8 | Devices operate with studio_control frozen | 30.0 s, 503 "... result unknown — check the position before trying again", event `real_command_unknown`, frozen program gone; a second operate refused at once; the next one works (0.1 s) |
| A8 gw | the stuck gateway command after its 30 s | log "result unknown: Field did not answer in time; ..." (before: empty line) |
| A9 | 20,008 malformed frames (review's 26-byte VLAN frame, bad goosePdu lengths, random) to studio_goose's bus | same process, stNum 2→3 received after a change. The studio_goose built from before 0220 crashed on 2,008 of them |
| A10 | B's IED worker frozen, its pipe full of updates | Stop 6.0 s, Quit 13.6 s, frozen worker killed, nothing left running |
| A11 | studio_gateway frozen, then Stop | 16.0 s (10 s halt + close), state "stopped", program gone |
| A12 | 4 silent masters, a fifth; 40 short connections | fifth turned away while fresh, all 4 closed after 60 s, fifth reads; threads 2 → 2 |
| A13 | float at register 10 + double at 11 | refused when saving the list: "Point 2 (Bay BayCTRL/XCBR1.Pos.stVal): address 11 is already used by point 1" |
| A14 | Modbus master waiting on the stuck command | answered when Stop ran (connection closed after 2.7 s; the others exception 4) |
| A18 | B's IED worker frozen | `live_stale` after 13.8 s (before this change: never in 100 s) |
| A19/A20 | Bay2's address busy (5 rounds); Bay2's worker frozen during its start | refused, Bay2 not running, one studio_ied left; frozen start 503 after 16 s, worker killed; the same Start then works, never "already started" |
| B-1 | this PC | firmware not a VM; served only Hyper-V Default Switch, VMnet1, VMnet8; Ethernet, Wi-Fi, ZeroTier, VPN and WSL never |

- Go tests failing before and passing after: `TestSilentDeviceMarksValuesStale`,
  `TestSilentModeReadIsAnError`, `TestGatewayCommandTimeoutIsResultUnknown`,
  `TestGatewayCommandFailureIsLogged`, `TestFailedSaveIsExplained`; `TestPartlyAnsweringDeviceStaysLive`
  guards against over-reach. `go vet` and `go test` pass for acquisition, gateway, project/service,
  realcmd, native, tools/worker, tools/control, store, labsetup, labnet, licensing.
- Native protocol tests with the release workers: `test_gateway_modbus_safety.py`,
  `test_native_worker.py`, `test_gateway_wire.py` 15 of 15 pass (A12-A14, A20).

## Limitations

- A device that answers reports but no read within 5 s for two cycles is now treated as gone and
  reconnects; "Read these now" on a silent device answers 503 instead of one error per point.
- Lab card (owner decision, fix plan Part B "not decided"): Stop, Quit and every launch remove
  each Lab-card address that no IED of *this* copy uses, also one another program or another
  ScadaBench put there (seen live). Two copies running together (or several test engines)
  remove each other's addresses. The app also refuses 127.x on the Lab card.
- With one frozen worker, Stop can take up to 16 s and Quit 18 s (a request already waiting
  ends after 10 s, then 3 s stop and 3 s exit wait before the kill): bounded, not instant.
- A13 on the page names the first point by its row ("point 1"), the native check names both.
- Not run here (Owner checks): A6 on a real card, B-1 with a real VM, B-3 uninstall (UAC).
