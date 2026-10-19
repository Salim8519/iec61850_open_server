# Capacity, RAM and CPU audit for the Go engine plan

Date 2026-10-19. Machine: Ryzen 9 5900HS, 16 logical CPUs. Synthetic template IEDs only (no customer SCL).
Isolation: own copies of the native binaries in `bin/`, temp project/cfg dirs in this folder,
addresses 127.61.x.y:18620 and 127.62.0.x:18630 (owner's app on 127.50.0.2-7:102 untouched; all test
processes confirmed gone afterwards). Raw data: `mem_per_point.json`, `worker_probe2.json`,
`realistic36k.json`, `coord_run.jsonl`, `micro.json`; scripts alongside.
**M** = measured here, **R** = measured in repo records (0006/0011/0109/0110/0111), **E** = estimate.

## 1. Measured coefficients

| Quantity | Value | |
|---|---|---|
| Python signal dict (11 keys) | 536 B container-only; 682-729 B incl. strings in the imported model | M |
| Python coordinator process, per point (model + 2 indexes) | ~1.06 KB (47 MB @2.5k pts -> 97.5 MB @50k pts) | M |
| Python full app, per point (owner project) | ~2.8 KB (280-320 MB @90k; includes SCL text, caches, Flask) | R |
| studio_ied private bytes | 2.6 MB + 0.40 KB/pt (3.6 @2.5k, 8.3 @15k, 16.4 @36k) | M |
| studio_ied working set | 8.3 MB + 0.38 KB/pt (9.5 @2.5k, 13.9 @15k, 21.5 @36k) | M |
| studio_ied start (serial) | 0.11-0.12 s each (20 IEDs: 2.2-2.6 s Run); 0.42 s @36k realistic | M |
| studio_ied idle CPU | 0.6-2 % of a core alone; **27 % of one core for 20 idle workers** | M |
| Native IPC update_batch(80) | 63-71k attr/s per worker, RTT p50 1.1 ms, p99 2.0 ms; worker 11-13.5 us/attr, Python 3.2-3.9 us/attr | M |
| MMS client held, server side | +0.2-0.3 MB private per connection | M |
| **MMS read latency vs connected clients** | 0.15 ms/ref (1), 10.4 (2), 20.9 (3), **41.5 ms/ref (5)** | M |
| studio_live (real-IED client) | 9-12 MB WS each; discover 2.5k pts 0.15 s; read 500 refs 75-80 ms | M |
| Python end-to-end per generated update | ~75-80 us (10k upd/s = 73-82 % of a core) | M |
| GIL ceiling, no browser | ~13.5k upd/s (5000 sources asked 50k/s, got 13.5k, Python 106 %) | M |
| **With 2-3 SSE clients** | 1000 sources (10k/s asked) -> only **1.4-1.6k/s** delivered, Python 100 % | M |
| EventBus.after scan (5000 deque) | 0.18 ms per wake per client; notify_all wakes every client per event | M |
| Gateway diff, 5000 points | 8.9 ms per 100 ms tick = 9 % of a core under the GIL | M |
| json.dumps(signal event) / publish | 4.3 us / 1.8 us | M |
| Full save | 0.8 s @90k (28.5 MB JSON); open 7.05 s @22.5 MB SCL | R |
| Go GOOSE engine | 8 MB, p50 0.074 ms, p99 0.146 ms (10 IEDs, 3000/3000) | R |

## 2. Resource model (Python today vs Go target)

Python backend RAM = 50 MB + (1.06 to 2.8 KB) x points (E, from coefficients). Workers = N x (8.3 MB + 0.38 KB x pts/IED) (M-fitted).

| IEDs x pts/IED | total pts | Python backend | workers WS total | Go backend target (E) |
|---|---|---|---|---|
| 1 x 5k | 5k | 55-65 MB | 10 MB | 30 MB |
| 1 x 36k | 36k | 90-150 MB | 22 MB | 35 MB |
| 6 x 5k / 6 x 36k | 30k / 216k | 80-135 / 280-650 MB | 61 / 132 MB | 35 / 60 MB |
| 10 x 5k / 10 x 36k | 50k / 360k | 100-190 / 430-1050 MB | 102 / 220 MB | 40 / 80 MB |
| 20 x 5k / 20 x 36k | 100k / 720k | 155-330 / 810-2050 MB | 204 / 440 MB | 45 / 135 MB |

