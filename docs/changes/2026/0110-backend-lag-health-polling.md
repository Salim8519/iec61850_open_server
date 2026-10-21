# 0110: Backend lag fixed: health no longer copies the model, polls are shared

Date: 2026-10-19. Owner report: "the app feels heavy, click open and close and watch how slow it
is" (dev app, SLD, QA1 Operate). Owner choice: fix the lag in Python now, then plan the Go engine.

## Findings (dev app, 6 simulated IEDs, 90,222 data points, This PC only)

- QA1 *Close*: `POST /equipment/<node>/operate` answered after **25.3 s**. The browser main thread
  was blocked 0 ms; the page was not the problem.
- At the same time `/health` took 23-26 s, `/goose/links` 4-10 s and the tiny `/activity` up to 3 s.
  The Python backend used 98 % of one core continuously (528 CPU seconds); the C++ workers 2-8 s.
- Profile of the saved project (offline copy): `station_health` deep-copied every IED with all
  90,222 points on every call, **1,142 ms** (5.2 s under the profiler, 2.5 million `deepcopy`
  calls), while holding the project lock. The health rules themselves took 0 ms and the mode badges
  132 ms (regex over every reference). Health only reads the reference and FC of Mod, Beh and Sim
  points (848 of 90,222).
- `/goose/links`: the link map itself takes 1 ms; `stored_sources` SHA-256 hashed the full SCL text
  of every IED on every call (84 ms, under the project lock).
- Frontend: `/health` was polled every 3 s separately by the mode badges, Signal flow and Health
  check, `/goose/links` every 5 s by the Devices page, GOOSE partners and the SLD layers. None
  skipped a tick while the previous answer was pending, so a slow backend received a pile-up.

## Change

- `acquisition/health_snapshot.py`: `health_view` copies only what health reads; the point list
  is replaced by the IED's Mod/Beh/Sim points (`domain/test_mode.mode_signals`), kept per IED
  and reused while the IED keeps the same point-list object (references and FCs never change in
  place; a new model brings a new list). Removed IEDs leave the cache.
- `projects/scl_sources.source_id`: cached for the (at most 20) SCL texts of a project.
- Frontend `api/sharedPoll.ts` + `api/polls.ts`: one request at a time per path, shared by every
  view, at the fastest interval asked for; a view that opens later gets the last answer at once;
  a request asked for while one is in flight runs once after it. Used by `useModes`,
  `HealthView`, `SignalFlowView`, `useGooseLinks` and `useSignalLayers`. Display only; no API,
  contract or project-data change, so no ADR.

## Evidence

- Identical output on the owner's project: `mode_points`, `mode_items`, `mode_summary`,
  `build_health` (simulated and real roles) and `mode_badges` compared old (full deep copy) against
  new for all 6 IEDs: equal. New cost: `health_view` 25 ms first call, 7 ms after; mode badges
  1.3 ms; `stored_sources` 84 → 5 ms.
- Live dev app after the change (Run, SLD, QA1, three operations): diagram symbol changes after
  215-272 ms, *Open/Close done* after 1.25-1.28 s, `operate` 1.16-1.19 s (was 25.3 s).
  `/health` 22-39 ms, `/goose/links` 14 ms, `/activity` 7-15 ms. Backend CPU ~20 % of one core
  (was 98 %) with two browser tabs open.
- Tests: `tests/unit/test_health_mode_points.py` (equivalence and cache follow a new model),
  `tests/frontend/sharedPoll.test.ts` (sharing, no overlap, late view, errors, timer); 460 Python
  unit + integration pass, 178 frontend pass, `tsc --noEmit` clean.

## Limitations

- `operate` still takes about 1.2 s: each command starts `studio_probe.exe`, associates, operates
  and waits for command termination (ADR-0069). Unchanged here; a persistent client is a candidate
  for the Go engine plan.
- About 20 % CPU remains with the app running and two tabs polling; not profiled further. Other
  pollers (gateway, explorer, SV traffic) were not moved to the shared poller.
- The dev reload restarted the owner's running session once; Run was pressed again for the retest.
- Packaged EXE not built (not requested).
