# 0136: Speed lane N3 merged: event-driven waits in studio_control, studio_live, studio_master

Date: 2026-10-20. [ADR-0089](../../decisions/ADR-0089-master-refused-connect-fails-at-once.md);
speed plan phase 3. Branch `speed-n3` (17402a4, 5f4797c) merged into `master` (77e48cb). Stream log:
[speed-n3](../../architecture/go-engine/streams/speed-n3.md). Applies to both engines once workers
are rebuilt in the app folder (not done: the owner's app uses `studio/native/build/Release`).

## Change

- `control/control_session.cpp`: CommandTermination wait is a condition variable signalled by the
  handler (was 10 ms polling); termination object declared before the control object, removing a
  race where a late termination could touch freed memory. `control_mux_v1` unchanged.
- `live/sv_header.*`, `live/sv_traffic.*`: SV header reader waits on Npcap's receive event plus a
  stop event; the libiec61850 SV receiver runs threadless from that loop (vendor thread fallback).
- `live/goose_inbox.*`: socket event + stop event instead of a 50 ms `select` loop.
- `live/goose_traffic.*`, `live/sv_traffic.*`: supervisors (250 ms / 500 ms) wait on a condition
  variable that `stop()` wakes; cadence while running unchanged.
- `master/socket_60870.c`, `cmake/master_socket_support.cmake`: refused IEC 104 connects fail at once.

## Evidence

| Measurement | Before | After |
|---|---|---|
| Enhanced operate direct / SBO (mean, 200 ops) | 30.9 / 30.9 ms | 0.90 / 1.12 ms |
| SV listener idle: wakes/s, Mcycles/s | 177-185, ~16 | 4.6-5.0, 0.76 |
| SV listener at 4000 frames/s: wakes/s, Mcycles/s | ~295, ~120 | 113-138, 66-70 |
| GOOSE inbox idle wakes/s | 16.4 | 0 |
| `goose_stop` / `sv_stop` / both (mean) | 102-157 / 327-388 / ~470 ms | ~0.8 / ~0.7 / ~1.2 ms |
| Master connect refused (t0 10 s) / unreachable (t0 5 s) | 10.1-10.3 s / 5.4 s | 0.07-0.10 s / 5.07 s |

- Operate results identical (400 per model); SV reports identical; master error text and
  `master_link` events identical; studio_control wire bytes identical after masking UtcTime;
  `native_wire_compare`: WIRE IDENTICAL.
- Protocol suite on a remapped copy incl. gateway, master, converter, GOOSE engine: 79 passed,
  9 skipped (adapter). `test_sv_listen` on VMnet1 4/4; Go native/app/toolsapi tests against the
  new workers pass.
- Integrator: merged natives incl. gateway/master built in `studio/native/build/integrate2`.
  `verify.py` first failed in `TestGatewayServesValuesAndCommandsOverIEC104`: the test used fixed
  port 20404, held at that moment by the Q2 lane's own gateway, so the test's master read the
  wrong outstation. Gateway and master wire tests now take a free port >= 20000
  (`internal/testutil.FreePort`); 3/3 runs pass with the other gateway running; `verify.py` 7/7.

## Limitations

- A stopping supervisor no longer runs one last tick (no stray `goose_stats`/`sv_rx` during Stop).
- libiec61850 MMS client receive thread still wakes ~64/s per open association (vendor; needs an
  ADR to change). Refused-connect text still says "No answer ... within 10 s" (UI text unchanged).
- No real IED or real IEC 104 outstation in the tests (loopback, TEST-NET, VMnet1).
