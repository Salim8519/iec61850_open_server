# ADR-0074: Go engine replaces the Python backend, identical behaviour, Python as fallback

Accepted: 2026-10-19 (owner approved the plan in full). Plan: [Go engine plan](../architecture/go-engine/README.md).
Change: [0112](../changes/2026/0112-go-engine-plan.md). Builds on [ADR-0073](ADR-0073-goose-engine-go.md).

**Problem**

The trace in [0111](../changes/2026/0111-performance-trace-go-rules.md) measured the Python backend on the owner's
project (6 IEDs, 90,222 points): a breaker operation 1.25 s (0.8 s of it a full 28.5 MB project
save), project open 7.05 s, every edit 0.7-0.8 s, backend RAM 280-320 MB, the GIL saturating at
~13.5k value updates/s (1.4k/s with three browser tabs), and one lock held across I/O. The owner
wants the app fast, light and able to carry many more devices, with **no change the user can notice**
other than speed.

**Decision**

1. The whole backend is rewritten in Go as `studio_server.exe` (stdlib only, no cgo), in one module
   `studio/engine` that also holds the GOOSE engine. Native workers, `studio_live`, gateway and
   master processes stay; commands use a persistent multiplexed `studio_control`; XSD validation uses
   a pinned native libxml2 helper. Details: [architecture](../architecture/go-engine/architecture.md).
2. **Identical behaviour is the contract**: same routes, status codes, JSON bytes (Flask and
   `json.dumps` formats), error texts, events, files (format 15), hash-derived names, limits and
   defaults, as listed in the [parity inventory](../architecture/go-engine/inventory/00-index.md). Bytes are
   compared after normalising ids, timestamps and ports; order is compared per IED. Go may be
   faster; nothing else may differ.
3. **Python stays the default** until every gate passes and the owner signs off; both engines are
   selectable in dev and the packaged app (Settings › Advanced, `--engine`), only one runs at a
   time (shared `studio.lock`), both read and write format 15, and a Go journal is folded before
   Python starts. After the switch Python stays as a fallback and is never chosen silently.
4. **Order by speed gain**: P0 foundation and harness, P1 runtime (commands, actors, value hub,
   journal save), P2 live reads, P3 SCL and projects, P4 tools, P5 platform (network helper,
   licensing, updates, launcher, packaging), P6 soak and switch.
5. **Nothing copied blindly**: the ten rules of 0111 §3 apply (immutable model, save the change not
   the project, no lock across I/O, persistent MMS client, push-only value hub, non-blocking readers,
   index once, parse once and cache, parallel Run/Stop, dev tooling never counted as engine load).
6. **Gates**: per phase, from [tests and gates](../architecture/go-engine/tests/06-gates.md); Go must be equal or
   faster on every timing and within the [capacity budgets](../architecture/go-engine/capacity/capacity-audit.md)
   (e.g. operate p95 ≤ 50 ms, edit save ≤ 10 ms, open ≤ 3 s cold, RAM ≤ 80 MB for 90k points).
7. Native improvements that help both engines (threaded MMS server, event-driven worker idle loop,
   several IEDs per worker process) get their own ADRs and protocol evidence.
8. The 20-IED limit stays until after the switch; raising it is a separate decision.

**Compatibility impact**

None visible while Python is the default. The Go preview is dev-only until its phase passes; a
shipped build is always one engine. Project files remain format 15 in both directions. Existing
firewall rules, network journals, licenses and recovery files keep working because their names and
formats are reproduced exactly. The wire contract to the frontend (SSE, polling) is unchanged until
after the switch; batching and push are a later ADR.

**Dependencies**

Go 1.24+ standard library only (as ADR-0073). New native dependency: libxml2 pinned to the version
bundled by lxml 6.0.2, in `studio_xsd.exe`. No cgo in production; a test-only cgo `-race` build would
need a separate owner decision.

**Alternatives rejected**

- Runtime-only Go sidecar under Python: splits state between processes and leaves the full save on
  every operate in Python.
- Pure-Go MMS client/server: protocol risk with no evidence; libiec61850 stays.
- cgo bindings to libiec61850/libxml2: one C crash would stop the whole server; second toolchain.
- Fixing Python only: the GIL ceiling (~13.5k updates/s) and per-object RAM (~2.8 KB per point in
  the running app) remain.
