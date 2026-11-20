# Speed trials R1-R7: Run and Stop (2026-11-20)

Plan: [speed and network trials plan](speed-trials-plan-2026-11-20.md), ideas R1-R7. Summary
rows: [trial results](speed-trials-results-2026-11-20.md). Change
[0223](../changes/2026/0223-run-stop-trials.md), decision
[ADR-0142](../decisions/ADR-0142-spare-ied-processes-and-connect-without-waiting.md).

## How it was measured

- Owner's PC (Ryzen 9 5900HS, 16 logical processors), other agents' engines, builds and browsers
  running at the same time (machine 52-76 % busy over a set, our own workers included). Every
  trial ran against today's code **in alternating sessions** (base, trial, base, ...), each
  session a fresh engine instance (temp data folder, port 26000-26999, packaged behaviour, no
  `--dev`), so both sides saw the same load. Medians and worst of at least 10 counted rounds per
  side; rounds in which Windows showed our addresses vanish, or whose Run another engine refused,
  are listed and not counted.
- Today's code: `master` at 4b75e777 for R1 (before ADR-0141), `master` at 214e7a0b (with
  ADR-0141: IED addresses stay for the session) for everything else, workers from
  `studio/native/build/Release`. Trial workers were built from the worktree into
  `studio/native/build/trial-r1` (R1) and `trial-r3` (R3 and later; same code as Release plus the
  `hello` request), never into `Release`.
- Lab card addresses 198.18.26.x (env `SCADABENCH_TRIAL_RANGE`), loopback 127.67.26.x for the
  real-device and master trials.
