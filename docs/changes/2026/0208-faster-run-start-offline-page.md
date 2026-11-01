# 0208: Faster Run start and project opening, and an offline page policy

Owner: "31 still takes time to start, why?", "go bro do all phases, fix and test", "make sure the
app ui is completely offline". Decision: [ADR-0133](../../decisions/ADR-0133-faster-run-start-offline-page.md).

## Change

| Area | Files |
| --- | --- |
| Value-cache index (native) | `studio/native/libfix/value_cache_index.{c,h}` (new), `cmake/value_cache_index.cmake` (new: hash-pinned generated copy of `mms_value_cache.c`), `CMakeLists.txt` |
| Overlapped start | `engine/internal/runtime/supervisor.go` (lab addresses with the GOOSE bus, `prepareNetwork`, `writeStartValues` in batches), `start_overlap.go` (new), `lifecycle.go` (listeners start with the IEDs, `startValues`) |
| Spare GOOSE engine | `engine/internal/acquisition/goose_spare.go` (new), `goose_engine.go` (`greet`, `takeSpare`), `app/wire_live.go` (prepare after Stop and 3 s after launch), `app/shutdown.go` (`Discard`) |
| Memory trim | `engine/internal/memtrim/memtrim.go` (`Hold`), `runtime/lifecycle.go`, `runtime/stop.go` |
| Opening | `engine/internal/project/service/cached_reads.go` (new: cached models read together), `restore.go`, `models.go`, `project/validation.go` (`ValidateDiagram` uses the index) |
| Idle start data | `engine/internal/runtime/start_warm.go` (new), `app/wire_project.go`, `app/compose.go`, `app/startup.go` |
| Offline page | `engine/internal/httpapi/offline_policy.go` (new), `static.go`, `frontend/vite.config.ts`, `tools/check_offline_ui.py` (new), `tools/verify.py` |
| Tests | `runtime/start_overlap_test.go`, `runtime/fake_native_test.go` (the fake's simulated operator commands keep their old moments), `acquisition/goose_spare_test.go`, `memtrim/memtrim_test.go`, `httpapi/offline_policy_test.go`, `runtime/start_warm_test.go`, `tests/unit/test_check_offline_ui.py` |

## Evidence (owner's laptop, Ryzen 9 5900HS, 16 threads; other sessions' builds and tests often kept the CPU at 30-100 %)

- Where the time went before: a timing build on a copy of the owner's 7-IED lab project
  (IEDs of 1.4-1.8 MB models, 293-474 controls) measured 10-12 s per Run; inside each worker
  `installDefaultValuesInCache` took 0.43-1.26 s of `IedServer_createWithConfig`.
- Worker alone, the owner's real start requests replayed: biggest IED 1.5-1.8 s -> 0.30-0.37 s;
  all seven at once 2.0 s -> 0.74-0.81 s (two rounds each, same PC, alternating old and new).
- Repeated Start/Stop on an open project, old (HEAD 449191b5 engine, installed 0.1.31 workers)
  against new, alternated on the same project copy: Start median 7.39 s -> 3.79 s, Stop 0.47 s ->
  0.32 s (6 rounds). A second run while Windows DNS Client was using 5-11 processors: Start 13.77 s
  -> 9.75 s, Stop 0.66 s -> 0.59 s.
- Whole app, a fresh process each round (open, page files, ten page-data requests at once,
  Start one second later, Stop), 5 alternating rounds: open 0.58 s -> 0.44 s, page 0.85 s ->
  0.71 s, page data 7.94 s -> 3.40 s, Start right after opening 10.75 s -> 5.73 s.
- First Start after opening, timing build: the start request of each IED (typed controls and
  writable deadbands from its SCL text) took 1.3-7 s while seven IED processes loaded; a text
  reads in 0.11-0.14 s alone (7 texts of 3.5-4.4 MB: 1.5 s one after another). With the idle
  read the first Start builds its requests at once, like the second.
- Wire: `native_wire_compare.py` (installed 0.1.31 workers against the new build, studio_probe
  as the client, recording proxy): 5 MMS connections byte-identical after masking times (9570,
  198, 746, 198, 758 bytes), 4 reports each, 14 GOOSE frames identical in order.
- Real models: the owner's 7 start requests served by the installed 0.1.31 `studio_ied` and the
  new one side by side, read with the installed `studio_probe`: browse output identical
  (3.7-5.4 MB per IED) and 19,478 reads (every logical node with each functional constraint it
  has) identical after masking times, except one per IED: the buffered report blocks' EntryID,
  which libiec61850 seeds from the worker's start time in milliseconds.
- Memory (private bytes of the server and its child processes, two rounds each): idle after
  opening 357 MB -> 372-377 MB (the spare `studio_goose` holds 12 MB). Go heap profiles taken
  after a 20 s Run and Stop: new 351 MB, old 377 MB; after a 3 s Run the new engine kept 12 MB
  more because its GOOSE monitor already ran and built its per-model index
  (`acquisition.refIndex`, 15 MB), which the old engine builds a few seconds later in any Run.
- Protocol suite against the rebuilt workers: 72 passed, 9 skipped; 9 files need the frozen
  Python backend (Flask is not installed in this checkout) and did not load.
- Page: served with the policy, renders the Devices and diagram pages with no blocked resource and
  Windows fonts; every request goes to 127.0.0.1.

## Release (0.1.31 rebuilt and published again, same version)

- `compile_go_release.py --version 0.1.31` with `studio\.venv` (the packer needs Flask): antivirus
  gate 19 programs, 0 findings. Packaged app on the project copy: page data 0.98 s, Start 2.53 s
  and 4.09 s, Stop 0.39 s and 0.80 s.
- Protocol files that need Flask, run with `studio\.venv`: 18 passed (with the rest: 90 passed,
  9 skipped).
- Changelog (`website/src/pages/changelog.astro`): title "Faster Start and more reliable network
  setup", two Faster lines and an offline line added. `website/deploy.cmd`: Pages 9992de50, "Saved
  release 0.1.31 from 2026-11-01".
- Live installer 2c090c48 (19,557,335 bytes) and portable 8eb23467 (53,122,048 bytes) match the
  build; the signed release record carries both hashes and the new notes line; copies of both
  downloads scan clean (101 objects, 0 detected).

## Limitations

- Timings move with the PC's load; every comparison alternates old and new rounds and reports
  medians. A Start in the same minute as Windows DNS Client activity can take 2-3 times longer.
- On the owner's PC each Start and Stop that adds or removes Ethernet IED addresses (ADR-0132)
  sets Windows DNS Client to 5-11 processors for 20-30 s, with old and new engines alike;
  another program there lists all 17 network cards 4 times a second. Not changed here.
- A Start in the first seconds after opening can still wait for the spare GOOSE engine (3 s) or
  the idle read (0.5 s, then about 1.5 s) that is running.
