# 0138: Stress test on the owner's laptop: Python (old/new natives) vs composed Go

Date: 2026-10-20. Owner: "stress test, see the long running"; stopped by the owner after the 2 h Go
soak and 19 min of the Python soak. Bench: `studio/tools/benchmark_engines.py` + `engine_bench`
(scenarios from [0123](0123-stress-bench-scenarios.md)). Evidence JSON/JSONL kept in the session
scratchpad (`stress2/out`), not in the repository.

## Setup

Ryzen 9 5900HS, 23.4 GB RAM (3.0-4.0 GB free). Synthetic SCL, 2000 points per IED, 127.66.0.x:20700,
temp data folders, kill-on-close jobs. OLD natives = copy of `studio/native/build/Release`; NEW =
merged master natives (`build/integrate2`: ADR-0079, 0088, 0089, N3, control_mux_v1) + master
`studio_goose`. Go = composed `studio_server` from master, XSD verdicts recorded from Python
(`STUDIO_XSD_SEED`; `studio_xsd.exe` not built). Breaker model 4 (SBO enhanced) for hammer and soak.

## Results

| Measure | Python OLD | Python NEW | Go NEW |
|---|---|---|---|
| App ready | 0.98-1.03 s | 0.91-1.03 s | 0.57-0.67 s |
| Import 20 IEDs / open 20 IEDs | 1.18 / 0.75 s | 1.11 / 0.73 s | 0.41 / 0.38 s |
| Run 1 / 5 / 10 / 20 IEDs | 0.20 / 0.62 / 0.88 / 1.69 s | 0.13 / 0.45 / 0.86 / 1.69 s | 0.14 / 0.20 / 0.32 / 0.59 s |
| Stop 20 IEDs | 0.33 s | 0.19 s | 0.08 s |
| Operate p50 / p99, 0 idle MMS clients | 142 / 204 ms | 142 / 174 ms | 47 / 70 ms |
| Operate p50 / p99, 3 idle MMS clients | 523 / 554 ms | 142 / 163 ms | 47 / 79 ms |
| Operates per second (500 hammer, 0 errors all) | 6.7 (1.8 with idle clients) | 6.8 | 17.6 |
| 50 Run/Stop at 10 IEDs, Run p50 / Stop p50 | | 0.85 s / 89 ms | 0.31 s / 32 ms |
| App shutdown (Ctrl+C) until natives gone | | 1.27-1.34 s | 0.17-0.18 s |
| Backend CPU, 20 IEDs running | 0.9 % | 2.3 % | 0.11 % |
| Backend RAM, 20 IEDs running | 126 MB | 127 MB | 205 MB |

- Every operate confirmed by `studio_probe` (50/50 per run); no leftover workers or ports after any
  Stop or app shutdown (0 errors in 50 Run/Stop and 10 app cycles, Ctrl+C and kill).
- Soak, Go NEW, full 2 h: 20 IEDs, ~1000 signals/s, operate every 5 s, push client + `/health`
  every 2 s. 1440 operates, 0 errors, no crash, no worker restart, no stream disconnect. Operate p50
  58.0 ms first 10 min, 58.4 ms last 10 min. Backend +0.12 MB/h working set, +6.4 handles/h; IEDs
  +25 MB in the first 30 min then flat. Backend CPU 15 % average under that load.
- Soak, Python NEW, partial 19 min, same load over `/events`: 0 errors; operate window median
  ~175 ms; backend CPU 43.6 % average (about 3x Go); event lag p99 up to 108 ms (Go <= 20 ms).

## Findings to act on

1. Go backend RAM per 2000-point IED ~7.4 MB vs Python ~2.7 MB (190 vs 121 MB at 20 IEDs stopped).
2. Go keeps `studio_control.exe` (+conhost, ~12.5 MB) after Stop (exits at app shutdown); Python
   has none. Bench "natives left" also matches `studio_server` itself (bench artefact).
3. Go backend handles +100 over the first 25 Run/Stop cycles, then level (400-408).
4. Data folder grows 1.5-1.7 GB/h at 1000 signals/s on both engines (append-only runtime event log,
   [ADR-0020](../../decisions/ADR-0020-append-only-evidence-ledgers.md), no size limit by design) - an owner decision on retention.
5. Go event lag ~16 ms by the push batching window (ADR-0087); worst case better than Python.

## Not run

Browser click-to-paint during the soak (no token-free sign-in for the built-in browser); Python
soak cut at 19 of 45 min.

## Bench changes (this commit)

`engine.py` (Go binary path `studio/build/engine/studio_server.exe`; `output(n)` hides the session
token), `scenarios_lifecycle.py` (redacted log tail; before, evidence could hold the token),
`fixtures.py` + `benchmark_engines.py` (`--ctl-model`), `soak_feed.py` + `scenarios_soak.py` +
`benchmark_engines.py` (`--push-topics`). Files from earlier runs that held a session token were
deleted from the scratchpad; a rescan finds none.
