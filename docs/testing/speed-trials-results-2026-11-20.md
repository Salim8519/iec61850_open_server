# Speed and network trials: results (2026-11-20)

Plan: [speed and network trials plan](speed-trials-plan-2026-11-20.md). Harness:
`studio/tools/trials/` (see its README for every command). Raw rounds:
`.local/trials/results/*.json` on the owner's PC (not committed).

## Step 0: today's code, measured first

**Code:** `master` at 3411ecfb (0.1.35 source), Go engine rebuilt from it, development native
workers (`studio/native/build/Release`, licence check off). The engine ran without `--dev`
(packaged behaviour) as a separate instance: own data folder, own `LOCALAPPDATA`, HTTP port
20000, MMS port 20102. No product code was changed.

**Projects:** small = the 8-IED panels project (134 diagram nodes, mapped breakers and
switches); large = 20 IEDs (the 14-IED 132 kV project and 6 IEDs of the 400 kV project, 203
diagram nodes). No IED had test modes to write, so "test modes" is 0 in every round.

**Card:** every IED on the ScadaBench Lab card through `POST /network/lab/use`. The brief asked
for 127.67.x.y, but the Lab card refuses every loopback (127.x) address ("loopback (127.x) ...
addresses cannot be used"). Sets A-C used 10.67.0.x (no card or route on this PC uses it);
afterwards the lead gave each trial its own /24 of the 198.18.0.0/15 test range, and the harness
now uses 198.18.20.x (checked: 2 rounds, 8 of 8 addresses on and off the card, same timings).
Real cards: Owner checks below.

**How a round is cut into steps:** see the table in `studio/tools/trials/README.md`. Each step
ends at a moment Windows or the engine recorded (lab journal, Windows' address list with its
own creation times, the listener table, kernel process times, the engine's events), so the
steps add up to the click-to-answer total. Medians of each step are taken separately, so they
do not add up exactly to the median total. A round = Run, 2 s running, Stop, 3 s stopped; the
first round starts 8 s after the project opened.

### Run and Stop, set A (the reference: least load; median / worst seconds, share of median total)

Sets of 12 rounds (GOOSE on) and 10 rounds (`--goose off`: every IED's GOOSE publishing off,
which leaves the listen check alone in "listen check + GOOSE bus"). Machine 68-75 % busy over a
set, our own workers included.

| Run step | small, GOOSE on | large, GOOSE on | small, GOOSE off | large, GOOSE off |
|---|---|---|---|---|
| addresses | 0.183 / 0.336 (14.5 %) | 0.328 / 0.741 (13.7 %) | 0.156 / 0.266 (13.9 %) | 0.354 / 0.409 (14.1 %) |
| listen check + GOOSE bus | 0.065 / 0.295 (5.1 %) | 0.116 / 0.167 (4.9 %) | 0.048 / 0.149 (4.3 %) | 0.092 / 0.175 (3.7 %) |
| process start | 0.062 / 0.327 (4.9 %) | 0.267 / 0.517 (11.2 %) | 0.060 / 0.165 (5.3 %) | 0.315 / 0.847 (12.6 %) |
| model load | 0.724 / 1.794 (57.6 %) | 1.208 / 1.491 (50.6 %) | 0.654 / 0.908 (58.4 %) | 1.326 / 1.643 (53.0 %) |
| saved values | 0.012 / 0.047 (0.9 %) | 0.013 / 0.066 (0.6 %) | 0.007 / 0.047 (0.7 %) | 0.009 / 0.051 (0.4 %) |
| first values read | 0.053 / 0.168 (4.2 %) | 0.160 / 0.211 (6.7 %) | 0.040 / 0.059 (3.5 %) | 0.126 / 0.195 (5.0 %) |
| initial states | 0.039 / 0.072 (3.1 %) | 0.145 / 0.193 (6.1 %) | 0.029 / 0.051 (2.6 %) | 0.104 / 0.177 (4.2 %) |
| test modes | 0 / 0 (0 %) | 0 / 0 (0 %) | 0 / 0 (0 %) | 0 / 0 (0 %) |
| rest (listeners, reply) | 0.058 / 0.175 (4.6 %) | 0.191 / 0.240 (8.0 %) | 0.042 / 0.131 (3.8 %) | 0.147 / 0.207 (5.9 %) |
| **click to Run answer** | **1.258 / 2.701** | **2.387 / 3.288** | **1.120 / 1.529** | **2.500 / 2.900** |
| click to `running` event | 1.110 / 2.425 | 1.877 / 2.720 | 1.010 / 1.288 | 2.023 / 2.579 |

| Stop step | small, GOOSE on | large, GOOSE on | small, GOOSE off | large, GOOSE off |
|---|---|---|---|---|
| workers stop | 0.118 / 0.475 (64.4 %) | 0.201 / 0.421 (71.0 %) | 0.131 / 0.234 (63.6 %) | 0.189 / 0.461 (61.2 %) |
| address release | 0.041 / 0.243 (22.1 %) | 0.107 / 0.150 (37.7 %) | 0.079 / 0.150 (38.6 %) | 0.124 / 0.406 (40.2 %) |
| stopped shown (`runtime` `stopped` event) | 0.001 / 0.034 (0.3 %) | 0.000 / 0.017 (0 %) | 0.000 / 0.001 (0 %) | 0.001 / 0.058 (0.3 %) |
| reply | 0.002 / 0.093 (1.1 %) | 0.002 / 0.013 (0.7 %) | 0.001 / 0.043 (0.7 %) | 0.002 / 0.007 (0.6 %) |
| **click to Stop answer** | **0.183 / 0.721** | **0.283 / 0.497** | **0.206 / 0.429** | **0.308 / 0.663** |
| click to `stopped` event | 0.179 / 0.719 | 0.278 / 0.495 | 0.204 / 0.386 | 0.304 / 0.656 |

"Stopped" reaches the screen only after the address release: the `stopped` event comes after
it, and the page reads the project again after the Stop answer (R5, U1).

Set A was taken before the harness could see other engines remove our addresses (see the
finding below); their journals show such removals in 3 of its rounds (05:25:00, 05:25:40
and 05:30:43 UTC), which can only shorten "address release" in those rounds.

### Same test under heavier load, set C (machine 83-97 % busy)

Rounds in which Windows showed our addresses taken off while running (another engine, see the
finding below) are left out: small GOOSE on 9 of 12 counted, large GOOSE on 11 of 12, small
GOOSE off 10 of 10, large GOOSE off 5 of 10.

| Median s (share) | small, GOOSE on | large, GOOSE on | small, GOOSE off | large, GOOSE off |
|---|---|---|---|---|
| click to Run answer, median / worst | 1.799 / 4.227 | 4.219 / 7.207 | 1.568 / 2.265 | 4.991 / 6.677 |
| addresses | 0.341 (19.0 %) | 0.674 (16.0 %) | 0.216 (13.8 %) | 0.721 (14.4 %) |
| process start | 0.096 (5.3 %) | 0.354 (8.4 %) | 0.084 (5.4 %) | 0.490 (9.8 %) |
| model load | 0.937 (52.1 %) | 1.806 (42.8 %) | 0.779 (49.7 %) | 2.233 (44.7 %) |
| first values read | 0.088 (4.9 %) | 0.177 (4.2 %) | 0.072 (4.6 %) | 0.326 (6.5 %) |
| initial states | 0.074 (4.1 %) | 0.172 (4.1 %) | 0.042 (2.7 %) | 0.237 (4.7 %) |
| click to Stop answer, median / worst | 0.303 / 0.392 | 0.468 / 0.983 | 0.366 / 0.914 | 0.363 / 2.085 |
| address release (share of Stop) | 0.090 (29.6 %) | 0.191 (40.8 %) | 0.082 (22.3 %) | 0.127 (34.9 %) |

The shares stay in the same ranges under both loads; the seconds do not. A set B (machine
82-98 %, before the disturbance check existed) agreed with set C and is kept only as raw data.

### Process start alone (`process_start.py`, 10 repeats)

N `studio_ied.exe` started at once, each timed until it answers its first request (no model,
no licence check): 8 at once 0.184 s median (worst 0.221), 20 at once 0.409 s (worst 0.536)
under set A's load; 0.316 / 0.674 s and 0.541 / 0.769 s under set B's load. This time is
inside "model load", so process start in total is about 0.24 s of 1.26 s (19 %) for the small
project and 0.68 s of 2.39 s (28 %) for the large one.

### GOOSE bus and listen check

GOOSE on minus GOOSE off, same project: 0.017 s (small) and 0.024 s (large) in set A, 0.017 s
and 0.056 s in set C. The GOOSE bus wait is at most 0.06 s (under 2.5 %); the listen check
alone is 0.05-0.09 s (1-4 %).

### Compared with change 0208 (2026-11-01)

0208 measured the packaged app on the owner's 7-IED project: Start 2.5-4.1 s, Stop 0.32-0.80 s.
Today, development workers on the Lab card: small (8 IEDs) Start 1.26 s median, Stop 0.18 s;
large (20 IEDs) 2.39 s and 0.28 s. Different project, workers and load, so this is no
before/after claim.

## Which steps cost under 5 % (the plan skips their trials)

A step counts as "under 5 %" when its median share stayed under 5 % in both projects in
sets A and C.

| Step | Share range | Plan trial that targets it | Decision |
|---|---|---|---|
| test modes | 0 % (no project had test modes to write) | R2, test-mode half | **skip the test-mode half of R2** |
| saved values | 0.4-2.6 % | none | nothing to skip |
| GOOSE bus (GOOSE on minus off) | under 2.5 % | none (already overlapped, ADR-0133) | nothing to skip |
| listen check | 1-4 % | none | nothing to skip |
| stopped shown, reply (Stop) | 0-1.3 % | none | nothing to skip |
| initial states | 2.6-6.2 % | R2, initial-state half | **run R2 on the large project only** (over 5 % only there); at most ~0.15 s to win |
| first values read | 3.5-6.7 % | none (already all IEDs at once, ADR-0129) | nothing to skip |
| addresses | 13.7-19.0 % | R1, N10 | run |
| process start (with its part of model load) | 19-28 % | R3 | run |
| address release | 22-41 % of Stop | R5, N10 | run |

So no whole trial is skipped; R2 shrinks to its initial-state half on the large project.
R4 (Stop with a real-IED session), R6, R7, N1-N11 and U1-U5 are not judged by Step 0 shares:
Step 0 had no real-IED session, master or screen, and the N trials count faults, not seconds.
The biggest step, model load (43-58 %), has no trial of its own in the plan apart from its
process-start part (R3) and the overlap with addresses (R1).

## Machine load and what else ran

- Ryzen 9 5900HS, 16 logical processors, Windows 11 Pro 26300.
- Before the sets the machine was 24 % busy with nothing of ours running (Explorer 1.5-2.2
  processors, OneDrive sync about 1, Claude about 1, search indexer, Kaspersky). A later 30 s
  idle check read 85 %.
- **Other agents ran on this PC at the same time**, against the brief's one-after-another rule:
  from 09:15 another worktree's engine (port 22001, 20 IEDs running on 127.50.0.x), from 10:02
  a second one (port 22002), and a third agent's benchmark engines (ports 23030/23031, temp data
  folders) started and stopped many times, with their builds and browsers. Set A ran 09:23-09:36,
  set B 09:58-10:05, set C 10:17-10:24 (local time). The harness now refuses to start while
  another `studio_server.exe` runs or the Lab card holds addresses; sets B and C used the
  override (`--shared`) and each result file lists the other engines (`other_engines`).
- Each set's file records the machine's busy share over the set (`machine_busy_percent`).

## DNS Client after address changes (Lab card)

`dnscache_cpu.py --repeats 3` (small project): CPU of the DNS Client's own svchost for 30 s
after each action; median of 3 / worst, then median CPU seconds and the busiest half-second.

| After | Average processors | CPU s in 30 s | Busiest 0.5 s |
|---|---|---|---|
| nothing (idle baseline) | 0.04 / 0.05 | 1.1 | 0.41 |
| one IED started alone (1 address added) | 0.28 / 0.29 | 8.5 | 1.91 |
| that IED stopped (1 address removed) | 0.14 / 0.16 | 4.1 | 1.31 |
| Run (8 addresses added) | 0.31 / 0.32 | 9.2 | 2.81 |
| Stop (8 addresses removed) | 0.09 / 0.29 | 2.7 | 3.31 |

On the Lab card the DNS Client stays far below change 0208's 5-11 processors for 20-30 s (seen
on Ethernet). Other engines changed Lab-card addresses during these samples, so part of this
CPU is theirs. The real-card case is an Owner check.

