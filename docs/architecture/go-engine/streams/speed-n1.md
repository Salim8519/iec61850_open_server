# Speed lane N1: simulated IED timers without `timeBeginPeriod(1)`

Branch `speed-n1` from master a41879c ([speed plan](../speed-plan.md), phase 3, items N1 and the
cheap part of N4). Owns `studio/native/hal/**`, `studio/native/worker/**`,
`cmake/event_loop_support.cmake` and the `studio_ied` part of `CMakeLists.txt`. Native workers
keep owning wire protocol state; no IPC, project, API, event or wire change.

## 1. Problem

`studio_ied` raised the machine-wide Windows timer resolution to 1 ms for its lifetime
(`timeBeginPeriod(1)`, [ADR-0070](../../../decisions/ADR-0070-goose-visible-between-machines.md))
so that GOOSE MinTime retransmissions stay inside their TAL. That costs power on every laptop
running the app and still left two timing sources coarse whenever the resolution is default:

- `Hal_getTimeInMs` (vendor `hal/time/win32/time.c`) reads `GetSystemTimeAsFileTime`, which
  only advances at the system tick (15.6 ms by default). GOOSE `nextPublishTime`, SBO, report,
  integrity timers and every timestamp derive from it.
- The [ADR-0079](../../../decisions/ADR-0079-event-driven-ied-server-loop.md) loop waited in
  `select()`, whose timeout rounds to the tick; so did the 1 ms retry waits.

Measured on this laptop (Windows 11 build 26300), a process that does not call
`timeBeginPeriod` gets 15.3-15.8 ms for `Sleep(1)` and a 1 ms `WaitForSingleObject` even while
another application holds the system timer at 1 ms (Windows 11 per-process timer resolution).

## 2. Change

- **Precise clock** `hal/time_studio.c`: the vendor `time.c` compiled unmodified under
  `Vendor_Hal_getTimeInMs/Ns` (ADR-0006 pattern, as `socket_studio.c`); `Hal_getTimeInMs` and
  `Hal_getTimeInNs` read `GetSystemTimePreciseAsFileTime` with the vendor's own conversions.
  `time.c` is pinned by SHA-256 in `cmake/event_loop_support.cmake`. The HAL is shared, so
  every executable linking it (studio_live, studio_control, studio_probe, test tools) gets the
  same UTC clock with sub-microsecond instead of tick resolution.
