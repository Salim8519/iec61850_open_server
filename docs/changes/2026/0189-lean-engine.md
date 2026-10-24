# 0189: Lean engine: less memory and CPU, same answers, same network

Date: 2026-10-24. Owner request: run many IEDs with less RAM and CPU; nothing else may get worse
(user experience, IED behaviour, network usage and speed measured before and after). Decision:
[ADR-0116](../../decisions/ADR-0116-lean-engine-memory-and-cpu.md).

## Change

| File | Why |
| --- | --- |
| `internal/pyjson/object.go`, `encode.go`, `decode.go` | compact ordered object (entries slice, index only above 16 keys), exact-size decode |
| `internal/scl/builder.go`, `builder_attr.go` | 9-key point objects sized once; one shared enum object per EnumType |
| `internal/scl/controls_extended_doc.go` (new), `controls_extended_read.go` | typed controls: one SCL read per text for all its IEDs, only typed-control objects built |
| `internal/project/sources.go`, `scenario.go` | sources and scenarios look up only the points they name |
| `internal/runtime/model.go`, `equipment_control.go`, `tap_changer_route.go` | commands read the control list directly (`controlOf`) |
| `internal/memtrim/` (new); `runtime/lifecycle.go`, `stop.go`; `project/service/restore.go`, `scl.go`, `removal.go` | memory returned to Windows 2 s after open, import, IED removal, Run start and Stop |

Tests: `pyjson/object_model_test.go` (random sequences against the former map-backed object,
decoder dict rules), `scl/controls_extended_doc_test.go` (identical answers over the fixtures, and
over the local vendor files with `STUDIO_SCL_CORPUS`; one read for 8 concurrent IEDs),
`project/sources_index_test.go`, `memtrim/memtrim_test.go`.

## How it was measured

Harness (session scratchpad, not committed): the 20-IED vendor project (7 SCL files, 8 IEDs in one
file; addresses moved to 127.61.0.x), server at below-normal priority. Phases: open, API latency,
Run, 60 s of activity (35 value sources, 13 report blocks enabled on 5 independent `mmscli` MMS
clients with dchg/qchg/dupd/integrity 2 s/GI, SLD open/close every 2 s on 3 breakers, GOOSE sends
from the SLD), 30 s idle running, Stop. Private memory and CPU from Windows; reports and entries
counted by the independent clients over an exactly timed window; GOOSE frames from stNum/sqNum.
Profiles: heap, allocations and CPU from a throwaway build with `net/http/pprof`.

## Evidence

Baseline = master 22e0e95c engine; lean = this change. Cold = fresh data folder (import); warm =
reopen with the model cache.

| | Base cold | Lean cold | Base warm | Lean warm |
| --- | --- | --- | --- | --- |
| Open | 6.2 s | 4.7 s | 2.1 s | 1.2 s |
| Server memory after open | 1,028 MB | 370 MB | 997 MB | 394 MB |
| Run start | 3,211 ms | 2,270 ms | 3,214 ms | 2,222 ms |
| Server memory peak | 1,595 MB | 761 MB | 1,636 MB | 857 MB |
| After Stop | 1,595 MB | 557 MB | 1,636 MB | 588 MB |
| IED programs (20) | 507 MB | 541 MB | 493 MB | 493 MB |
| Server CPU, whole run | 43.8 s | 13.4 s | 42.0 s | 16.6 s |
| SLD command median / p95 | 56 / 77 ms | 49 / 58 ms | 58 / 82 ms | 50 / 67 ms |
| Reports per second (mmscli) | 76.0 | 76.0 | 75.9 | 76.1 |
| Report entries per second | 274 | 273 | 274 | 277 |
| GOOSE frames per second | 32.9 | 33.1 | 32.4 | 33.5 |
| API `/project`, signals, GOOSE links | 15 / 2.6 / 8.9 ms | 15 / 2.8 / 7.9 ms | 14 / 2.9 / 9.0 ms | 13 / 2.8 / 7.1 ms |
| Errors | 0 | 0 | 0 | 0 |

Earlier lean runs without the memory return (three cold, two warm) gave the same picture: peak
947-1,074 MB against 1,580-1,916 MB, server CPU 10-17 s against 37-50 s. CPU during the busy and idle
minutes and of the IED programs varies run to run in both builds (IED programs 3.7-15.4 s base,
4.4-16.2 s lean per busy minute) with no difference in the mean; idle server CPU stays at 0-0.2 s
per 30 s.

Rejected settings (same harness, base build): GOGC=50 peak 1.1-2.0 GB at slightly higher CPU;
GOGC=25 0.98 GB but about 8 % of a core while running and Run start 4.3 s; GOMEMLIMIT=512MiB
0.8 GB with the collector using about 10 cores.

- Go: `go test ./...` all packages pass (launcher with `STUDIO_PYTHON`; the licensing
  `missing_anchor_asks_one_online_check` case is the known flaky `.SERVER_TIME.tmp` case and passed on
  rerun, as the video session also found on unchanged code); `go vet ./...` clean.
- Typed controls: 185 files, 1,070 IED answers byte-identical (186 with typed controls); 58 other
  differences are IEDs the importer refuses.

## Limitations

- The native IED programs (about 25 MB each) are not changed.
- Points are still stored as objects; a typed point store remains future work (own ADR).
- The harness lives in the session scratchpad; a memory gate in `verify.py` is still future work.
