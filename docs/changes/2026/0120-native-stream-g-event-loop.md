# 0120: Native stream G merged: event-driven studio_ied loop (no idle-client MMS penalty)

Date: 2026-10-20. Decision: [ADR-0079](../../decisions/ADR-0079-event-driven-ied-server-loop.md).
Branch `go-g-native` (d3002bd) merged into `master` (404d2a0). Stream log:
[g-native](../../architecture/go-engine/streams/g-native.md). Applies to both engines once the
native workers are rebuilt; the binaries in `studio/native/build/Release` are not rebuilt yet
(the owner's app uses them).

## Change

- `worker/server/event_loop.*`, `worker/server/schedule.*`: threadless vendor server driven by a
  loop that sleeps until the next control/report/GOOSE deadline or a wake from the IPC thread.
- `hal/socket_studio.c`, `hal/studio_hal.h`, `hal/ethernet_dispatch.c`: vendor `socket_win32.c`
  under renamed symbols with loop-thread wait/poll modes and a wake socket; GOOSE send times.
- `worker/server/server.*`, `worker/goose/publishing.cpp`, `worker/updates/values.cpp`: start the
  loop and wake it. `cmake/event_loop_support.cmake`: SHA-256 pins of the private vendor files.
- Test-only: `testtools/mms_bench.cpp` (`studio_test_mms_bench`, not packaged),
  `studio/tools/native_bench_rig.py`, `native_speed_bench.py`, `native_wire_compare.py`.

## Evidence

- Stream (same machine, before -> after, mean): read with 1 active + 4 idle clients 42.8 ms ->
  0.20 ms; SBO with 2 idle clients 30.4 -> 0.47 ms; enhanced SBO + CmdTerm with 2 idle clients
  60.9 -> 0.72 ms; GOOSE MinTime burst with 2 idle clients max gap 23.9 ms (4 frames over the
  12 ms TAL) -> 4.5 ms, 0 late; idle CPU of 20 workers 31.3 % -> 1.9 % of one core (3.3 % with
  GOOSE); report latency 0.64-1.72 -> 0.45-0.58 ms; integrity and SBO timeout timing unchanged.
- Wire identity through a recording proxy: 5 MMS server-to-client streams identical after masking
  UtcTime/BinaryTime, 4 reports each, first 14 GOOSE frames identical after masking `t`.
- Protocol suite on a remapped copy (127.67.0.x, ports 28xxx, owner's app untouched): 58 passed,
  5 skipped (Npcap adapter), 25 deselected (gateway/master/converter, SV adapter), two runs.
- Integrator on `master`: only native files, its tools and log; merged cleanly with stream C's
  `canvas_mux` sources in `CMakeLists.txt`; merged sources built in a scratch folder (see ledger
  line); `verify.py`: see ledger line.

## Limitations

- Depends on private fields of the pinned libiec61850 commit (hash pin forces review on upgrade);
  anything not modelled is bounded by the 50 ms safety wake. Windows-only socket wrapper.
- Connection limit still 5 per IED: a kept canvas command association (0117 open decision) uses
  one slot; raising the limit is an owner decision.
- Gateway/master tests not run (unchanged code, `STUDIO_GATEWAY=OFF` in the stream build).
- Workers in `studio/native/build/Release` still the old build until rebuilt with the app stopped.
