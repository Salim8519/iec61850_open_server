# ADR-0133: Faster Run start and opening (value-cache index, overlapped steps, spare GOOSE engine) and an offline page policy

Accepted: 2026-11-01. Owner: "31 still takes time to start, why?", then "go bro do all phases, fix and
test", "run 3 threads or something so light pc handles", "make sure the app ui is completely
offline, logos, designs, and all components are offline". Change
[0208](../changes/2026/0208-faster-run-start-offline-page.md). Builds on
[ADR-0129](ADR-0129-run-start-without-waiting.md) and [ADR-0006](ADR-0006-native-array-parser.md).

## Problem

Run took 10-12 s on the owner's 7-IED project on an idle 16-thread laptop and 17-24 s in the
owner's own runs. Measured, step by step:

- Inside each `studio_ied`, `IedServer_createWithConfig` took 0.45-1.43 s of a 0.5-1.6 s start;
  90 % of it was `installDefaultValuesInCache`. The pinned libiec61850 value cache keeps one entry
  per logical node in a linked-list map, and every lookup that misses (each parent level of a
  data attribute name) walks the whole list, then scans the domain's variables again: about
  50 000 attributes x hundreds of entries per IED. Controls and deadbands added only ~0.3 s.
- The coordinator waited for each step before the next: lab addresses, GOOSE engine start,
  workers, start values one write at a time, bindings, start states, test modes, then the
  listeners (GOOSE monitor, SV, real IEDs).
- Starting a fresh `studio_goose` inside the busy start took 1-1.7 s to answer "hello" (alone it
  answers in 0.09 s).
- A memory trim (`debug.FreeOSMemory`, ADR-0116) due 2 s after a Stop could run in the middle of
  the next Start.

Loading the IEDs a fixed "3 at a time" was measured slower on the owner's PC (3.1 s against 1.9 s
for all 7 at once); it was not adopted.

## Decision

1. **Constant-time value-cache lookups (native).** `cmake/value_cache_index.cmake` compiles a
   generated copy of the pinned `mms_value_cache.c` (hash-checked, like the ADR-0006 parser
   correction; the vendor tree stays untouched). Every lookup asks
   `libfix/value_cache_index.c` first, which returns exactly the entry the vendor map would (the
   first added under the key); after an allocation failure lookups go back to the vendor map. A
   lookup that found an entry reuses that entry's stored type instead of scanning the domain
   again (it was found with the same name at insertion). The biggest IED starts in 0.36 s instead
   of 1.5-1.8 s; all seven at once in 0.75 s instead of 2.0 s. Reads served from the cache use the
   same index.
2. **Independent steps overlap.** A fresh Run opens the GOOSE bus while the lab addresses are
   prepared, and starts the listeners (they only listen) at the same time; IED processes start
   after the bus, so a slow PC never starts the GOOSE engine and every IED at the same moment (a
   measured "hello" timeout under 100 % CPU). Errors keep the old order (lab, preflight, bus, IEDs,
   bindings, states, test modes, listeners); a failed Run waits for the started steps, then stops
   as before. Adding an IED to a running simulation stays sequential, so a refused addition never
   registers publishers in the running engine.
3. **Start values in one batch per IED.** The last positions, commands, qualities and changed
   times are written with `update_batch` (one data-model lock, ADR-0115) instead of one request
   each; the worker validates the whole batch before changing anything, so a refused value still
   fails the Run.
4. **A spare GOOSE engine.** After every Stop, and 3 s after the application starts, a fresh
   `studio_goose` is started and greeted in the background; the next Run takes it instead of
   starting one. It is a new process every Run as before (ADR-0073), only started earlier; it is
   not used when the program file changed since it started, and it is closed at shutdown. Idle it
   uses no CPU (its share ticker stops when nothing is shared).
5. **No memory trim during Start or Stop.** `memtrim.Hold` keeps a due collection away until the
   job ends, then schedules it as before.
6. **The page loads nothing from the internet.** The engine serves the page with a
   Content-Security-Policy that allows only itself (`httpapi/offline_policy.go`); the Vite dev
   server sends the same policy plus its own reload needs. `studio/tools/check_offline_ui.py`
   (part of `verify.py`) fails on any web address in the interface source that is not a listed
   click link (tutorials, Npcap download) and on any font, style, script or image from an
   address. Logos, icons and fonts were already local (Windows fonts, `/assets/brand`).
7. **Opening a project waits less.** Every request waits for the stored project to be restored
   (`httpapi.waitForStartup`). Goroutine samples showed two costs inside it:
   - the cached models of all sources (11-15 MB of JSON per IED) were read and decoded one after
     another. `project/service/cached_reads.go` now reads them together, at most one per
     processor; ids, imports and errors still follow the stored order.
   - the diagram check looked up every bound point with a linear scan over all IEDs.
     `ValidateDiagram` now uses the same `project.Index` lookup as the rest of the engine
     (`project/lookup.go`: identical answers to the linear scan).
8. **Start data is read while the app is idle.** The first Start after opening read every IED's
   SCL text again for its typed controls and writable deadbands (ADR-0104, ADR-0118) while the
   seven IED processes loaded: a text that reads in 0.12 s alone took 1.3-7 s there, so the
   first Start was 2-4 s slower than the next. `runtime/start_warm.go` reads them 0.5 s after the
   project opened or changed (edits in a row give one read), one text at a time on one
   processor, into the same caches the start request uses (same values; an IED it already read
   is skipped; a text it cannot read is left to the Run, which handles it as before). It stops at
   shutdown.

## Considered and not adopted

- **Keeping IED models loaded between Runs.** Each idle `studio_ied` with its model holds about
  52-58 MB of private memory: ~380 MB for 7 IEDs, ~1 GB for 20, on every PC while ScadaBench is
  open. After decision 1 an IED starts in 0.3-0.4 s, so the memory is not worth it.
- **Compressing the save cache after the page opens.** `archive.Encoder.Warm` takes 0.1-0.27 s of
  the opening and its checks (save limit, non-finite numbers) must still refuse a bad project
  before the page shows it; moving only the compression would save ~0.1 s.

## Compatibility

- Wire behaviour: the same requests reach each worker; values and data model are identical. With
  one batch, a start value change that is in a GOOSE dataset is published under one data-model
  lock instead of one per value.
- Event order: listener events can now appear before the `running` event of the same Run.
- `studio_ied`, `studio_live`, `studio_probe` are rebuilt with the index (any program linking
  libiec61850). Packaged and development builds use the same native code.
- One extra idle `studio_goose` process while ScadaBench is open.
