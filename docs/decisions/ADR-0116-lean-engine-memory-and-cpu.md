# ADR-0116: Lean engine: compact JSON objects, one SCL read per file, no throwaway work

Accepted: 2026-10-24. Owner request: "squeeze more memory while keeping the app light, low CPU
and RAM, many IEDs with less resources; the user must not notice anything; network usage and
speed the same or better". Change [0189](../changes/2026/0189-lean-engine.md). Go engine only
([ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md)); the frozen Python engine,
the native workers and every answer's bytes are unchanged. First item of
[FUTURE-WORK.md](../../FUTURE-WORK.md) section 1.

## Problem

A 20-IED vendor project (about 340,000 points) used 1.0 GB after open, 1.6-2.4 GB while running
and kept that after Stop. Heap and allocation profiles of the Go engine showed:

- 445 MB of the 470 MB live after open were IED models held as `pyjson.Object`, an ordered map:
  a Go map per point (9 keys) costs about three times its contents; 75 MB were enum lists copied
  into every Enum attribute.
- Run start allocated about 2 GB: for every IED the typed-control list (ADR-0104) parsed the whole
  SCL file again and rebuilt all its signals, although 8 IEDs shared one file.
- Each `POST /signals/source` indexed all 340,000 signals of the project (about 50 MB) to find one.
- Each SLD command built the IED's whole signal model to read one entry of its control list.
- Freed memory stayed with the process: after these one-off jobs nothing allocates, so no
  collection runs and Windows kept seeing the peak.

## Decision

1. **Compact `pyjson.Object`.** Entries are one slice of `{key, value}` in insertion order; a key
   index map exists only above 16 keys. The API (`Set`, `Get`, `Delete`, `Keys`, `Len`) and the
   Python dict rules (first place kept, last value wins, delete then set moves to the end) are
   unchanged; `NewObjectSize(n)` presizes builders. The decoder collects members on a shared
   stack and gives each object one exact-size slice.
2. **Shared read-only enum lists.** The model builder keeps one enum object per EnumType and one
   empty object for other signals; nothing writes a signal's `enums`.
3. **Typed controls read once per SCL text.** `CachedExtendedControls` asks
   `extendedForDocument`: one parse per text for all its IEDs (concurrent starts wait for the first
   reader), building only the data objects whose type tree holds an INC/ENC/APC/BAC/BSC/ISC class.
   `ExtendedControls` stays as the reference.
4. **Look up only what a request names.** Sources and scenarios index only their own (IED, ref)
   keys; commands read the control list directly (`controlOf`).
5. **Give memory back after one-off jobs.** `memtrim.Soon()` after open/restore, import, IED
   removal, Run start and Stop: two seconds after the last of them, one `debug.FreeOSMemory`.
6. **Not chosen:** a lower `GOGC` (less memory but more CPU at open and Run start) and any
   `GOMEMLIMIT` (a limit below the live data made the collector use about 10 cores in the test).

## Compatibility

- Every API answer, saved file and worker request keeps its bytes: pyjson's CPython differential
  tests and all golden suites pass; a model test drives random Set/Get/Delete sequences across the
  index threshold against the former map-backed object.
- Typed controls: byte-identical answers for 1,070 IED names in 185 SCL files (fixtures, sample
  stations, 162 vendor files, the owner's files; 186 with typed controls). The 58 differences are
  IEDs the importer refuses (the full build stops on an attribute outside the typed objects), so no
  project holds them.
- Nothing visible changes in the browser; no new setting, cache, file or dependency.

## Evidence (summary; details in change 0189)

Same 20-IED project, same activity (35 sources, 13 report blocks on 5 independent mmscli clients,
SLD commands, GOOSE sends), today's master build against the lean build: server memory after open
1,028 -> 370 MB, peak 1,595 -> 761 MB, after Stop 1,595 -> 557 MB; total server CPU 42-50 s -> 13-17 s;
open 6.2 -> 4.7 s (from cache 2.1 -> 1.2 s); Run start 3.2 -> 2.3 s; SLD command median 56 -> 49 ms;
reports per second 76.0 -> 76.0, GOOSE frames per second 32.9 -> 33.1; zero errors.

## Limits

- The 20 native IED programs (about 25 MB each) are unchanged.
- Models still hold every point as an object; a typed point store would save more but changes the
  core signal model (future, own ADR).
