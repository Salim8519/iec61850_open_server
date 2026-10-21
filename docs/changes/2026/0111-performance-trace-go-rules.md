# 0111: Full performance trace of the Python engine, and the rules the Go engine must follow

Date: 2026-10-19. Evidence-only diagnosis; no product code changed. Owner request: before the
backend is rewritten in Go, trace where Python is slow so the Go engine is designed from evidence,
"not copied blindly", including network and communication paths. Follows [0110](0110-backend-lag-health-polling.md).

Method: dev app running the owner's project (6 simulated IEDs, 90,222 points, 22.5 MB SCL,
This PC only); `py-spy` sampling of the live backend (installed in a scratch folder, not the
project); cProfile and timing scripts on an offline copy of the project; browser resource timing
and an extra event-stream listener; three read-only code audits (backend hot paths,
communication paths, SCL open/save). Customer SCL content was never printed or committed.

## 1. What the user feels, measured

| Action | Today | Where the time goes |
|---|---|---|
| CB Close/Open (SLD) | 1.25 s to "done", symbol 0.2 s | ~0.8 s full project save on every operate (`node_data` → `save` → `encode`); ~0.1-0.2 s new `studio_probe.exe` per command; ~0.06-0.08 s real MMS SBO + operate + CmdTerm |
| Open project | 7.05 s (8.7 s under profiler) | XSD 2.3 s, pure-Python XML parse 1.5 s (+ second lxml parse), model build 1.3 s, final save 0.73 s |
| Any diagram edit, gateway tick box, source change | ~0.7-0.8 s each | same full save; scenarios save twice (1.4 s) |
| Delete an IED | ~3-5 s | three sorted JSON dumps, deep copy of the whole project incl. points, backup + save |
| Signal flow tab, SLD import dialog | 1.1-1.5 s per request | SCL re-parsed every time; SLD import parses while holding the project lock |
| Idle while running (dev) | ~20 % of one core | 97 % of samples are the werkzeug dev file watcher (dev only) |
| Before 0110 | operate 25 s, `/health` 25 s | health deep-copied 90k points per poll (fixed in 0110) |

Direct comparison: `studio_probe` SBO operate from a script 120-150 ms (process start ~100 ms
included); the same command through the backend 1.25 s.

## 2. Root causes (ranked)

1. **Whole-project save on hot paths.** `projects/service.py` `save()` runs on operate, diagram
   PUT (every drag, 400 ms debounce, and before every Run), sources, scenarios (twice), test
   panel, gateway and master settings, GOOSE reach, IED settings, test mode. Each one dumps
   28.5 MB JSON, re-hashes every SCL (SHA-256) and deflates it (0.69 s) under `projects.lock`;
   most then `changed()`, so every tab refetches `/project`.
2. **One global lock around I/O.** `coordinator.cycle_lock` is held across the probe spawn and the
   save during operate, across every IED's pipe round trip in the 10 Hz tick, by pulses 50 times a
   second even when idle, and by native control feedback. One click stalls the run loop ~1 s.
3. **Live values written into the engineering model.** The coordinator indexes the project's own
   point dicts, so save, snapshot and health need copies and exclusions, incremental save is
   impossible and threads share mutable dicts.
4. **One process per command.** Canvas operate spawns `studio_probe.exe` with a new association
   each time. The gateway path already keeps a warm client, but one global lock serialises all IEDs
   and confirmation polls every 100 ms.
5. **One event per value, serialised many times.** Each changed point is one SSE event; the bus
   scans the whole 5000-item deque per wake per client, dumps JSON per client and again for the
   run log (file I/O inside the tick). The browser copies the whole value map and re-renders once
   per event. Every connect replays the buffer (5,014 events measured).
6. **Pull instead of push.** About 15 views poll (300 ms - 3 s); `/project` (309 KB, deep copy) is
   refetched after every action and on `runtime`, `live_status`, `resync`, `project_changed`.
   Live traffic itself is small (13 events/s, 4 KB/s while running).
7. **Blocking reader threads.** Worker pipe readers run consumers inline (emit, log I/O, locks,
   cross-IED `update_batch` IPC from GOOSE routing), so a slow Python stalls native MMS/GOOSE
   threads through the pipe.
