# ADR-0115: Simulated IED runtime: derived Beh, one batch per mode change, log volume, times and origin

Accepted: 2026-10-24. Owner-approved runtime fixes from the 24-vendor-IED runtime test (H3, M3, L1,
L3, L4, L6, L7). Change [0186](../changes/2026/0186-runtime-fixes.md), spec
[test-mode](../specs/test-mode.md) (TMODE-05, TMODE-10, TMODE-13 to TMODE-17). Go engine and the
native worker only ([ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md)): the frozen
Python engine is unchanged; where it shares the native worker it gets the native part too (noted
below). Commands keep ignoring Beh and the Test flag ([ADR-0104](ADR-0104-typed-iec61850-commands.md)
unchanged).

## Problem

Measured with an independent MMS client (libiec61850 client API, no Studio code):

- `Beh` was never derived from `Mod`: an imported model kept `Beh` as imported (often 0 for an Enum
  without `Val`) at Run, and `LLN0.Mod` operated to 3 left every `Beh` unchanged.
- Test mode on a logical device with 148 logical nodes answered 400 "Sample batch exceeds the native
  256-attribute limit" (Mod, every Beh and their times are about 300 attributes).
- The run log grew by megabytes a minute: every 10 Hz generator sample was one ~480-byte line, and
  every "report created" of a buffered RCB that no client enabled was an event.
- Every timestamp read 1970 with good quality until its first change; TimeQuality was 0x8a after a
  command (vendor default) and 0x0a after a signal write (`MmsValue_setUtcTimeMs`).
- SLD and plain `/controls/operate` commands carried orCat 0 and an empty orIdent; typed commands
  carried 3 / "ScadaBench".
- The run log recorded `"application":"0.1.0"` whatever the version.

## Decision

1. **Beh follows IEC 61850-7-4 Annex A** (`runtime/behaviour.go`, `deriveBeh` unchanged): an LN's Beh
   combines its own Mod with its LD's `LLN0.Mod`; off wins, then test and blocked combine (on-blocked +
   test = test/blocked). Studio writes it:
   - at every Run and whenever an IED joins a run, after the saved test-mode settings (which still
     win): the app's table first takes the imported Mod and Beh values again (a worker restarts from its
     SCL values while the table keeps the last run's; Mod and Beh are not restored like positions), then
     every Beh that differs is written in one batch (about 0.3 s for the 1172 Beh of project B; Run
     start 2.7-3.2 s after against 2.9-3.3 s before, same project, back to back);
   - with every Mod write Studio makes (Devices, test mode, scenarios), **in the same native batch** as
     the Mod, so a client sees one change;
   - after an MMS client's Mod command (`control_applied` with a `.Mod.stVal` status), as a separate
     write a few milliseconds later, only for the Beh that differ (`behaviour` source, event
     `behaviour_failed` with a reason if it cannot be written).
   An LD whose `LLN0.Mod` holds no mode 1-5 (an Enum without `Val` reads 0) is left as imported: nothing
   is invented. An LN without its own valid Mod follows its LD. Hierarchies through `GrRef` are not
   followed (future work if a model needs it).
2. **One native batch holds up to 4096 attributes** (`worker/updates/values.cpp`, was 256). A batch is
   applied under one data-model lock, so a mode change is one change for reports and one GOOSE state
   change. Evidence: the whole IPC round trip (JSON, validation outside the lock, update under it) is
   2.6 ms for 256, 10.4 ms for 1024 and 40.2 ms for 4096 attributes (best of 5, Sample template); the
   largest LD measured (148 LNs, project B) needs about 300 attributes for a mode change. A write above 4096 attributes is sent as several
   batches (`runtime/sample_batches.go`), never splitting one point's value, quality and time; only such
   a write can produce several GOOSE states. A guarded write (a pulse) must still fit one batch. Python
   keeps sending at most 256.
3. **Log volume** (contract change of the run log; history is never rewritten):
   - generator samples become one `signal_summary` record per IED and second: `ied_id`, `source`
     ("generator"), `second` (epoch second), `samples`, `from_id` / `to_id` (the bus event ids of the
     first and last sample: the correlation back to the live stream, which still carries every sample)
     and `values` (each point's last applied value). Every other event, manual and scenario writes
     included, stays one line with its own id, so request, applied and observed records stay distinct;
   - the native worker raises no `report_control` event for "report created" (kind 9) of an RCB that
     no client has enabled. Enabled RCBs still report it. This native part applies to Python too.
   - Measured on project B (4 vendor IEDs, 6 generator sources, 4 report clients, 120 s): 1.787 MB/min
     before, 0.055 MB/min after.
4. **Times.** At start `studio_ied` sets every data attribute named `t` of type Timestamp to the Run
   start time; the start answer carries `started_at`, `stamped_times` and `time_quality`, and the Go
   engine shows the same time for every `t` that has none yet. A time restored from the previous run is
   kept; one that was only last run's start time is not written again. **One TimeQuality, 0xAA**:
   leap seconds known, clock not synchronised (the PC clock is no IEC 61850 time source; Studio has no
   way to know it is synchronised), 10-bit accuracy. Commands (vendor `IedServer_setTimeQuality`) and
   signal writes (`MmsValue_setUtcTimeMsEx`) use the same value. This native part applies to Python too.
5. **One origin for every app command**: `studio_control`'s canvas command (SLD, point panel send,
   plain boolean `/controls/operate`) sets orCat 3 (remote control) and orIdent "ScadaBench" like the
   typed path. `studio_probe` (the Python engine's canvas command) is unchanged.
6. **Words and codes.** Out-of-range Enum ords ("9 is not a choice of … Use one of: 1 (on), …") and
   non-ASCII VisString text ("VisString text takes plain ASCII characters only …") are refused before
   the wire; a bad quality says "Quality must be a whole number from 0 to 8191 (the 13 quality bits)";
   stopping an IED that is not in the project says "Unknown IED" with code `not_found`. Status codes
   stay as the API conventions say (StudioError 400, also for `not_found`; inventory
   `http-conventions`): changing them to 404 would change every not-found answer and the Python parity
   vectors, which this change does not do.
7. **Uploads and preview.** A refused upload is read to the end before the JSON answer (Go's server
   reads at most 256 KiB of an unread body and then resets the connection; Werkzeug read it all), and
   `POST /scl/preview` works while running (it imports nothing; Python refused it).
8. The run log's `run_started` records the real app version (`application`).

## Compatibility impact

- Project format: unchanged. Saved test-mode settings keep their meaning and still win at Run.
- Wire: Beh values change where models were imported with Beh not matching Mod; timestamps read the
  Run start time with quality 0xAA instead of 1970/0x00; SLD and plain commands carry an origin.
  Clients that reject "clock not synchronised" times will now flag them.
- Run logs: new runs contain `signal_summary` instead of generator `signal` lines and no
  `report_control` kind 9 for idle RCBs; `query_runtime_logs.py --ref` also matches a summary holding
  that point. Old runs are untouched.
- Native worker: batches up to 4096; start answer has three more keys; both engines get the report and
  time changes. No Python code changed.
- API: preview while running is allowed (Go only); `POST /runtime/ieds/<unknown>/stop` answers
  "Unknown IED" (`not_found`, 400) instead of "This IED is not running" or a silent stop.
