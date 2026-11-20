# Speed and network trials plan (2026-11-20)

Owner, 2026-11-20: the SCADA network "works fine sometimes, sometimes we don't understand what
happened"; adding and removing IP addresses is "not that good"; Run and Stop are slow. The owner
asked for a plan in which every idea is first **tested on the real machine against today's code**
and only the winners are proposed.

Source: a read-only review of the Go engine and frontend on Linux. Nothing was run on Windows.
The findings below come from reading code and must be reproduced by a trial before any fix.
This plan changes no product code.

## Rules for whoever runs the trials

- **Nothing is merged.** Each idea is a small trial on its own branch `trial-<id>`, built only to
  be measured. A winner becomes a normal change later (ADR where noted, change record, ledger).
- **Run on the owner's Windows PC**: real cards, the `ScadaBenchLabNet` service, admin prompts,
  real processes, a real browser. Fakes do not count as evidence here.
- **Never touch the owner's running app, its data folder or `studio/native/build/Release`.** Use a
  separate instance with a temp data folder, loopback 127.67.x.y, ports >= 20000. Native trial
  builds go to `studio/native/build/trial-<id>`. No EXE.
- **Trial tests are not part of `verify.py`.** They live in `studio/tools/trials/`, one file per
  trial, under 300 lines each. They may be slow and may need an admin prompt.
- **Same test, both sides.** Run each test on today's code first, then on the trial: at least 10
  runs each (races: the loop counts given below), same project, same card, same PC state. Use two
  projects: about 7 IEDs and about 20 IEDs. Record median and worst.
- **Windows is the witness.** Address checks read the real Windows address list, not the project
  or the journal. Protocol checks use an independent MMS or IEC 104 client (AGENTS.md).
- Python stays frozen. Trials are Go under `studio/engine`, frontend, or native where noted.

## Verdict for each idea

| Verdict | When |
|---|---|
| **Consider** | The trial is at least 20 % faster (median, and the worst run is not worse), or the fault count goes from some to zero; and nothing the user sees or saves changes |
| **Drop** | Not faster, breaks something, or the fault could not be reproduced on today's code. Write the reason; do not propose it |
| **Needs the owner's eye** | The numbers pass but a person must look or act. Collect these and ask the owner once, after all trials |

Only ideas marked Consider later get one small, quick test in normal verification.

## Step 0: measure today first

Time each step of Run and Stop (addresses, listen check, process start, model load, saved values,
first values read, initial states, test modes, GOOSE bus, address release, "stopped" shown) on the
Lab card and on a real card, with the packaged-style engine. The last numbers are from
2026-11-01 (change 0208: Start 2.5-4.1 s, Stop 0.32 s). Skip any trial below whose step turns out
to cost under 5 % of the total.

Also measure the Windows DNS Client CPU for 30 s after an address add and after a remove (change
0208 recorded 5-11 CPUs busy for 20-30 s).

## Network trials

Paths are under `studio/engine/internal/`.

| # | Finding in today's code | Idea | Test on the machine | Wins if |
|---|---|---|---|---|
| N1 | Add failures ignored (`labsetup/apply.go:144-155`); only the first refused address shown (`labsetup/realcards.go:57-61`); a stopped service makes clean-up do nothing (`labsetup/release.go:26`) | Report every failed address with Windows' reason, on Run and on Stop | Stop the service; refuse an address; make an add fail | Each case shows a clear message naming the address and reason |
| N2 | A pipe error or timeout journals nothing (`labsetup/realcards.go:156-158`, `batch.go:38-39,63-64`, `apply.go:175-176`); the service may still have added the address, and nothing removes it | Journal "requested" before the call; after an error, ask the service what is really there | Cut the reply mid-add (client timeout shortened in the trial harness), then Stop | Address left on the card today, none with the trial |
| N3 | Own-address calls take lock `change`, Run and Release take `card` (`labsetup/service.go:61-62`) | One lock for every add and remove; save the choice before adding | Add a gateway own address while IED Stop and SNMP Stop loop 200 times | Address missing at least once today, never with the trial |
| N4 | `go releaseLeftLab()` and `go resumeGateway()` run unordered (`app/startup.go:36-37`) | Clean-up finishes before gateway and master auto-start | Launch 30 times with gateway auto-start on an own lab address | Bind failures today, none with the trial |
| N5 | SNMP devices are protected only once "running" (`snmp/service/labcard.go:51-53`) | Count "starting" devices as kept | Start SNMP devices while IED Stop loops 200 times | Same rule as N3 |
| N6 | Card down: no ARP sent, wait returns at once, address journaled "applied" (`labnet/realaddr_windows.go:76-78,145`); still-tentative addresses after 6 s are reported OK (`:149-164`) | Report "card disconnected" or "still checking" as a failure and take the address off | Run with the cable out. **Owner unplugs the cable** | Clear message; no address left; Run works again after re-plug |
| N7 | When the 0.7 s ARP wait gets no answer the address is added as Preferred, skipping Windows' duplicate check (`labnet/realaddr_windows.go:86,95-96,125-127`) | Treat a timeout as "not asked" so Windows checks | A VM holding the address that answers ARP after 1 s | Address taken today, refused with the trial; measure the extra wait |
| N8 | Nothing re-checks a running gateway's address | Watch the bind address; show "address gone" | Remove the address from Windows while the gateway runs | Status changes within a few seconds |
| N9 | Gateway own address stays after Stop and exit (`gateway/service/stop.go` has no Lab call; `labsetup/release.go:34` keeps it) | Add at Start/Connect, remove at Stop/Disconnect/exit | Start and Stop the gateway 10 times, read the card each time | Address gone after Stop; gateway starts every time. **Amends ADR-0125** |
| N10 | Every Run adds and every Stop removes the addresses | Keep approved addresses for the session; remove at project close or exit | 10 Run/Stop cycles: time and DNS Client CPU. Then kill the app and relaunch | Faster; nothing left after exit, and nothing left after a kill plus relaunch. **Amends ADR-0127 and ADR-0132** |
| N11 | Each address change needs a new approval and firewall rule (`netcfg/commands.go:56-59`) | One fixed per-program firewall rule | Change an address 5 times; count admin prompts | Fewer prompts; an outside client still connects. **New ADR (deferred by ADR-0136)** |

