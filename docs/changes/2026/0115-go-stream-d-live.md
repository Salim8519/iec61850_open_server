# 0115: Go stream D merged: live values, GOOSE/SV supervision, health, test mode, signal search

Date: 2026-10-20. Workstream D of [ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)
(phase P2). Branch `go-d-live` (8 commits, 2ba5a9e..49de5ac) fast-forwarded into `master`. Stream log:
[d-live](../../architecture/go-engine/streams/d-live.md). Python stays the engine; this Go code is not
wired into a running server yet.

## Change

- `studio/engine/internal/acquisition`: domain ports `goosedom`, `svdom`, `livedom`, `modes`,
  `health`, `signalindex`, helpers `pyv`; services for real-IED sessions (studio_live), live values,
  GOOSE engine/monitor/inputs/share, SV monitor, MMS clients, health snapshot (cached per-model
  views, no project copies), test modes; integration ports in `ports.go`.
- `internal/api/liveapi`: 17 routes (`/live*`, `/goose*`, `/mms/access`, `/sv`, `/health*`,
  `/ieds/{id}/signals`, `PUT /ieds/{key}/test-mode`).

## Evidence (re-checked by the integrator)

- Stream: goldens from the Python functions on public SCDs and synthetic inputs, 0 differences:
  goosedom+svdom 1,078, livedom 2,088, modes+health 423, signal search 546 recorded Flask answers.
- Route replay: real Flask app on the model substation, stopped and running (fake native workers,
  22-request conversation replayed): 48 HTTP answers identical in status, content type,
  disposition and body bytes, and identical events per step.
- Real IED (opt-in test): stand-in `studio_ied` on 127.64.0.31:20131 through the real
  `studio_live.exe`, independent client: 135 points match, report block reserved/enabled, written
  value read back, health ok, block released on Stop; start-to-live 123 ms Go vs 138-340 ms Python.
- Median per request, Go vs Python: `/health` 214 µs vs 1,905 µs; `/goose/links` 42 vs 977;
  `/sv` 32 vs 521; `/live` 118 vs 436; signal search over 90k points 4.3 ms vs 79.6 ms.
- Integrator: only owned paths changed (135 files); `go vet` clean; `go test -p 2 ./...` all
  packages pass (GOMAXPROCS=4); `verify.py` on `master` after merge: see ledger line.

## Limitations

- Not wired: needs stream C (native client, value store, coordinator calls), stream B (immutable
  project, declarations cache), stream A (route mount, emitter, shared error writer), stream F
  (adapter list) through the interfaces in `acquisition/ports.go`.
- Discovery cache across reconnects not built (Python rediscovers each time; identity rule open).
- Deviations listed in the stream log: malformed internal data reads as None instead of raising
  (not reachable from users); watched-point order deterministic instead of Python set order;
  reconnecting a connected IED stops the replaced session; a lost connection ends the poll wait at once.
- Real-device evidence used a stand-in worker, not a physical relay.
