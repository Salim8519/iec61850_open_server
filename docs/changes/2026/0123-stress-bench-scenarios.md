# 0123: Stress benchmark scenarios and Python baseline (scale, hammer, lifecycle, soak)

Date: 2026-10-20. Owner asked for a real stress test of ScadaBench on this laptop; stopped by the
owner after the first real run because the speed work comes first. The tools stay for the
before/after comparison.

## Change

- `studio/tools/benchmark_engines.py`: scenarios SCALE, HAMMER, CYCLES, APPCYCLES, SOAK; options
  `--base-port`, `--idle-clients`, `--probe-every`, `--levels`, `--cycles`, `--interval`,
  `--operate-every`, `--stream`. Pass `--base-port 20700` (or higher) on this laptop.
- `studio/tools/engine_bench/`: `canvas.py` (canvas operate endpoint), `machine.py`, `mms_idle.py`
  (real idle MMS associations through a relay), `pertype.py`, `scenarios_{scale,hammer,lifecycle,
  soak}.py`, `shutdown.py`, `soak_feed.py`, `soak_report.py`; `engine.py` gains `popen_options`.

## Evidence (Python + current natives, temp data folder, 127.66.0.x:20700, 2000 points/IED)

| IEDs | App ready | Import | Open | Run | Stop | Backend RAM running | Backend CPU | studio_ied each |
|---|---|---|---|---|---|---|---|---|
| 1 | 0.94 s | 0.08 s | 0.05 s | 0.15 s | 0.02 s | 68.4 MB | 0.45 % | 9.3 MB, 1.8 % |
| 5 | 1.06 s | 0.28 s | 0.19 s | 0.59 s | 0.06 s | 80.7 MB | 1.0 % | 9.3 MB, ~2 % |

No leftover workers or ports after Stop. Each worker also has a ~7.7 MB `conhost.exe`.
Smoke-only (tool checks, not measurements): canvas operate with 3 idle MMS clients ~308 ms mean;
60 s soak with 200 signal events/s: SSE lag p50 ~2 ms, data folder +~280 MB/hour from the
append-only event log (no size limit by design).

## Limitations

- Not run: SCALE 10/20 IEDs, hammer OLD vs NEW natives, 50 Run/Stop and 10 app cycles, 2 h soak,
  browser click test. `scenarios_wire.py` S4 "confirmed by probe" compares Dbpos text with numbers
  and always reports 0 (left unchanged; fix when S4 is next used).
