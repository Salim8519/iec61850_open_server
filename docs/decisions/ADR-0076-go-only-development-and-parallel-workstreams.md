# ADR-0076: New code only in Go, Python frozen, all phases built in parallel workstreams

Accepted: 2026-10-20. Owner: "Do all phases at once, in one go. Complete restructure to Go in
parallel with Python. Any new code should be in Go, Python is a stable version only."
Amends [ADR-0074](ADR-0074-go-engine.md) (which ran phases one after another and sent new features
to Python first). Workstreams: [workstream map](../architecture/go-engine/workstreams.md).

**Decision**

1. **Python is frozen** as the stable engine on `master`: no new features; only critical fixes the
   owner approves. It stays the default engine and the fallback exactly as ADR-0074 says.
2. **All new backend code is Go** in `studio/engine`. A feature asked for during the migration is
   built in Go only and appears when the Go engine is switched on; it never makes Go and Python
   differ on anything Python already does.
3. **Phases P0-P5 run in parallel** as six workstreams with disjoint package ownership, each in its
   own worktree and branch cut from `go-engine`, merged back into `go-engine` only with the owner's
   go-ahead. P6 (soak and switch) stays last and needs every gate.
4. **Shared contracts fixed now** so streams do not wait for each other:
   - JSON: `internal/pyjson` (ADR-0075).
   - Engineering model: immutable ordered `pyjson` trees with exactly Python's keys and order
     (the dicts `scl/importer.py` builds); hot paths build typed indexes once (point id tables).
   - Routes: `internal/api.Route` / `api.Area`; errors: `internal/apperr` (StudioError 400 or 409
     for code `conflict`, ValueError/KeyError 400, Runtime 503).
   - Native IPC: 4-byte little-endian length + JSON (≤ 16 MiB), as today.
5. **Shared files are written only by the integrating session**: `docs/history/ledger-*.jsonl`,
   `docs/changes/INDEX.md`, `docs/decisions/INDEX.md`, `docs/INDEX.md`, `studio/tools/verify.py`,
   `AGENTS.md`, `go.mod`. A workstream records its work in its own stream log under
   `docs/architecture/go-engine/streams/` and proposes ADR text there; the integrator numbers ADRs,
   writes change records and ledger lines when a stream merges. This keeps the hash-chained ledger
   linear.
6. Gates, parity and budgets are unchanged (ADR-0074, tests/06-gates.md); a stream is "done" only
   with its gate evidence.

**Compatibility impact**

None for users: the shipped engine stays Python until P6. Development of new features moves to Go,
so they reach users when the Go engine becomes the default.