- Tools (`studio/tools/trials/`): `runstop_ab.py` (Run/Stop, R1/R2/R3/R5, the Step 0 cut into
  steps), `real_stop_ab.py` (R4/R6, a separate `studio_ied` as the real device or a listener that
  never answers), `r7_master_connect.py` (R7, a separate `studio_gateway` as IEC 104 outstation),
  `r3_spare_cost.py` (spares' memory and processor time, licence-check parts). Raw rounds:
  `.local/trials/results/ab-*.json`, `real-*.json`, `r7-*.json`, `r3-cost-*.json` (not committed).
- Projects as Step 0: small = 8 IEDs, large = 20 IEDs, every IED on the Lab card.

## R1: load the models while the addresses are added (Dropped)

Built: a worker request `load` (parse the model, build the MMS server, no bind) and a Run that
started every process and loaded its model right after the GOOSE bus, while the addresses were
added and checked, binding afterwards. 10 rounds per side, alternating, before ADR-0141:

| Click to Run answer, median / worst s | today | R1 trial |
|---|---|---|
| small (8 IEDs), machine 70 % busy | 1.595 / 4.075 | 1.541 / 3.063 (-3 %) |
| large (20 IEDs), machine 74 % busy | 2.257 / 4.151 | 2.482 / 4.590 (+10 %) |

On the large project the address step grew from 0.446 to 0.554 s and the loads from 1.107 to
1.447 s: the processor is the limit on this PC, so loading 20 models while the service adds the
addresses slows both. The most R1 could hide is the address step (14-20 % of Run), and since
ADR-0141 later Runs add no address at all. **Drop**: not faster. The `load` request and the split
start were removed again; ADR-0133 item 2 is unchanged.

## R3: spare, already-checked IED processes (Consider, implemented)

While stopped the coordinator keeps one greeted `studio_ied` per simulated IED (ADR-0142); a Run
takes them. Alternating, 10 rounds per side, today = `master` 214e7a0b:

| | today | R3 | change |
|---|---|---|---|
| small: click to Run answer, median / worst s (52 % busy) | 0.884 / 1.201 | 0.647 / 0.795 | -27 % / -34 % |
| small: click to `running` event | 0.737 / 1.046 | 0.526 / 0.590 | -29 % |
| large: click to Run answer (76 % busy) | 3.116 / 5.869 | 1.167 / 2.763 | -63 % / -53 % |
| large: click to `running` event | 2.512 / 5.569 | 0.948 / 2.001 | -62 % |
| processes started by the Run | 8 / 20 every round | 0 (one large round: 19, its spares were not ready yet) | |

Where the time went (large, medians): today the first `studio_ied` appeared 0.715 s after the
click (worst 4.65 s) and the last 0.151 s later, the last IED listened 1.147 s after that; with
spares the last IED listened 0.922 s after the click. Starting 20 processes on this PC (antivirus
watching each new process, other agents' load) costs far more than Step 0's bare
`process_start.py` suggested. Stop was not slower (large 0.315 to 0.204 s, small 0.113 to
0.104 s median).

What the spares cost (`r3_spare_cost.py`, large project): 20 spares waiting, 0.82 MB private
memory each (16.5 MB in all), 0.000 processor seconds in 20 s. They are readied one after
another after Stop: on a machine 82 % busy, in 3 of 10 rounds a Run 3 s after Stop found only
part of them ready (it started 13, 14 or 20 processes itself, as today).

The packaged licence check (development workers skip it, so it is not in the numbers above) does
a machine ID read and a SHA-256 of `studio_server.exe` (26.5 MB) in each worker. On this PC: 20
`studio_license_check hwid` at once 0.223 s (8 at once 0.070 s), 20 hashes of 26.5 MB at once
0.131 s (8: 0.055 s). With spares that work is done before the Run; without them it is added to
every packaged Run. **Consider**: 27 % (small) and 63 % (large) faster, worst run better, nothing
the user sees or saves changes. Wire: unchanged (below).

### R2: initial states for all IEDs at once (Dropped)

Step 0 limited R2 to its initial-state half on the large project. Trial: one goroutine per IED,
first error kept in project order. Alternating, 10-11 rounds, both sides with R3 (82 % busy):

| Large project | today | R2 |
|---|---|---|
| "initial states" step, median / worst s | 0.087 / 0.397 | 0.003 / 0.011 |
| click to Run answer, median / worst s | 1.579 / 3.913 | 3.035 / 6.067 (noise: 3 rounds found spares missing) |
| order of the 198 `initial_state` events | the same every round (12 IED groups) | differs every round (13-65 interleaved groups) |

The step it shortens is 0.08 s, 3-6 % of Run, inside the noise of the total; and the events no
longer come in the same order, which the plan requires. **Drop**.

### R4: Stop with a real-IED session open (Consider, implemented)

Station engine: ReportTest as a real IED (report block reserved) on a separate `studio_ied` at
127.67.26.10:26102, and six simulated template IEDs (loopback). 10 rounds per side alternating
(sessions of 5), today = `master` 214e7a0b, trial = the final code (R3, R4, R6, R7, plus the R5
build of 2 ms, see R5):

| Click to Stop answer, median / worst s | today | R4 |
|---|---|---|
| Stop answer | 0.079 / 0.278 | 0.047 / 0.127 (-40 %) |
| `stopped` event | 0.076 / 0.272 | 0.046 / 0.126 |
| report block held while running (independent `studio_live`) | 10 / 10 | 10 / 10 |
| report block free after Stop | 10 / 10 | 10 / 10 |

**Consider**: faster, report blocks still given back (ADR-0067).

### R5: "stopped" before the address release (Dropped)

Built as specified (Stop answers after `stopped`, the release runs after it with the cycle lock
held). Since ADR-0141 Stop removes no IED address. Small project, 10 rounds each, both with R3:
click to Stop answer 0.099 / 0.389 s today, 0.097 / 0.254 s with R5; click to `stopped` 0.096
s on both; last worker gone to `stopped` 0.0174 s today, 0.0156 s with R5. The release now costs
about 2 ms. **Drop**: not faster.

### R6: Stop while connecting to a device that does not answer (Consider, implemented)

The real IED's address is a local listener (127.67.26.11:26102) that accepts and never answers;
Stop 1 s after Run, while the session shows "connecting". 10 rounds per side:

| Click to Stop answer, median / worst s | today | R6 |
|---|---|---|
| Stop | 4.074 / 4.231 | 0.045 / 0.109 |

Today Stop waited for the live client's connect attempt to time out ("disconnect" queued behind
it); while it reads a big data model it can wait up to 120 s. **Consider**: within 2 s. Unit test
`TestStopWhileConnectingKillsTheClient`: 15.0 s before, 0.02 s after.

### R7: SCADA master Connect with many points (Consider, implemented)

Engine in the SCADA master role, 5000 points (127.67.26.30:26404), 10 rounds per side in sessions
of 5, today = `master` 214e7a0b:

| Median / worst s | today | R7 |
|---|---|---|
| stand-in outstation that never answers the counter interrogation (`r7_station.py`): Connect answer | 15.117 / 15.181 | 0.073 / 0.125 |
| same: every point shown | 15.119 / 15.182 | 0.078 / 0.151 |
| same: Disconnect (once every point is there) | 0.104 / 14.645 | 0.008 / 0.016 |
| `studio_gateway` outstation (answers the counter interrogation): Connect answer | 0.461 / 0.804 | 0.260 / 1.172 |
| same: every point shown | 0.471 / 0.812 | 0.271 / 1.181 |

Today Connect waited t1 (15 s) for a counter-interrogation confirmation that never came, and the
first Disconnect of each session queued 14.6 s behind an interrogation still in flight. With our
own gateway as outstation Connect was already under a second (its worst round with R7 was one
1.17 s outlier on the busy PC). **Consider**: returns in well under a second, every point arrives.

All 5000 points arrived in every round on both sides.

## Wire and protocol checks

- `native_wire_compare.py Release trial-r3` (on 127.67.26.20-22): 5 MMS connections identical
  after masking (9570, 198, 746, 198, 758 bytes), probe outputs identical, 4 reports each, 14
  GOOSE frames identical in order: **WIRE IDENTICAL**. The only native change is the `hello` request.
- R4: the independent `studio_live` client saw the report block reserved while running and free
  after every Stop (20 of 20).
- Native protocol tests with `STUDIO_NATIVE_BIN=trial-r3` (`test_native_worker.py`,
  `test_native_reports.py`, `test_native_samples.py`, `test_live_client.py`): 16 passed.

## Owner checks

1. Packaged app (next release): open a project of about 7 and of about 20 IEDs, wait 10 s, press
   Run, Stop, Run three times; Run should feel quicker from the second Run on. Task Manager shows
   one idle `studio_ied` per simulated IED while stopped (about 1 MB each, 0 % CPU).
2. A real IED that is switched off or unplugged: Run, then Stop within 5 s; Stop should end at
   once (today it waited for the connect attempt).
3. SCADA master toward a real RTU or gateway: Connect should answer within a second; the points
   fill in by themselves; a converter started right after Connect serves every point.
