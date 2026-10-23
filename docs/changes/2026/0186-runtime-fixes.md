# 0186 Runtime fixes: derived Beh, large mode changes, log volume, times, origin, words

Date 2026-10-24. Go engine + native worker. Decision
[ADR-0115](../../decisions/ADR-0115-simulated-ied-runtime-fixes.md), spec
[test-mode](../../specs/test-mode.md) (TMODE-05, -10, -13 to -17). Branch `runtime-fixes-0186`. No
Python code changed; the native report and time changes reach the frozen Python engine too.

## Problem (owner-approved findings of the 24-vendor-IED runtime test)

H3 Beh never derived from Mod; test mode refused on a large LD (256-attribute batch); M3 run log
about 10 MB/min with no cap; a >2 MB upload while running ended in a TCP reset and preview was refused
while running; L1 out-of-range Enum and non-ASCII VisString writes reached the wire; L3 SLD and plain
commands without origin; L4 1970 timestamps with good quality and two TimeQualities; L6 wrong words;
L7 hard-coded application version in the run log.

## Change

- Beh (`runtime/behaviour.go`, `test_modes.go`, `feedback.go`, `sample_writer.go`): derived from the
  LN's Mod and `LLN0.Mod` at every Run (from the imported values the restarted worker holds, after
  saved settings), in the same batch as every Mod write, and after an MMS client's Mod command. LDs
  without a valid `LLN0.Mod` are left alone. Run start 2.7-3.2 s against 2.9-3.3 s on master (same
  project, back to back; the Beh batch of 1172 values takes about 0.3 s).
- Batches (`runtime/sample_batches.go`, native `worker/updates/values.cpp`): native limit 256 → 4096;
  above it Go splits per point.
- Log (`runtime/runlog.go`, `runlog_summary.go`, native `worker/server/server.cpp`): generator samples
  become one `signal_summary` per IED and second with `from_id`/`to_id`; no `report_control` kind 9 for
  RCBs no client enabled; `run_started.application` is the app version (`Options.Version`).
  `tools/query_runtime_logs.py --ref` matches summaries.
- Times (native `server.cpp`, `values.cpp`; Go `start_times.go`, `supervisor.go`): every `t` starts at
  the Run start time; one TimeQuality 0xAA for commands and writes.
- Origin (native `control/canvas_command.cpp`): orCat 3, orIdent "ScadaBench".
- Checks and words (`runtime/signal/sample_rules.go`, `model.go`, `values.go`, `stop.go`): Enum ords
  outside the EnumType and non-ASCII VisString text refused; quality words; "Unknown IED" for an
  unknown stop. Status codes unchanged (400 `not_found` is the API convention and Python parity).
- Uploads (`api/projectapi/api.go`, `scl_routes.go`): refused bodies drained; preview while running.

## Evidence

Independent client `mmscli` (libiec61850 client API only), project B (`py61850_mixed_Siemens_SEL.scd`:
QMA1_MU1, QPC1_UPC1, QPC1_UPC4, TR01_2440), app on 127.0.0.1:8912/8913, IEDs on ports 22100-22103,
fresh data folders; the owner's IEDs on 127.50.0.x:102 untouched. Before = master f3ffef9f built
separately, after = this branch.

| Item | Before | After |
| --- | --- | --- |
| 1 Beh at Run (`QMA1_MU1Bay1_CB1`, Mod 1) | LLN0/XCBR1/CSWI1 Beh 0 | 1, 1, 1 (GAPC2 Mod 0 → follows LD: 1) |
| 1 client operates `Application/LLN0.Mod`=3 | Beh 0, LNs 0 | LLN0 3, LNs 3 |
| 1 LD 1 + GOGGIO1.Mod 3 / LD 2 + LN 3 | Beh unchanged | LN 3 / LN 4, LD 2 |
| 2 test mode on 148-LN LD | 400 "exceeds the native 256-attribute limit" | 200, Mod 3, Beh 3 |
| 4 2-60 MB import while running (urllib) | reset or abort in 13 of 15 (two 2 MB tries answered) | 400 JSON 15 of 15 |
| 4 preview while running | 400 "Stop the simulation…" | XML verdict |
| 5 `XCBR1.Mod.stVal`=9 | 200, wire 9 | 400 "9 is not a choice…", wire 1 |
| 5 VisString "Café µ" | 200, wire UTF-8 bytes | 400 plain words, wire unchanged |
| 6 SLD close/open, plain operate | orCat 0, orIdent empty | orCat 3, orIdent "ScadaBench" |
| 7 untouched `t` at Run | 0 (1970), q=00 | Run start time, q=aa |
| 7 time quality command / signal write | 8a / 0a | aa / aa |
| 8 bad quality / stop unknown IED | "This signal requires an integer" / "This IED is not running" | quality words / 400 `not_found` "Unknown IED" |
| 9 `run_started.application` | 0.1.0 | 0.1.24+dev |

Log volume, same load (6 generator sources at 10 Hz on dataset members, 4 report clients, 120 s):
1.787 MB/min before, 0.055 MB/min after; `report_control` kind 9 702 → 0, generator lines 8147 → 266
summaries. Native batch round trip: 2.6 ms (256), 10.4 ms (1024), 40.2 ms (4096 attributes).

## Limitations

- An MMS client's Mod command updates Beh as a second write a few milliseconds later (Studio's own
  Mod writes are one batch). `GrRef` LD hierarchies are not followed.
- LDs whose `LLN0.Mod` is 0 (Enum without `Val`) keep their imported Beh until the import fix (the
  parallel import branch) gives them a valid mode or a client sets one.
- "ExtRef without intAddr" wording lives in `acquisition/goosedom/`, owned by the import branch; not
  changed here.
- TimeQuality says "clock not synchronised" always; Studio cannot know whether the PC clock is synced.
- Slice re-run: project B plus ABB `MNSREF615_1.cid` (imports on this branch) and GE
  `CritBench_GE_F60.cid` ran together; every LD with a valid `LLN0.Mod` read Beh = Mod, and the sampled `t`
  of each IED read the Run start time (or its restored last change) with q=aa, none 1970. The full 24-IED sweep was not repeated.
