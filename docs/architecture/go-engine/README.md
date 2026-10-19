# Go engine plan (approved 2026-10-19)

Decision: [ADR-0074](../../decisions/ADR-0074-go-engine.md). Evidence: [0110](../../changes/2026/0110-backend-lag-health-polling.md),
[0111](../../changes/2026/0111-performance-trace-go-rules.md), [0112](../../changes/2026/0112-go-engine-plan.md).

The owner approved rewriting the whole Python backend in Go with three goals: **fast, light, able
to carry many more devices**, and one absolute rule: **the user must not notice any change** except
speed. Same screens, texts, defaults, API, events, files, limits and protocol behaviour.

## Documents

| Part | Content |
|---|---|
| [Architecture](architecture.md) | Process model, coexistence, packages, concurrency, data model, risks |
| [Parity inventory](inventory/00-index.md) | 127 routes, 61 event kinds, ~30 disk artefacts, 7 processes, ~150 limits/defaults |
| [Tests and gates](tests/06-gates.md) | 550 Python, 178 frontend, 21 Go tests classified; harness, golden replay, SCL corpus, gates |
| [Workstreams](workstreams.md) | Six parallel streams, ownership, shared contracts, integration order (ADR-0076) |
| [Capacity](capacity/capacity-audit.md) | RAM/CPU per component and per point, where Python breaks, Go budgets, benchmark S1-S10 |

## Rules that hold for every phase

1. Python stays the default engine until the final gate; Go is selectable (Settings › Advanced
   and `--engine`) in dev **and** the packaged app; after the switch Python stays as a fallback.
2. Only one engine runs at a time (shared `studio.lock` byte lock); both read and write project
   format 15; a Go journal is folded into the `.subsim` before Python starts.
3. A shipped build is always all-Python or all-Go. Routes not yet ported answer 501 in the Go
   preview, which is dev-only until its phase passes.
4. Nothing is copied blindly: every Python cost found in 0110/0111 is fixed by design (the ten Go
   rules in 0111 §3) while the visible behaviour stays byte-identical.
5. Every phase ends with its gate. Go must be **equal or faster on every timing** and **within its
   RAM/CPU budget**; a regression blocks the phase.
6. New backend code is written only in Go; Python is frozen as the stable engine (critical fixes
   only), and phases P0-P5 run in parallel workstreams ([ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)).
7. AGENTS.md applies to Go: files under 300 lines, one responsibility each, domain code pure, ADRs
   for contracts and dependencies, change record and ledger line for every step, no EXE unless asked.

## Phases (biggest speed gain first)

| Phase | Scope | Gate (summary; details in tests/06-gates.md) |
|---|---|---|
| P0 Foundation | One Go module `studio/engine` (GOOSE engine moves in), Python-compatible JSON (`pyjson`) and number/text rules (`pycompat`), HTTP skeleton with the same security order (403 → 401 → 402 → startup → route), session cookie, SSE ring, engine switch and lock, parity harness (`STUDIO_ENGINE=python|python-http|go`), benchmark harness S1-S10, derived-model cache bridge | `pyjson` differential fuzz vs CPython 10⁶ cases, 0 diff; `python-http` = `python` on all tests; security/error bodies identical |
| P1 Runtime | Project read + runtime mutations, Run/Stop/single IED (parallel start), per-IED actors, cadence, pulses (timers), scenarios, automation, operate through a persistent multiplexed MMS client, native feedback, energization (on switch change only), equipment states, run log on its own goroutine, journal save | Scenario and protocol suites identical; operate symbol ≤ 20 ms, done p95 ≤ 50 ms; edit save ≤ 10 ms; 50k updates/s at ≤ 1 core |
| P2 Live reads | Health, GOOSE/SV monitors and links, MMS access, test mode, live values, signal search index, real-IED acquisition (persistent sessions, discovery cache) | API parity; real-IED evidence with an independent client; health ≤ 5 ms |
| P3 SCL + projects | One-pass XML parse, model builder, type resolver, validation, export, import, migrations 1-15, removal, SLD import, signal flow; XSD through a pinned native libxml2 helper + cache | Byte-identical cfg/JSON/diagnostics/export over the full corpus; `.subsim` both directions; open ≤ 3 s cold, ≤ 0.5 s cached |
| P4 Tools | Gateway IEC 104 / converter / master, explorer, find/scan, test panel, activity, CSV | Wire tests + API parity; gateway 5000 points ≤ 1 % CPU |
| P5 Platform | Network helper (IP Helper via syscall, journal, firewall), licensing, updates, Go launcher, packaging with both engines | Full suite on both packaged engines; journal and license compatibility |
| P6 Switch | Soak 24 h (20 IEDs; 50/100 IEDs for Go), owner hands-on check, default flips to Go | All gates green, identical UI walk-through at 1366×768, owner sign-off |