If a race (N3, N4, N5) does not show on today's code within its loop count, widen the window
once with a trial-only delay hook to prove the mechanism. If it still does not show, Drop.

## Run and Stop trials

| # | Finding in today's code | Idea | Test | Wins if |
|---|---|---|---|---|
| R1 | No IED process starts until addresses are ready (`runtime/supervisor.go:79-86`) | Start processes and load models while addresses are added; bind after | Click-to-running time, Lab card and real card | Faster; an independent MMS client reads the same values. **Amends ADR-0133 item 2** |
| R2 | Initial states and test modes go one IED after another (`runtime/run_start_writes.go:129-136`, `test_modes.go:158-175`) | All IEDs at once, first error kept in project order | Same timing; compare the event log | Faster; same events in the same order |
| R3 | Each IED is a new process that redoes the licence check; packaged builds also hash `studio_server.exe` (`native/common/license_guard.cpp:119-121`) | Spare, already-checked IED processes | Process-start share of Run, dev and packaged style | Run only if Step 0 shows this step is large. **New ADR** |
| R4 | Real-IED sessions stop fully before the workers (`runtime/stop.go:42-46`) | Stop both together | Click-to-stopped with one real session open | Faster; report blocks still given back (ADR-0067) |
| R5 | Address release runs before "stopped" (`runtime/stop.go:52-53`) | Send "stopped" first, release after, lock still held | Click-to-"stopped" on screen | Faster; addresses still removed; an immediate Run still works |
| R6 | Disconnect queues behind a connect (15 s) or discover (120 s) (`acquisition/session.go:207-229`) | Kill the busy worker instead | Stop while connecting to an address that does not answer | Stop within 2 s |
| R7 | Master Connect waits for the full interrogation, up to 60 s (`gateway/service/master_connect.go:42,95`, `master_ask.go:39`) | Return once connected; points arrive by events | Connect to an IEC 104 simulator with many points | Returns in seconds; every point still arrives |

## Screen trials (automated browser, 1366x768)

Paths are under `studio/frontend/src/`.

| # | Finding in today's code | Idea | Test | Wins if |
|---|---|---|---|---|
| U1 | Every action re-reads the whole project (`state/useStudio.ts:108-109`), including operate, test and set value, which do not change it | No re-read after runtime-only actions | Click-to-ready time on the 20-IED project | Faster; a second tab still shows the right state |
| U2 | One diagram edit triggers 2-8 extra requests (`state/useAutoSaveSwitch.ts:14`, `state/useHistory.ts:20-26`, hooks keyed on revision) | Key on revision; history and auto-save by push | Count requests after one drag | Fewer requests; Undo, Redo and the auto-save switch still correct |
| U3 | No lazy loading (`App.tsx:9-13`) | Load pages and the diagram library when opened; static spinner | Time to first paint; bundle size | Faster start. **Owner's eye:** no flash when switching pages |
| U4 | Point tables rebuilt whole each second (`features/tools/gateway/usePointList.ts:48`, `master/MasterPointTable.tsx:33`) | Apply the delta row by row | 5000-point gateway: browser CPU | Lower CPU; same rows and counts shown |
| U5 | Explorer refresh uses its own timer (`features/tools/explorer/useNodeReads.ts:48`); `/network/lab` and GOOSE messages still polled | Move onto the shared poller or push | Requests per minute when idle | Fewer requests; same values, 2 s refresh kept (ADR-0135) |

## Order

1. Step 0.
2. N1-N6 and U1: no decision changes, and they answer the owner's main complaints.
3. R2, R4, R5, R6, U2, U4, U5.
4. Trials that change a decision (N9, N10, N11, R1, R3, R7) and N7, N8, U3.

## Report at the end

One table: idea, today's number, the trial's number, runs, verdict and the reason. Then the list
for the owner's eye (N6 cable, U3 page switch, and anything else found on the way). Ideas that
amend a decision stay proposals until the owner agrees, even when they win. Record the results as
an evidence-only ledger entry and link the report from `docs/INDEX.md`.
