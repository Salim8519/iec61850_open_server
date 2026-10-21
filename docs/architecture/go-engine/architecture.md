# Go engine architecture

Part of the [Go engine plan](README.md); decision [ADR-0074](../../decisions/ADR-0074-go-engine.md).
Design derived from the performance trace [0111](../../changes/2026/0111-performance-trace-go-rules.md).

## 1. Processes

| Process | Today | Go engine |
|---|---|---|
| Backend | Python (Flask + Waitress), started by the launcher | `studio_server.exe` (Go, stdlib only, no cgo); Python kept as selectable fallback |
| Simulated IED | `studio_ied.exe` per IED (C++, libiec61850) | unchanged binary and IPC; later threaded MMS + event-driven idle (own ADRs) |
| Live decoder | `studio_live.exe` | unchanged |
| GOOSE engine | `studio_goose.exe` (Go) | stays a separate process (crash isolation, Npcap kept out of the server, Python fallback needs it); source moves into the shared module |
| Commands | `studio_probe.exe` per canvas operate; `studio_control` (warm) for gateway/tools | `studio_control` with a multiplexed mode: one association per endpoint, requests by id, a small pool of processes; canvas op identical to `studio_probe control` |
| XSD validation | lxml in-process | `studio_xsd.exe` (C, pinned libxml2) + result cache by SCL SHA-256 + schema + libxml2 version |
| Gateway / master | native processes | unchanged |
| Network helper | same exe with `--network-helper` | Go, same journal and rule names |

