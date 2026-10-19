# Stream G: native IEC 61850 speed

Branch `go-g-native` (from `master` c983a8f). Scope: [Native work](../README.md) items 1 and 2
(multi-client MMS server, event-driven idle loop) with protocol behaviour unchanged.
Evidence below was measured on the owner's laptop (Ryzen 9 5900HS), 127.67.x.y, ports >= 20000,
synthetic template IEDs and the public `scd/open_substation.scd` only; every process stopped.

## 1. Root cause (measured, then read in the pinned libiec61850 source)

- The penalty is per **idle** connected client, not per active one. Single-threaded server
  (`CONFIG_MMS_SINGLE_THREADED=ON`): `handleClientConnections` visits every connection when any
  socket is ready, and `CotpConnection_readToTpktBuffer` -> `readFromSocket` waits
  `Handleset_waitReady(conn, 10)` on each one. An idle client therefore blocks the only server
  thread ~10 ms per request of every other client (1 active + 1/2/4 idle: 10.7/30.4/43.5 ms).
  Two or five *active* clients stay fast (0.23/0.52 ms) because their sockets have data.
- The same block delays everything else that thread does: operate, CmdTerm, report sending and
  GOOSE retransmission (operate with 2 idle clients: 30 ms normal SBO, 60 ms enhanced SBO).
- Idle CPU: the server thread loops `select(1 ms)` + periodic tasks, and the worker keeps the
  1 ms Windows timer (ADR-0070), so each idle worker wakes ~1000 times/s.

Threaded MMS (`CONFIG_MMS_SINGLE_THREADED=OFF`) was rejected by reading the vendor code:
connection threads tick only every 10 ms, and reports are sent only from that tick
(`Reporting_sendReports`, "to be called only by connection thread"), so report latency would
grow from <=1 ms to <=10 ms; and its accept thread still polls `select(1 ms)` under the 1 ms
timer, so idle CPU would not drop.

## 2. Change

`studio_ied` keeps the vendor single-threaded MMS server and drives it with its own loop
(`IedServer_startThreadless` + `IedServer_waitReady` / `processIncomingData` /
`performPeriodicTasks`). One thread still does all MMS handling, ticks and periodic tasks.

- `hal/socket_studio.c`: the vendor `socket_win32.c` compiled unmodified under renamed symbols
  (ADR-0006 pattern, as `ethernet_pcap.c`). Thread-local modes, used only by the loop thread:
  *wait* (the server `select` also wakes on a loopback UDP wake socket) and *poll* (zero
  timeout while handling traffic the loop already waited for, so idle clients cost nothing).
  Every other thread and executable gets the vendor functions unchanged. `Socket_write` notes
  partial sends so the loop keeps 1 ms ticks while the vendor retries them.
- `worker/server/schedule.c` reads the vendor's own schedule (private fields of the pinned
  commit, like `read_hook.c`, ADR-0066): `nextControlTimeout`; per report block the GI flag,
  `nextIntgReportTime`, `reportTime` of triggered reports (enabled or buffering BRCBs) and queued
  entries; open setting-group edits. `cmake/event_loop_support.cmake` pins these headers and
  `socket_win32.c` by SHA-256 and requires the single-threaded build.
- GOOSE retransmission time is private to `mms_goose.c`; the Ethernet HAL dispatcher tells the
  loop when a frame leaves, and the loop wakes at send time + MinTime and + MaxTime of the
  enabled GoCBs (public getters), +-1 ms.
- Wake-ups from other threads: IPC `update`/`update_batch` (reports and GOOSE events queued on
  unlock), GOOSE enable, GoEna writes. Longest sleep 50 ms as a safety net.
- Loop pass: wait -> requests -> periodic tasks -> requests again (ticks send what the periodic
  tasks queued). A due deadline the vendor leaves unchanged (strict SBO timeout `>`, model locked)
  is retried every 1 ms, as the vendor loop did.
- Unchanged: IPC protocol, start reply and capabilities, events, `timeBeginPeriod(1)` (1 ms
  `Hal_getTimeInMs` timestamps and waits), data-model locking, connection limit, stop sequence
  (`Reporting_deactivateAllReports` then stop listening, same RCB events).

Test-only additions: `studio_test_mms_bench` (`native/testtools/mms_bench.cpp`, never
packaged), `studio/tools/native_bench_rig.py`, `native_speed_bench.py`, `native_wire_compare.py`.

## 3. Evidence

Before = `master` binaries built in this worktree; after = this branch; same machine, same run
(`python studio/tools/native_speed_bench.py BIN OUT.json`, 2026-10-20). Read = back-to-back
`readObject` of one ST attribute (300 per client); clients are separate libiec61850 connections.

| MMS read, ms (mean / p50 / p99) | before | after |
|---|---|---|
| 1 active | 0.17 / 0.12 / 0.24 | 0.19 / 0.13 / 0.31 |
| 1 active + 1 idle | 10.64 / 10.47 / 11.78 | 0.16 / 0.15 / 0.33 |
| 1 active + 2 idle | 30.15 / 30.09 / 32.23 | 0.16 / 0.15 / 0.53 |
| 1 active + 4 idle | 42.82 / 41.51 / 57.75 | 0.20 / 0.14 / 0.29 |
| 2 / 3 / 5 active | 0.22 / 0.32 / 0.50 | 0.17 / 0.19 / 0.30 |

| Control round trip (100 ops), ms mean / p50 | before | after |
|---|---|---|
| SBO normal (select + operate), 0 idle | 0.25 / 0.24 | 0.41 / 0.25 |
| SBO normal, 2 idle clients | 30.38 / 30.49 | 0.47 / 0.31 |
| SBO enhanced (+ CmdTerm), 0 idle | 0.42 / 0.40 | 0.60 / 0.61 |
| SBO enhanced, 2 idle clients | 60.90 / 61.11 | 0.72 / 0.56 |