8. **Rebuild instead of index.** Point lookups scan lists (validation, test panel, `signal()`
   when stopped), Data points search rebuilds a 15k index per keystroke, energization recomputes
   topology on any mapped write (also analog sources at 10 Hz), the gateway diffs every point at
   10 Hz.
9. **Re-parse instead of cache.** SCL parsed twice per open (defusedxml pure Python + lxml),
   models rebuilt and re-validated on every open, re-parsed again by signal flow / SLD import.
10. **Serial Run/Stop.** Workers start one after another; one blocking round trip per start value;
    Stop closes workers one by one under `cycle_lock`.

Already good: worker framing (length + JSON) is sound; `studio_live` emits GOOSE only on stNum or
problem change; the Go GOOSE engine forwards with p99 0.15 ms; adapters come from the IP Helper API
(PowerShell only as a fallback and for explicit actions).

## 3. Rules the Go engine must follow

1. **Engineering model is immutable** after import, shared by pointer, with a revision number.
   Runtime values live in a separate table indexed by point id (arrays, not maps of dicts).
2. **Save the change, not the project.** Original SCL stored once by content hash and never
   re-serialised; compressed SCL segments cached; full snapshots written in the background,
   coalesced, flushed on exit. Format 15 stays readable by Python (both directions tested).
3. **No lock across I/O, process start, pipe call or file write.** One actor goroutine per IED
   owns its worker pipe and MMS client; commands are messages with a deadline.
4. **Persistent MMS client per endpoint**, shared by canvas, gateway and tools. No process per
   action. Target: symbol 5-20 ms, "done" 20-50 ms (loopback).
5. **Value hub, push only.** Latest value per point, emit on change only, coalesce to 10-20 Hz
   batches, encode once and share bytes across clients, ring buffer by sequence, snapshot on
   subscribe then deltas, per-topic subscriptions. REST only for one-shot reads.
6. **Readers never block.** Decode and enqueue only; bounded channels with an explicit
   coalesce policy for status events; logging on its own goroutine.
7. **Index once.** (IED, ref) → point id at import; search index per IED; energization
   recomputed only when a switch position changes, incrementally.
8. **Parse once, cache derived data** by SCL hash + builder version + schema/libxml2 version
   (outside the project, treated as untrusted, rebuilt on miss). Fidelity rule unchanged: the
   original SCL text is the only authority.
9. **Parallel Run/Stop**, start values in the worker config, batched reads.
10. **Same behaviour in dev and the packaged app.** The dev file watcher is dev tooling only and
    must never be mistaken for engine load in measurements.

## 4. Expected gains (estimates, to be proven by the parity and timing gates)

| Path | Python now | Go target |
|---|---|---|
| CB operate to "done" | 1.25 s | 0.02-0.05 s |
| Diagram edit / settings save | 0.7-0.8 s | < 0.01 s (journal), background snapshot |
| Open project | 7.05 s | ~2.5 s without cache (XSD via libxml2 does not get faster), ~0.2 s with derived cache |
| Live value to screen | per-event renders | one batch per frame, 10-20 Hz |
| Delete IED | 3-5 s | < 0.1 s |

## 5. Parity traps found (Go must match Python exactly)

BOM removed by `utf-8-sig` before hashing; Python `str(float)` writes `100.0` (Go `100`); FLOAT
defaults are int `0`; `round()` is half-to-even; `int()` accepts `+5`, `1_000` and spaces;
`str.strip` treats `\x1c-\x1f` as whitespace; `json.dumps(ensure_ascii=True)` in cfg lines; Go JSON
must not escape HTML; key order preserved; ISO timestamps via `fromisoformat`. Golden outputs:
per-IED cfg text, signals/controls/datasets/reports/GOOSE/SV/validation JSON, diagnostics and error
strings, exported SCL, source ids, signal flow and GOOSE sources, `.subsim` both directions.

## Limitations

- Expected Go gains are estimates; only the Python measurements above are evidence.
- Event-storm costs at thousands of generated sources are code-reading estimates, not measured.
- Browser long-task timing during the 5,014-event replay could not be captured (pane in the
  background); page load measured 0.55 s.
- Real-IED acquisition, gateway and IEC 104 paths were audited from code only, not run.
- `py-spy` lives in the session scratch folder, not in the project or its requirements.
