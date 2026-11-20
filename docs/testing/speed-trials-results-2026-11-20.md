# Speed and network trials: results (2026-11-20)

Plan: [speed and network trials plan](speed-trials-plan-2026-11-20.md). One row per idea: today's
number, the trial's number, runs, verdict and reason. Harness scripts: `studio/tools/trials/`.

## Screen trials U1-U5 (change [0224](../changes/2026/0224-screen-trials-u1-u5.md))

Set-up for every screen trial: two engine instances built from the same commit, today's build
(`studio_server.exe` + `vite build` of master) and the trial build, each with its own temp data folder
and port (22001, 22002) and the same project made by `u_setup.mjs`: 20 sample IEDs (Feeder01-20,
the built-in template) in loopback mode on their own MMS ports, 20 bound breakers and one IED
component on the SLD, 20 test points, and for U4 a 5000-point IEC 104 gateway list
(`u4_gateway_setup.mjs`: 300 IED points, 4700 own points, sine generators on the 20 measured IED
points). Headless Chrome at 1366x768 driven over the DevTools protocol (`u_cdp.mjs`); rounds
alternate today/trial (`u_rounds.mjs`). No Lab card address, no administrator prompt.

| Idea | Today | Trial | Runs | Verdict | Reason |
|---|---|---|---|---|---|
| U1 no project re-read after runtime-only actions | Set value (Test page): click to ready median 30/29.5/27 ms, worst 52/36/37, 5 requests a press (POST, GET project, test panel, auto save, push topics). SLD sheet Test button: 26.5/27 ms, worst 44/49, 4 requests | Set value: 20/21.5/25.5 ms, worst 38/35/42, 1 request (the POST). Test button: 20/18.5 ms, worst 34/30, 2 requests (the sheet's signal read + the POST) | 3 x 10 presses (value), 2 x 10 (test) per side | Consider (implemented) | 15-30 % faster to ready, 80 % / 50 % fewer requests; `u1_revision_check.mjs`: revision and project body unchanged after each runtime-only action, so nothing the re-read showed can be missed. Operate/tap kept as a re-read (they store the last position in the project) |
| U2 one diagram edit: fewer requests, history and auto save by push | 9 requests per drag (8 when the sheet already showed the IED), worst 9 | 2 (PUT diagram + the sheet's signals for a newly selected breaker), worst 2 | 2 x 10 drags per side | Consider (implemented) | 78 % fewer requests; Undo, Redo, the auto save switch and a second tab checked (`u2_still_correct.mjs`): all correct with the trial; today the second tab never showed the switch change |
| U3 lazy pages and diagram library | first paint median 156/176/164 ms, ready (Devices rows) 319/357/351 ms, JS before ready 1 175 kB | first paint 132/136/124 ms, ready 494/363/318 ms (worst 880 vs 486), JS 456 kB | 3 x 5 cold starts per side | Drop | Usable start not faster (ready median not better, worst worse); first SLD visit slower until the background load ends (81-140 vs 39-50 ms); and loading page stylesheets later changed the look (`u3_same_look.mjs`: SLD zoom buttons 15 to 20 px margin and another shadow, a Test page line 8 px higher). Kept as today |
| U4 point tables: apply the change row by row | Gateway point list (5000 points, 20 changing each second, IEDs running): browser main thread 19.2/19.3/20.5 ms per second (worst 22.4), script 13.2/13.5/14.6 ms/s | 10.3/11.6/11.8 ms/s (worst 13.8), script 5.5/6.0/6.1 ms/s | 3 rounds x 3 samples of 20 s per side | Consider (implemented) | About 42 % less main-thread time and 56 % less script time; the same 100 rows, filter counts and first rows shown, same HTTP request count (33-34 in 60 s) |
| U5 lab status, GOOSE messages and Explorer onto the push stream / shared poller | Idle requests a minute: Settings with the Lab card chosen 20 (`GET /network/lab`), SLD with an IED sheet open while running 30 (`GET /ieds/{id}/goose/messages`), Devices 0, Test 0. Explorer: 30 a minute (2 s, ADR-0135) | Lab card 0, IED sheet 0, Devices 0, Test 0 (the answers come on the push stream). Explorer unchanged, by choice | 2 rounds x 60 s per page per side | Consider (implemented) for lab status and GOOSE messages; Explorer part Drop | The lab and messages answers still change on screen as before (push re-reads `/network/lab` every 3 s while shown and the messages on value changes, at most once a second). Explorer: ADR-0135 fixes the 2 s read, its timer already skips hidden windows, and the shared poller would add a read whenever the window shows again, so nothing can be saved. The setup wizard's 3 s `/network/adapters?refresh=1` (20 a minute) is outside U5 and left as is |

### Found on the way (U2)

- **Undo right after a drag did nothing the first time** (today's code, also before this change):
  Undo saves the drawing first, and that second identical PUT made the engine record an empty
  step, so the first Undo took back nothing and the second one took back the drag. Fixed in 0224
  (`diagramSave.ts` sends nothing when nothing changed); `u2_still_correct.mjs` now shows the
  breaker back in both tabs after one Undo.
- **A background tab does not follow the auto save switch on today's code** (it keeps "All
  changes saved" after the other tab turned auto save off); fixed by the U2 push topic.
- The network setup page still asks `/network/adapters?refresh=1` every 3 s while no card is chosen
  (20 requests a minute); it re-reads Windows' card list and is outside U5. Left as is.

### Owner's eye

- Nothing to look at for U1, U2, U4, U5: no visible change. U3 was dropped, so page switching is
  exactly as today.