After P6 (visible only as speed): batched value frames and a frontend value store outside React
state (one render per frame, not per event), push instead of the ~24 polled paths, targeted
patches instead of `/project` refetches. These change the wire contract, so they get their own ADR
and run with both engines' frontends kept in step.

## Native work (benefits both engines; own ADRs and protocol evidence)

| Item | Why | Evidence |
|---|---|---|
| MMS server built threaded (or event loop) | Single-threaded server adds ~10 ms per request per extra client (41.5 ms/ref with 5 clients) | capacity §3 |
| Event-driven worker idle loop | 20 idle workers use 27 % of a core | capacity §3 |
| `studio_control` multiplexed canvas op | Replaces one `studio_probe` process per command; must match its orIdent, CmdTerm wait, timeouts and texts (PDU capture diff) | 0111 §1 |
| `studio_xsd` helper (pinned libxml2) | Identical XSD messages without cgo | architecture §1 |
| Several IED servers per process (later) | 8 MB floor per worker at 100 IEDs | capacity §3 |

## Budgets (Python today → Go target)

| Metric | Python (measured) | Go budget |
|---|---|---|
| CB operate to "done" | 1.25 s | p95 ≤ 50 ms (≤ 100 ms with 5 MMS clients) |
| Symbol after operate | 0.2 s | ≤ 20 ms |
| Diagram edit / setting save | 0.7-0.8 s | ≤ 10 ms |
| Open project (22.5 MB SCL) | 7.05 s | ≤ 3 s cold, ≤ 0.5 s cached |
| Backend RAM, 6 IEDs / 90k points | 280-320 MB | ≤ 80 MB |
| Backend RAM, 20 IEDs × 36k points | 0.8-2 GB (est.) | ≤ 200 MB |
| Backend CPU idle while running | ~1-2 % (+ dev watcher) | ≤ 1 % |
| Generated updates | saturates at ~13.5k/s (1.4k/s with 3 tabs) | 50k/s at ≤ 1 core, 1-10 tabs |
| Run 20 IEDs | 2.2-2.6 s (serial) | ≤ 1 s (5k points), ≤ 2 s (36k) |
| Delete IED | 3-5 s | ≤ 0.1 s |

The 20-IED limit is a constant, not a resource limit. It stays 20 until P6 so nothing changes for
the user; raising it later (50/100) is a separate owner decision after the soak and native items.

## Owner decisions taken with the plan (changeable by a new ADR)

- No cgo in production; no non-stdlib Go modules (as ADR-0073); race checking by actor design and
  stress tests. A cgo build for `-race` tests only needs a separate yes.
- New native dependency: libxml2 pinned to the version lxml 6.0.2 bundles, in `studio_xsd`.
- Journal written and synced on every change; full snapshot in the background, flushed on exit.
- Identical means: same bytes after normalising ids/timestamps/ports, same order per IED; Go may
  be faster, and events of different IEDs may interleave differently.
- Engine choice is a user setting (Settings › Advanced), never switched silently.

## Not decided yet

- Raising the IED limit above 20 (after P6).
- Binary protection of the Go server (e.g. `garble`) — build tool decision, not needed for parity.
- Playwright for the pixel UI gate (not installed in the studio venv today).