Why no cgo for MMS or XSD: a C crash would take down the whole server (today a worker crash is "one
failed IED"), cgo needs a second compiler next to MSVC-built libiec61850, and ADR-0073 set the
stdlib-only precedent. A pure-Go MMS stack would be a protocol risk without evidence.

## 2. Coexistence

A **complete Go server behind an engine switch**, not a Python/Go hybrid. A hybrid would split state
(project in Python, values in Go) and keep the biggest cost, the full save on operate, in Python.

- Switch: `--engine go|python`, `SCADABENCH_ENGINE`, or the stored Settings › Advanced choice; the
  same in `run_dev.py` and the packaged launcher. Python is the default until P6.
- One owner at a time: both engines take the `studio.lock` byte lock (msvcrt `LK_NBLCK` byte 0 ↔
  `LockFileEx` byte 0); Go starts children in a kill-on-close Job Object.
- Until P3, Go reads derived per-IED models from the derived-model cache written by Python (or by a
  one-shot `--derive-models`); in P3 the Go builder must reproduce those files exactly.
- Journal fold: Go keeps a change journal next to `autosave.scadabench` (`autosave.subsim` before ADR-0096) and folds it into the file on
  exit; the launcher folds before starting Python, so switching never loses an edit.
- Preview gaps answer 501 in dev only; a shipped build is always one engine.

## 3. Packages (`studio/engine`, dependency direction top → bottom)

```
cmd/studio_server, cmd/studio_goose
httpapi        routes (125 + static), security middleware, Flask-compatible errors, SSE
app            composition only
services       runtime, acquisition, gateway, tools, netcfg, licensing, updates, launcher
store          zip archive, journal, background snapshot, derived cache, preferences
native         framed pipe client; typed clients: ied, live, control, probe, goose, gateway, master, xsd
domain/*, scl, project   pure: signals, sources, equipment, energization, health, test mode,
                         links, flow, validation, SCL parse/build/export, document + migrations
pyjson, pycompat         Python-compatible JSON and number/text semantics
```

An architecture test in `verify.py` enforces the direction, the 300-line limit and "domain imports
nothing above it".

- `pyjson`: ordered maps; int/float/bigint kinds; Python float repr (`1.0`, `1e-05`); two encoders:
  Flask (`jsonify`: sorted keys, compact, ASCII escapes incl. U+007F, trailing newline, NaN allowed)
  and `json.dumps` (insertion order, `", "` / `": "`, used for SSE, run logs and project files).
  No HTML escaping.
- `pycompat`: `int()` parsing (`+5`, `1_000`, spaces), half-even `round`, `str.strip` set,
  `casefold`, `fromisoformat`, BOM handling, CPython `random.Random` for the noise generator.

## 4. Concurrency and data

- **Project actor**: serialises every mutation → revision check → new immutable snapshot published
  by atomic pointer → synced journal line → `project_changed` → coalesced background snapshot.
- **IED actor** (one goroutine per IED): owns the worker pipe, the command client handle and the
  IED's value array; commands are messages with a deadline. Pipe readers only decode and enqueue.
- **Cadence goroutine** at 0.1 s with 1 ms timer resolution, same arithmetic as `cadence.py`;
  pulses are timers, not a 50 Hz loop; energization recomputed only on switch position changes.
- **Event ring**: 5000 entries by sequence number, each event encoded once and shared by all SSE
  clients; one writer goroutine per client; run log writer on its own goroutine.
- **No lock is held across I/O, a process start, a pipe call or a file write.**
- **Data**: typed engineering model, interned strings, vendor/unknown fields kept as ordered raw
  values; runtime values in `[]Slot` per IED indexed by point id (assigned once at load) with a
  sequence number per slot; the original SCL text stays on disk by hash, not duplicated in RAM.

## 5. Compatibility contract (kept identical in P0-P6)

HTTP: same routes, status codes, JSON bytes and error texts (pydantic and Werkzeug included); every
error has a JSON body; a missing item never answers 404 (the UI shows "restart" for 404). SSE:
`retry: 1000`, heartbeat every 10 s, 5000-entry replay, `resync`, one `signal` event per value.
Files: format 15, CRLF text files, `+00:00` timestamps, hash-derived names (firewall rules, plan
fingerprints, source ids, node ids) computed exactly as Python does. IPC: 4-byte little-endian
length + JSON (≤ 16 MiB), int/float/bool kept distinct, events before later replies.

## 6. Top risks

| # | Risk | Mitigation |
|---|---|---|
| 1 | JSON bytes (order, separators, escapes, floats, int vs float, big ints) | `pyjson` + differential fuzzing against CPython |
| 2 | Project key order and mutation semantics | ordered tree; golden file after every scripted mutation |
| 3 | Error strings (Python, pydantic, Werkzeug) | catalogue from every `raise` + negative golden tests per route |
| 4 | Event order, ids, replay, resync | ring copied from `event_bus.py`; per-IED order compared |
| 5 | Faster replies reorder HTTP vs events | frontend end-to-end tests on Go; keep implied orderings |
| 6 | Durability across engines | synced journal, fold before switching, crash-kill tests |
| 7 | Canvas operate via `studio_control` differs from `studio_probe` | MMS PDU capture diff + mapped-control tests |
| 8 | Races without `-race` | actors without shared mutable state + stress tests |
| 9 | XML parser differences (DTD, charsets, line numbers) | corpus golden tests; explicit charset handling |
| 10 | XSD message drift | pinned libxml2 helper + invalid-file golden tests |
| 11 | Number/text semantics in cfg and validation | `pycompat` differential tests |
| 12 | Memory at many IEDs | typed points, interning, RSS budget in gates |
| 13 | Encoder becomes the hotspot | encode once, per-IED chunk cache, benchmarks in gates |
| 14 | Stale or mismatched binaries | `hello` with build id; dev launcher rebuilds newer sources (as ADR-0073) |
| 15 | Loopback/origin/session security | access-policy tests on both engines; Flask cookie accepted; constant-time compare |

## 7. Decision records to write when each phase starts

0075 Python-compatible encoding and parity definition · 0076 parity/performance gates and harness ·
0077 runtime actors, value hub, event ring · 0078 multiplexed MMS command client · 0079 journal and
background snapshot · 0080 derived-model cache · 0081 `studio_xsd` and libxml2 · 0082 single Go
module and architecture checks · 0083 Go launcher and dual-engine packaging · 0084 licensing and
updates in Go · 0085 network helper in Go · 0086 default engine switch · native: threaded MMS server,
event-driven worker loop.
