# 0112: Final Go engine plan, parity inventory, test gates and capacity budgets

Date: 2026-10-19. Planning and evidence only; no product code changed. Owner request: consider every
part of the app (commands, IED counts, devices, GOOSE, MMS, everything), calculate RAM and CPU, do a
final check with several agents, and write the final plan; plan approved in full; the Go app must be
identical for the user. Decision: [ADR-0074](../../decisions/ADR-0074-go-engine.md). Plan:
[Go engine plan](../../architecture/go-engine/README.md).

## Work done

Four read-only agents in parallel, plus measurements of the running dev app:

- Parity inventory (17 documents): 127 routes (125 `/api/v1` in 24 blueprints + `/` and `/assets`),
  ~121 frontend calls, 24 polled paths (300 ms to 30 min), 61 event kinds, ~30 disk artefacts,
  7 native/Go executables plus the network helper, ~150 limits/defaults. Routes missing from
  `docs/reference/API.md` were found (`/sv`, test mode, health, signal flow, substation sources,
  diagram preview).
- Tests and gates (7 documents): 550 Python tests in 144 files, 178 frontend, 21 Go; classified
  black-box HTTP / protocol / internal / frontend / packaging; harness with
  `STUDIO_ENGINE=python|python-http|go` needing ~40 lines in `conftest.py`; golden replay on the
  public demo only (recorder refuses non-public SCL); SCL corpus with ~120 synthetic edge cases;
  gates G0-G5 and a pixel UI walk-through at 1366×768.
- Capacity (isolated processes on 127.61/127.62.x and high ports, copies of native binaries,
  synthetic data; the owner's running session untouched).
- Architecture and risks (process model, coexistence, packages, concurrency, 15 risks, ADR list).

## Measurements (M = measured, E = estimate)

| Item | Result |
|---|---|
| Running dev app, 6 IEDs / 90k points | Python backend 280-320 MB; `studio_ied` ~16 MB each (95 MB); `studio_goose` 8 MB; `studio_live` 6 MB (M) |
| Python RAM per point | signal dict 536 B alone, 682-729 B with strings; engine without web ~1.06 KB/point; full app ~2.8 KB/point (M) |
| Python CPU per generated update | 75-80 µs; 100 sources 9-18 %, 1000 sources 73-82 % of a core (M) |
| Python ceiling | GIL saturates at ~13.5k updates/s; with 2-3 SSE clients only 1.4-1.6k/s delivered (M) |
| Native worker | 8.3 MB + 0.38 KB/point working set; start 0.11-0.12 s each; Run 20 IEDs 2.2-2.6 s serial (M) |
| MMS server | built single-threaded: 0.15 / 10.4 / 20.9 / 41.5 ms per ref with 1 / 2 / 3 / 5 clients (M) |
| Idle workers | 20 idle workers 27 % of a core (M) |
| Gateway 5000 points | 10 Hz diff 8.9 ms per tick, ~9 % of a core (M) |
| 20-IED limit | a constant in five places, not a resource limit (code) |

## Outcome

ADR-0074 accepted with phases P0-P6, budgets, native items and owner decisions; architecture and
inventories stored under `docs/architecture/go-engine/`.

## Limitations

- All Go figures are budgets/estimates until benchmarks S1-S10 run on the Go build.
- SV, IEC 104 wire traffic and physical real IEDs were estimated from code, not run.
- SSE clients in the capacity test were simulated in-process, not real sockets.
- The full Flask app's RAM per point comes from the owner's running app, not a controlled run.
- Effort estimate from the architecture review: about 20-22 weeks of phased work.