## Finding on the way: another engine takes our addresses off the Lab card

Every engine's Release (at launch, after every Stop and after Use; ADR-0127) takes off the Lab
card **every** address that engine does not use itself (`labsetup/release.go` `releaseLab`; the
journal filter of change 0220 A6 covers real cards only). Two engines on one PC therefore remove
each other's IED addresses. Tonight the other agents' engines did it to ours: their lab journals
record `removed` for our 10.67.0.x addresses at 05:25:00, 05:25:40, 05:27:43, 05:28:09,
05:29:51, 05:30:43, 06:06:38 and 06:08:56 UTC (several of their data folders were deleted
already, so there were likely more). What Windows and the engine showed:

- **Run refused with "MMS bind failed: 10.67.0.x:20102"**: 1 of 12 large rounds in the first
  try (at 05:29:51, the moment another engine's journal removed our 20 addresses), 3 of 10 in
  set B's large GOOSE-off set, 1 of 12 (small) and 3 of 10 (large GOOSE off) in set C. In set C's
  large refusals Windows lists all 20 addresses gone 2.1-4.2 s after the click, while the
  workers were still loading their models; the Run answered the error 4.8-12.4 s after the click.
- **Addresses removed while running** (the IEDs keep running but cannot be reached): 2 of 12,
  1 of 12 and 2 of 10 rounds in set C.

In the product the same happens whenever two ScadaBench engines run on one PC, for example the
installed app and a development app, which use different data folders and so do not stop each
other: one app's IEDs fail to start or become unreachable "sometimes". This fits the owner's
"sometimes we don't understand what happened" and belongs with N1/N3: a Release should remove
only addresses its own journal added (as A6 does for real cards), and a refused bind should say
that the address left the card. Not changed here (Step 0 changes no product code); the lead
has put "remove only addresses this copy journaled" in a later group. Until then every timing
round must check Windows for addresses that vanished; `step0_run_stop.py` leaves such rounds
out and lists them (`disturbed_rounds`, `failed_runs` with Windows' address changes).

## Owner checks (real cards)

Wi-Fi is the owner's DHCP card and Ethernet is unplugged, so no real-card number exists yet.

1. Close every ScadaBench window (see the finding above). Plug the Ethernet cable in. Open the
   dev app (`start-studio.cmd`) with a project of about 7 IEDs on Ethernet with approved
   addresses (ADR-0132), stopped.
2. In a terminal at the repo root: `python studio/tools/trials/card_watch.py --card Ethernet
   --seconds 300`. It only reads Windows: every address added or removed on the card with its
   time, and the DNS Client's processors each second (printed above 0.2).
3. Click Run, wait 30 s, click Stop, wait 30 s; three times. Note how long each Run and Stop
   took on screen.
4. Send `.local/trials/results/card-watch-*.json` and the noted times. Repeat with about 20
   IEDs if such an Ethernet project exists.

## Results for the ideas

Filled in by each trial (same test on today's code and on the trial, at least 10 runs each,
median and worst; verdict per the plan's table).

| Idea | Today's number | Trial number | Runs | Verdict | Reason |
|---|---|---|---|---|---|
| N1 | | | | | |
| N2 | | | | | |
| N3 | | | | | |
| N4 | | | | | |
| N5 | | | | | |
| N6 | | | | | |
| N7 | | | | | |
| N8 | | | | | |
| N9 | | | | | |
| N10 | | | | | |
| N11 | | | | | |
| R1 | | | | | |
| R2 | | | | | |
| R3 | | | | | |
| R4 | | | | | |
| R5 | | | | | |
| R6 | | | | | |
| R7 | | | | | |
| U1 | | | | | |
| U2 | | | | | |
| U3 | | | | | |
| U4 | | | | | |
| U5 | | | | | |

## Limitations

- Development workers: the packaged licence check and the hash of `studio_server.exe` that
  packaged workers do at start (R3) are not in these numbers.
- Every set ran while other agents' engines and builds used the same PC (machine 67-98 % busy
  over a set, our own workers included); timings move with that load, so later trials must
  measure today's code again next to the trial, as the plan says.
- No project had test modes to write and none had a real-IED session, so "test modes" and the
  real-session part of Stop (R4) are not measured here.
