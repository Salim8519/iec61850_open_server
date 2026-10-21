# 0114: Python frozen, new code only in Go, six parallel Go workstreams

Date: 2026-10-20. Owner: "Do all phases at once, in one go. Complete restructure to Go in parallel
with Python. Any new code should be in Go, Python is a stable version only."
Decision: [ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md).
Map: [workstreams](../../architecture/go-engine/workstreams.md).

## Change

- Branch `go-engine` created from `master`; commit `90123dd` holds 0110-0113. `master` unchanged.
- `AGENTS.md`: all new backend code in Go under `studio/engine`; Python backend frozen (critical
  fixes only); Python stays default until the Go gates pass.
- Shared contracts in `studio/engine`: `internal/api` (`Route`, `Area`) and `internal/apperr`
  (StudioError with default code `validation`, 409 for `conflict`; ValueError; KeyError with
  Python's repr text; Runtime 503), matching `app_factory.py` error handlers.
- Workstream map: A platform-core (P0 server, security, SSE, switch), B scl-project (P3),
  C runtime (P1), D live-acquisition (P2), E tools (P4), F harness-platform (P0 harness, P5
  network/licensing/updates). Disjoint ownership; shared files written only by the integrator;
  stream logs under `docs/architecture/go-engine/streams/`.
- The engineering model contract is the ordered `pyjson` tree with Python's exact keys (observed
  from the public template IED: 18 IED keys; signal, control, dataset, report key orders).

## Evidence

- `go vet ./...` clean; `python studio/tools/verify.py`: see the ledger line for this change.

## Limitations

- The streams start from this commit; their work lands on their own branches and is merged into
  `go-engine` only with the owner's go-ahead.
- Calendar effort is unchanged by parallelism for the longest stream (C runtime, B SCL); total
  elapsed time shortens, gates are unchanged.
