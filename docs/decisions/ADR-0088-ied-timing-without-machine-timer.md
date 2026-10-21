# ADR-0088: Simulated IED timing without a machine-wide timer request

Accepted: 2026-10-20 ([speed plan](../architecture/go-engine/speed-plan.md) phase 3, lane N1;
owner: "ultra lightweight on CPU", "top notch network performance"). Evidence: change
[0133](../changes/2026/0133-speed-lane-n1-ied-timers.md), log
[speed-n1](../architecture/go-engine/streams/speed-n1.md). Builds on
[ADR-0079](ADR-0079-event-driven-ied-server-loop.md); supersedes ADR-0070 decision 2 (1 ms timer
for the worker's lifetime) except as the fallback below.

## Problem

ADR-0070 made every `studio_ied` raise the Windows timer resolution to 1 ms for its lifetime,
which costs power machine-wide. Without it, the vendor clock (`GetSystemTimeAsFileTime`) and the
ADR-0079 loop's `select()` timeouts move in 15.6 ms steps: GOOSE MinTime 4 ms frames leave 13-16 ms
apart, past their 12 ms TAL (negative control: 11 of 15 burst gaps late).

## Decision

1. The HAL clock `Hal_getTimeInMs/Ns` reads `GetSystemTimePreciseAsFileTime`
   (`hal/time_studio.c`: vendor `time.c` compiled unmodified under renamed symbols, hash-pinned).
2. The server loop sleeps in `WaitForMultipleObjects` on socket events (`WSAEventSelect`), a wake
   event and a `CREATE_WAITABLE_TIMER_HIGH_RESOLUTION` timer set to the next millisecond boundary
   of the precise clock; processing keeps the vendor `select(0)` poll.
3. The worker opts out of power throttling (execution speed, timer resolution).
4. Only where Windows cannot create high-resolution timers (before Windows 10 1803) the worker
   calls `timeBeginPeriod(1)` as before and reports it once as a `native_diagnostic`.

The ADR-0070 TAL rule is unchanged and met.

## Compatibility

No IPC, project, API, event or wire change (masked MMS streams and GOOSE frames identical).
Timestamps carry the same UTC clock with finer resolution; every executable linking the HAL gets
the precise clock. Windows only, like ADR-0079. The Go GOOSE engine stopped raising the timer in
change 0129, so after a rebuild no ScadaBench process raises the machine timer.