- **Loop wait** `hal/studio_wake.c` + `hal/socket_studio.c`: in wait mode the loop thread
  associates each server socket once with a manual-reset event (`WSAEventSelect`, `FD_ACCEPT |
  FD_READ | FD_CLOSE`; accepted sockets inherit the listening socket's association), resets the
  event, polls with the vendor `select(0)` (anything already pending returns at once), then
  `WaitForMultipleObjects` on {socket event, auto-reset wake event, high-resolution waitable
  timer}. The timer comes from `CreateWaitableTimerExW(CREATE_WAITABLE_TIMER_HIGH_RESOLUTION)`
  and is set to the relative time to the deadline in 100 ns units; it is cancelled when the
  wait ends early. Processing keeps the vendor `select(0)` poll mode. The UDP loopback wake
  socket is replaced by the wake event (`SetEvent`).
- **Deadlines** `worker/server/event_loop.cpp`: deadlines are millisecond boundaries of the
  precise clock (`ms * 1e6` ns), i.e. the first instant `Hal_getTimeInMs()` returns that value.
  GOOSE wakes at exactly send time + MinTime/MaxTime (the vendor's `currentTime >=
  nextPublishTime` first holds there), so the +-1 ms wake candidates are gone. The strict-`>`
  retry waits for the next millisecond boundary. 50 ms safety wake unchanged.
- **Timer policy** `worker/server/timer_policy.cpp`: the worker opts out of Windows power
  throttling (`ProcessPowerThrottling`, ControlMask `EXECUTION_SPEED | IGNORE_TIMER_RESOLUTION`,
  StateMask 0; Windows 10 retries with execution speed only). If Windows cannot create a
  high-resolution timer (before Windows 10 1803), the worker calls `timeBeginPeriod(1)` for its
  lifetime exactly as before and writes one stderr line, which the engines deliver as one
  `native_diagnostic`: "High-resolution timers are not available on this Windows version; using
  the 1 ms system timer for GOOSE and control timing".
- **N4 IPC hot path**: `Server::attribute` caches reference -> `DataAttribute*` per start
  (cleared on stop; IPC thread only), `update_batch` resolves each reference once (was three
  tree walks); `read_frame` reads the stdin pipe with `ReadFile` into a reused buffer instead of
  `std::cin.read` (shared by every worker using `worker/ipc/framing.cpp`). Same frames, errors
  and bytes.

## 3. Evidence

Laptop Ryzen 9 5900HS, Windows 11 26300, 2026-10-20; other agents were building and another
lane's app instance (4 IEDs) was running, so all comparisons are interleaved or concurrent.
Before = master a41879c built in its own folder (`build/speed-n1-base`), after = this branch
(`build/speed-n1`), both `-DSTUDIO_GATEWAY=OFF`, each tool from its own folder. Addresses
127.67.x.y, ports >= 20000, synthetic template and public `scd/open_substation.scd` only; every
process started was stopped; the owner's app and `build/Release` were not touched.

**System timer resolution during the runs**: `NtQueryTimerResolution` reported **1.0 ms** in
every probe (14 probes, 03:24-04:53): another application holds it (the Claude desktop app and
WebView2 processes were the only candidates running; no studio_ied was running). The default
15.6 ms could therefore not be recorded. What matters for the worker still applies: in a process
that does not call `timeBeginPeriod`, `Sleep(1)` and a 1 ms `WaitForSingleObject` took
15.3-15.8 ms (median) in every quiet probe, i.e. Windows 11 gives each process the default timer
behaviour for its own waits. The negative control below shows the after build relies on the
high-resolution timer, not on the 1 ms global resolution.

**Final runs C and C2** (`native_speed_bench.py` + scratch `n1_extras.py`; order base, after,
after, base; cells "C ; C2"):

| | before | after |
|---|---|---|
| GOOSE MinTime 4: burst gap max (ms), 0 clients | 5.03 ; 4.54 | 4.37 ; 4.41 |
| same, 2 active MMS clients | 4.98 ; 5.04 | 4.07 ; 4.16 |
| same, 1 active + 2 idle clients | 4.76 ; 4.53 | 4.20 ; 4.09 |
| frames past TAL (12 ms), all three | 0 ; 0 | 0 ; 0 |
| MaxTime 1000 repeats (36 gaps) | 999.49-1001.26 ms | 999.81-1000.59 ms |
| report latency, IPC update -> report bytes, p50 / max (36 each) | 0.55 / 1.08 ; 0.55 / 0.97 | 0.44 / 1.55 ; 0.46 / 1.10 |
| integrity spacing IntgPd 500, p50 / max | 499.6 / 500.9 ; 499.4 / 500.9 | 499.7 / 500.2 ; 499.9 / 500.3 |
| SBO select timeout (sboTimeout 200, enhanced, 20 runs) min-max | 200.0-202.8 ; 200.2-202.1 | 200.2-201.3 ; 200.0-201.3 |
| MMS read p99 ms, 1 active + 1 / 2 / 4 idle | 0.28/0.24/0.28 ; 0.69/0.49/0.26 | 0.24/0.24/0.25 ; 1.00/0.76/0.50 |
| MMS read p50 ms, 1 active / 5 active | 0.13 / 0.21 ; 0.12 / 0.34 | 0.12 / 0.23 ; 0.19 / 0.38 |
| SBO normal / enhanced p50 ms, 0 idle | 0.25 / 0.42 ; 0.34 / 0.61 | 0.25 / 0.42 ; 0.29 / 0.46 |
| update_batch 1 attr, us per request | 203 ; 194 | 198 ; 167 |
| update_batch 28 distinct attrs (IED1_XCBR), us/attr | 17.5 ; 17.9 | 18.8 ; 15.6 |
| update_batch 256 attrs, attrs/s (us/attr) | 119685 (8.4) ; 107888 (9.3) | 110723 (9.0) ; 134444 (7.4) |
| idle CPU 20 workers, % of a core: no GOOSE / GOOSE / 1 idle client each | 3.23/2.76/3.12 ; 3.28/4.17/1.93 | 2.34/3.39/2.08 ; 2.66/2.97/3.07 |

Read/SBO single ~15-17 ms p99 values appear in both builds (the libiec61850 client's 10 ms tick
under the default timer, as recorded by stream G). update_batch is timed from Python over the
pipe, so Python dominates small batches and run-to-run noise (+-15 %) hides the native saving;
the public sample model is small (227 signals), so each saved tree walk is only microseconds.

**Idle CPU, both builds at the same time** (scratch `concurrent_cpu.py`, 20 + 20 workers, 60 s,
`QueryProcessCycleTime`, because `GetProcessTimes` counts in 15.6 ms ticks, ~2 ticks per worker
here): Mcycles/s per worker before / after: no GOOSE **4.18 / 4.11**; GOOSE publishing
**5.25 / 4.68** (one GOOSE wake per frame instead of three); one idle client each **4.24 / 4.24**
(~0.1 % of a core per worker; ADR-0079 recorded 1.93 % for 20 workers on a quiet machine).

**Timer path proof** (binary 618b2be with temporary environment switches, removed before
commit; GOOSE MinTime 4, 5 bursts, same session):

| variant | burst gap max / mean | late > TAL | MaxTime | diagnostic |
|---|---|---|---|---|
| after (high-resolution timer) | 4.64 / 4.02 | 0 | 999.8-1000.5 | none |
| fallback: no high-resolution timer -> timeBeginPeriod(1) | 4.95 / 4.31 | 0 | 1000.0-1001.4 | the one line above |
| negative control: neither | 16.05 / 13.14 | **11** | 1004-1015 | none |

**Wire identity** (`native_wire_compare.py BASE NEW`, final binaries): five MMS server-to-client
streams identical after masking (9570, 198, 746, 198, 758 bytes; 4 reports each); probe outputs
identical; first 14 GOOSE frames identical in order after masking `t`: **WIRE IDENTICAL**.

**Protocol suite** (`studio/tests/protocol` copied with 127.50.0.x -> 127.67.0.x and ports
18xxx -> 28xxx, copy deleted after; gateway/master/converter/SV deselected as in stream G;
`STUDIO_NATIVE_BIN` = the build, with studio_live, studio_control, studio_sv_test_publisher and
studio_goose.exe built into the same folder): after **58 passed, 5 skipped** (GOOSE
share/conflict need `STUDIO_GOOSE_ADAPTER`), 25 deselected, on 618b2be and again on 3ca9eed.
Before: 57 passed, 1 failed (`test_interlock_hints_runtime`; passed on two reruns, flaky),
5 skipped. GOOSE engine latency test after: p50 0.102-0.118 ms, p99 0.202-0.339 ms.

**Wait mechanism under load** (scratch `wait_compare.py`, both in parallel, 20 s): 4 ms / 50 ms
waits, `timeBeginPeriod(1)` + millisecond wait vs high-resolution timer: late p50 0.44/0.43 vs
0.47/0.41 ms, p99 4.0/3.0 vs 4.7/4.1 ms; with 24 busy processes both reach 90-120 ms late.
Earlier sequential runs (A, B, superseded) showed after-only report outliers of 48-112 ms while
`Sleep(1)` took 45-75 ms machine-wide; they did not reproduce in 4 standalone repeats, 2
synthetic-load runs or 2 concurrent runs (base and after measured together: no value > 10 ms,
after max 5 ms) nor in runs C/C2, so they are attributed to machine load.

## 4. Proposed ADR (number to assign at merge)

**ADR-00xx: Simulated IED timing without a machine-wide timer request**

*Problem.* ADR-0070 made every `studio_ied` raise the Windows timer resolution to 1 ms for its
lifetime, which costs power machine-wide. Without it, the vendor clock (`GetSystemTimeAsFileTime`)
and the ADR-0079 loop's `select()` timeouts move in 15.6 ms steps: GOOSE MinTime 4 ms frames
leave 13-16 ms apart, past their 12 ms TAL (measured: 11 of 15 burst gaps late).

*Decision.* (1) The HAL clock `Hal_getTimeInMs/Ns` reads `GetSystemTimePreciseAsFileTime`
(`hal/time_studio.c`: vendor `time.c` compiled unmodified under renamed symbols, hash-pinned).
(2) The server loop sleeps in `WaitForMultipleObjects` on socket events (`WSAEventSelect`), a
wake event and a `CREATE_WAITABLE_TIMER_HIGH_RESOLUTION` timer set to the next millisecond
boundary of the precise clock; processing keeps the vendor `select(0)` poll. (3) The worker opts
out of power throttling (execution speed, timer resolution). (4) Only where Windows cannot create
high-resolution timers (before Windows 10 1803) the worker calls `timeBeginPeriod(1)` as before
and reports it once as a `native_diagnostic`. The ADR-0070 TAL rule is unchanged and met.

*Compatibility.* No IPC, project, API, event or wire change (masked MMS streams and GOOSE frames
identical). Timestamps carry the same UTC clock with finer resolution; every executable linking
the HAL gets the precise clock. Supersedes ADR-0070 decision 2 (1 ms timer for the worker's
lifetime) except as the fallback. Windows only, like ADR-0079.

## 5. Limitations and open points

- The default 15.6 ms system resolution never occurred during the session (another app held
  1 ms), so "after" was proven by per-process waits and the negative control, not on a machine
  at 15.6 ms. Worth one confirmation run with nothing raising the resolution.
- The fallback was exercised by forcing the timer to NULL on 618b2be; the last commit 3ca9eed
  changed only socket association, not the fallback path.
- Precise time changes timestamp *resolution* (no longer tick aligned) in all native
  executables; values and encodings are unchanged.
- Under heavy machine load a wake can be tens of milliseconds late with either mechanism; the
  worker does not raise its thread priority (unchanged from before).
- `framing.cpp` is shared by studio_live, studio_control, studio_gateway, studio_master and the
  SV test publisher; gateway/master were not built (`STUDIO_GATEWAY=OFF`) or tested here.
- Rebuilding the workers into the app folder needs the owner's app closed and the owner's OK
  (speed plan phase 3). No EXE was built.