Repeated enhanced runs (3 x 300): before 0.49-0.57 ms mean, after 0.61-0.63 ms. Single ~15 ms
outliers appear in both builds: the libiec61850 client's 10 ms tick sleep (Windows 15.6 ms timer).

| GOOSE, MinTime 4 / MaxTime 1000 (5 changes, ADR-0070 method) | before | after |
|---|---|---|
| no MMS clients: burst gaps max / mean, late > TAL 12 ms | 4.90 / 4.12, 0 late | 5.85 / 4.34, 0 late |
| 2 active MMS readers | 4.99 / 4.07, 0 late | 5.08 / 4.24, 0 late |
| 1 active + 2 idle MMS clients | **23.87 / 8.10, 4 late** | 4.51 / 3.95, 0 late |
| steady repeats | 999.3-1020.4 ms | 999.1-1001.0 ms |

| Idle CPU, 20 workers, 30 s (% of one core, total / per worker) | before | after |
|---|---|---|
| no GOOSE | 31.3 / 1.57 | 1.93 / 0.10 |
| GOOSE publishing (1 GoCB each, MaxTime 1000) | 30.9 / 1.54 | 3.33 / 0.17 |
| each worker holding one idle client connection | 37.2 / 1.86 | 2.29 / 0.12 |

An earlier baseline run measured 43 % for 20 idle workers (machine load varies); the capacity
audit recorded 27 %.

Other timing (base vs after): report latency, IPC `update` to first report bytes at the client,
0.64-1.72 ms vs 0.45-0.58 ms; integrity spacing (IntgPd 500) 500.2/500.3 vs 500.4/499.4 ms;
enhanced SBO select-to-timeout (sboTimeout 200 ms, 10 runs) 199.5-201.8 vs 199.7-201.7 ms.

**Wire identity** (`python studio/tools/native_wire_compare.py BASE NEW`): browse, read, operate,
read, and an unbuffered RCB with GI plus three IPC data changes, all through a recording proxy.
All five MMS server-to-client byte streams identical after masking UtcTime/BinaryTime
(9570, 198, 746, 198, 758 bytes; 4 reports each); probe outputs identical; the first 14 GOOSE
frames (3 state changes) identical in order after masking `t`.

**Protocol tests**: `studio/tests/protocol` copied with addresses moved to 127.67.0.x and ports
18xxx -> 28xxx (the owner's app on 127.50.0.x was never touched), `STUDIO_NATIVE_BIN` = this
branch's build, gateway/master/converter/SV tests deselected (not built, unaffected):
**58 passed, 5 skipped** (GOOSE share/conflict wire tests need `STUDIO_GOOSE_ADAPTER` and
Npcap), 25 deselected; two runs, 97 s and 88 s. GOOSE engine latency test (10 IEDs x 300
rounds): p50 0.123 / 0.078 ms, p99 0.676 / 0.176 ms.

## 4. Proposed ADR (number to assign at merge)

**ADR-00xx: Event-driven simulated IED server loop**

*Problem.* The single-threaded libiec61850 server blocks ~10 ms per idle connected MMS client on
every request (41-58 ms per read with four idle clients) and delays operate, reports and GOOSE
retransmission by the same amount; 20 idle workers used 40 % of a core polling every 1 ms.

*Decision.* `studio_ied` runs the vendor single-threaded server threadless from its own loop. The
loop sleeps on the server sockets plus a wake socket until the next deadline of the vendor's own
control/report schedule (read from private fields of the pinned commit), a GOOSE retransmission
(learned from the Ethernet HAL send path), or a wake from the IPC thread. The vendor socket HAL is
compiled unmodified under renamed symbols; only the loop thread selects its wait/poll modes. The
private files are pinned by hash in CMake. Threaded MMS was rejected (10 ms report tick, 1 ms
accept polling).

*Compatibility.* No IPC, project, API or wire change; masked MMS byte streams and GOOSE frames
are identical (section 3). Other executables keep vendor socket behaviour. A libiec61850 upgrade
fails configuration until the loop is reviewed against the new private layouts.

## 5. For stream C (Go runtime)

- Worker IPC is unchanged; no new op, event or capability.
- Persistent MMS connections to simulated IEDs (the planned multiplexed `studio_control`) are
  now free: an idle connection no longer adds 10 ms to every other client's requests. Before
  this change such a connection would have slowed every SCADA client of that IED.
- The worker still accepts at most 5 MMS connections (`CONFIG_MAXIMUM_TCP_CLIENT_CONNECTIONS=5`
  at build, runtime setting 8): a persistent studio_control uses one of five slots. Raising it
  is a behaviour change left to the owner.
- Reports caused by `update_batch` now leave immediately (loop is woken) instead of within the
  next 1 ms poll; operate round trip is unchanged.
- Budgets in the capacity audit for idle CPU (<= 5 % of a core at 20 IEDs) and operate with
  5 external clients are now met on the native side.

## 6. Limitations and open points

- Windows only (the HAL wrapper wraps `socket_win32.c`); the product is Windows-only.
- Deadlines come from private fields of the pinned commit; the hash pin forces a review on
  upgrade. A deadline the loop does not model is bounded by the 50 ms safety wake.
- `studio_live` was reviewed, not changed: no 1 ms loops (50 ms inbox select, 250 ms supervisor,
  libiec61850 client 10 ms tick) and it does not raise the timer resolution.
- Protocol tests ran on a remapped copy (127.67.0.x, ports 28xxx) so the owner's dev app was
  never touched; gateway/master tests were not built (`STUDIO_GATEWAY=OFF`, unaffected code).
