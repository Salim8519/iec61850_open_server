# 0133: Speed lane N1 merged: studio_ied precise clock and high-resolution waits; IPC hot path

Date: 2026-10-20. [ADR-0088](../../decisions/ADR-0088-ied-timing-without-machine-timer.md); speed
plan phase 3, lanes N1 + N4. Branch `speed-n1` (61f26fb, 618b2be, 3ca9eed, 2b65cf0) merged into
`master` (438359f). Stream log: [speed-n1](../../architecture/go-engine/streams/speed-n1.md).
Applies to both engines once workers are rebuilt in the app folder (not done: the owner's app
uses `studio/native/build/Release`).

## Change

- `hal/time_studio.c`, `hal/studio_wake.c`, `hal/loop_wait.h`, `worker/server/event_loop.*`,
  `worker/server/timer_policy.*`, `worker/main.cpp`, `cmake/event_loop_support.cmake`: precise
  clock, event + high-resolution waitable timer loop wait, exact millisecond deadlines (no +-1 ms
  GOOSE candidates), power-throttling opt-out, `timeBeginPeriod(1)` only as fallback.
- `worker/server/server.*`, `worker/updates/values.cpp`: reference -> `DataAttribute*` cache per
  start (`update_batch` resolves each ref once). `worker/ipc/framing.cpp`: `ReadFile` into a
  reused buffer; frames, errors and bytes unchanged.

## Evidence

- Before = master build, after = N1 build, interleaved runs on a loaded machine: GOOSE MinTime 4
  largest burst gap 4.5-5.0 -> 4.1-4.4 ms, 0 frames past TAL in all client mixes; MaxTime repeats
  999.8-1000.6 ms; report latency p50 0.55 -> 0.45 ms; SBO timeout 200.0-201.3 ms; integrity
  spacing p50 499.7-499.9 ms; MMS read p99 with idle clients 0.24-1.00 ms; idle CPU per worker
  equal or lower (20 workers 2-3.4 % of one core, both builds); `update_batch` within noise.
- Timer path proof: high-resolution 4.64 ms max gap, fallback 4.95 ms, neither 16 ms with 11 late.
- `native_wire_compare`: WIRE IDENTICAL (5 masked MMS streams, first 14 GOOSE frames).
- Protocol suite on a remapped copy (127.67.0.x, 28xxx): 58 passed, 5 skipped.
- Integrator: merged with streams C and G native changes without conflict; merged sources incl.
  gateway/master built in `studio/native/build/integrate` (see ledger line); `verify.py`: see
  ledger line.

## Limitations

- The default 15.6 ms system resolution never occurred during the session (another app held
  1 ms); proven by per-process waits and the negative control. One confirmation run on a quiet
  machine is pending. Gateway/master workers were not exercised at runtime by the lane.