Go estimate: point struct ~48 B (ids, kind, fc, flags, int32 refs for q/t) + ref arena ~40 B + index ~40 B
+ runtime value slot 24 B = ~150 B/pt; original SCL kept on disk by hash, not in RAM.

| Load dimension | Python cost | Go engine cost (E) | Native cost (unchanged by Go) |
|---|---|---|---|
| Generated sources (0/100/1000 @10 Hz) | 1-2 % / 9-18 % / 73-82 % of a core (M) | <0.5 % / 1 % / 3-5 % | 0 / +4 % / +34 % (3 attrs x 11 us x 10k/s) (M) |
| SSE/browser clients | collapses rate 6x at 2 clients (M) | encode once per batch, O(1) per client | - |
| MMS clients per IED 0/1/5 | 1 summary event/s per IED | same | +0.3 MB each; +10 ms per extra client per request (M) |
| GOOSE pub/sub | Python sees state changes only (stNum), routes via update_batch | in-engine routing, <10 us/frame | already Go, p99 0.15 ms (R) |
| SV streams (max 16, lab only) | stats only | same | studio_live/Npcap; not measured (E: native-bound) |
| Real IEDs acquired | thread + poll per IED, <=500 watched pts | goroutine per IED | +1 studio_live 9-12 MB each (M) |
| Gateway points (<=5000) | 9 % of a core continuous (M) | change-driven, <0.5 % | studio_gateway process |

## 3. Where Python breaks first (in order of load growth)

1. **Browser fan-out x event rate**: each published event `notify_all`s every SSE thread, each rescans the
   5000-item deque (0.18 ms) and dumps JSON per client. 2-3 clients cap delivery at ~1.5k upd/s (M).
   The 5000-event buffer holds only 0.5 s at 10k/s, so reconnects resync.
2. **GIL saturation** at ~13.5k generated updates/s with no clients (M); ~75-80 us of Python per update
   (make_sample, index writes, emit, run-log JSON, equipment/energization checks). IPC encode is only 3.5 us.
3. **Point count -> RAM and save time**: ~1-2.8 KB/pt; 720k points = 0.8-2 GB and ~6 s per full save
   (E, linear from 0.8 s @90k). `MAX_XML` 20 MB per SCL file also blocks large SCDs.
4. **Lock contention**: `cycle_lock` held across probe spawn + save during operate (R 0111) stalls ticks ~1 s.
   In clean runs zero late ticks were seen until the GIL saturated.
5. Gateway diff and 10 Hz energization recompute are steady GIL taxes (9 % for 5000 points).

## 4. Are 20 IEDs / worker-per-IED / per-process RAM the real ceilings?

- **20-IED limit is a constant, not a resource limit**: `projects/service.py:51,178`, `projects/document.py:29`,
  `api/scl.py:38`, `projects/bulk_removal.py:11`, plus `Automation` pool `max_workers=20`. Measured 20 x 2.5k
  costs only ~100 MB Python + 190 MB workers. Python's real ceilings are total points and event rate.
- **Worker-per-IED RAM is moderate** (8 MB floor + 0.38 KB/pt): 100 IEDs x 5k = ~1 GB WS; acceptable on 16 GB,
  but the floor dominates for small IEDs.
- **Real native ceilings Go does not fix**:
  1. `CONFIG_MMS_SINGLE_THREADED=ON`: every extra connected client adds ~10 ms to every request (M). 5 SCADA
     clients -> 41 ms per read; operate latency targets are impossible with external clients connected.
     Needs rebuilding libiec61850 threaded (or threadless + event-driven loop). Highest-priority native change.
  2. Idle worker CPU ~1.3 % of a core each at 20 (27 % total); 100 IEDs -> ~1.3 cores idle. Needs an
     event-driven wait in the worker loop (measure access-log flusher / server loop).
  3. JSON-over-pipe ~11-13 us/attr native; fine to ~65k attr/s per IED; binary framing optional.
  4. Synthetic 34.6k-point single-LN IED: 4.5 s start and GetServerDirectory timeout (MMS PDU 65000 / huge
     LN type). Not representative (realistic 36k over 120 LNs starts in 0.42 s) but Go fixtures must use
     realistic multi-LN shapes and guard pathological LNs with a diagnostic.
  5. Loopback plan 127.50.0.2-254 caps at 253 IEDs; `CONFIG_MAXIMUM_TCP_CLIENT_CONNECTIONS=5` (build) vs 8 (runtime).
- **What Go could support (E)**: 50 IEDs x 15k (750k pts): Go ~140 MB, workers ~700 MB, Run <2 s parallel.
  100 IEDs x 5k: Go ~100 MB, workers ~1 GB (~400 MB with several IedServers per process), needs native
  fixes 1+2 to keep idle CPU < 10 % of a core.

## 5. Go budgets for the plan (acceptance thresholds)

| Metric | Python today | Go budget |
|---|---|---|
| Backend RAM 6 x 15k (90k) | 280-320 MB (R) | <= 80 MB |
| Backend RAM 20 x 36k (720k) | 0.8-2 GB (E) | <= 200 MB |
| Backend CPU idle running, no clients | 1-2 % (M), 20 % dev (R) | <= 1 % of a core |
| Total idle CPU, 20 IEDs running | ~28 % of a core (M) | <= 5 % (with native fix 2) |
| Sustained generated updates | 10k/s @80 % core; 1.5k/s with 2 browsers (M) | 50k/s @ <=1 core total Go, any 1-10 clients, 0 skipped ticks |
| UI push | per-event | 10-20 Hz batches, <= 2 ms encode per batch |
| Operate to "done" (loopback) | 1.25 s (R) | p95 <= 50 ms; <= 100 ms with 5 external MMS clients (native fix 1) |
| Symbol update after operate | 0.2 s (R) | p95 <= 20 ms |
| Run, 20 IEDs | 2.2-2.6 s @2.5k (M) | <= 1.0 s @5k, <= 2.0 s @36k; Stop <= 0.5 s |
| Project open, 22.5 MB SCL | 7.05 s (R) | <= 3 s cold, <= 0.5 s cached |
| Edit/settings save | 0.7-0.8 s (R) | <= 10 ms journal; snapshot in background |
| Gateway 5000 pts | 9 % core (M) | <= 1 % core |
| GOOSE inside, 20 publishers | p99 0.15 ms @10 (R) | p99 <= 3 ms, 0 loss |

## 6. Benchmark scenarios (one harness, black-box, same machine, same native binaries)

Harness drives the HTTP API only, samples process WS/private/CPU (GetProcessMemoryInfo/GetProcessTimes)
every second for backend + every child, records provenance (`benchmark_provenance.py`), uses a separate
loopback range and high ports, dev file watcher off. Fixtures: `load_fixture` shapes plus realistic
multi-LN 5k/15k/36k IEDs, all synthetic.

1. S1 Idle: N in {1,6,10,20,50*,100*} x 5k, Run, 60 s, no sources. (*Go only, limit raised)
2. S2 Generators: 20 IEDs, sources {100,1000,5000}, 10 Hz, SSE clients {0,1,3,10}, 60 s; 1 h soak at 1000.
   Pass: delivered/target >= 99.5 %, 0 skipped ticks, tick lateness p99 <= 20 ms, client lag <= 200 ms.
3. S3 Scale: 6 x 15k and 20 x 36k: open, Run, Stop, RAM.
4. S4 Operate: 200 SBO operates via API with {0,1,5} external MMS clients; confirm with independent `studio_probe`.
5. S5 External MMS: {1,5} clients per IED polling 500 refs/s + one report subscriber; read p99, report loss 0.
6. S6 GOOSE: 10/20 publishers inside (extend `test_goose_engine_latency`).
7. S7 Real IEDs: {1,6,20} sessions against simulated IEDs on a second range, 500 watched pts, 1 s poll.
8. S8 Gateway: 5000 IEC 104 points with an external master, 10 Hz.
9. S9 Project ops: open/save/edit/delete IED on public fixture; Python format 15 parity both directions.
10. S10 Browser fan-out: 1/3/10 tabs on SLD + Data points during S2 at 1000 sources.

## Limitations

- Python-backend memory for the full Flask app was not re-measured (only coordinator-in-process); the 2.8 KB/pt
  upper coefficient comes from the owner's running app.
- SSE clients were simulated in-process (same `after()` + `json.dumps` loop as `api/events.py`, no sockets).
- SV, gateway wire, IEC 104 and real-IED acquisition against physical devices were not run; costs are code estimates.
- All Go numbers are estimates until S1-S10 are run on the Go build.
