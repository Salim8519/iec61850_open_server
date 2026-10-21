# Speed lane N3: native waits in studio_control, studio_live and studio_master

Branch `speed-n3` from master 247e6d6 ([speed plan](../speed-plan.md), phase 3, N3). Owns
`studio/native/control/**`, `studio/native/live/**`, `studio/native/master/**`,
`cmake/master_socket_support.cmake` and their lines in `studio/native/CMakeLists.txt`. No IPC,
event, result-text or wire change; the Python and Go engines use the same workers unchanged.

## 1. Changes

| Where | Before | After |
|---|---|---|
| `control/control_session.cpp` (`operate`, used by IED Explorer and gateway pass-through) | After an enhanced-security operate the command polled for CommandTermination with `sleep_for(10 ms)`; with the default Windows timer each poll is a 15.6 ms tick and the first check is always too early, so every enhanced operate waited **~31 ms** | The CmdTerm handler signals a condition variable under its mutex; the command wakes at once, same `termination_ms` limit (default 5 s) and same result fields. The `Termination` object is declared before the control object, so it outlives `ControlObjectClient_destroy` (which unlinks the object under the lock the receive thread holds while calling the handler); the old atomics had a stack-lifetime race on a late termination |
| `live/sv_header.cpp`, `live/sv_traffic.cpp` (SV listener) | Two polling threads: the SV header reader looped `Ethernet_receivePacket` + `sleep_for(1 ms)`; libiec61850's SV receiver thread looped on a 1 ms Npcap read timeout (its `EthernetHandleSet` is compiled out on Windows) | The header reader opens its own Npcap capture (same device name, snap length, promiscuous flag and BPF filter as the HAL), non-blocking with MinToCopy 0, and sleeps in `WaitForMultipleObjects` on Npcap's receive event and a stop event, no timeout. The libiec61850 receiver runs threadless (`SVReceiver_startThreadless`) and is drained with `SVReceiver_tick` on every wake of the same loop; one extra look 20 ms after traffic stops. If the own capture cannot open, the receiver keeps its vendor thread exactly as before |
| `live/goose_inbox.cpp` (GOOSE listener, frames forwarded by the GOOSE engine) | `select()` with a 50 ms timeout so `close()` could stop it: ~16 wakes/s while idle | `WSAEventSelect(FD_READ)` + stop event; reset, read every waiting datagram, then wait without timeout. Same datagrams to the same handler in the same order |
| `master/socket_60870.c` + `cmake/master_socket_support.cmake` (IEC 104 master connect) | lib60870's `Socket_connect` waits only for *writable*; Windows reports a failed non-blocking connect in the *except* set, so a refused port (RST) was reported only after the whole t0 (10 s default). Windows itself also repeats the SYN for ~2 s after an RST | lib60870's `socket_win32.c` is compiled unmodified with `Socket_connect` renamed (ADR-0006 pattern, SHA-256 pinned). The replacement waits on writable + except. The first attempt asks for no SYN repetition (`SIO_TCP_INITIAL_RTO`, `TCP_INITIAL_RTO_NO_SYN_RETRANSMISSIONS`): a refusal (`WSAECONNREFUSED`) fails at once; if the SYN only times out (~1 s), a second socket with Windows' default retransmissions continues until t0; every non-refusal failure still returns at t0. A socket bound to a local address keeps one attempt. Same error text and `master_link` events as before |

| `live/goose_traffic.cpp`, `live/sv_traffic.cpp` (Stop tail, integrator request from lane I2) | The supervisors slept 250 ms (GOOSE) and 500 ms (SV) and `stop()` joined them, so `goose_stop`, `sv_stop` and `stop` waited for the rest of the tick | Each supervisor waits on a condition variable with the same 250 / 500 ms timeout; `stop()` clears `running` under its mutex and notifies, so the join returns at once. Same tick cadence and events while running; the one tick a stopping supervisor used to run after its sleep (a last `goose_stats`/`sv_rx` during Stop) no longer happens |

Reviewed and left alone (behaviour, not latency): the SV supervisor's 500 ms report cadence and the
GOOSE supervisor's 250 ms timeout/statistics cadence (they define when `sv_rx`, `goose_rx` timeout
and `goose_stats` events go out); `live/mms_session.cpp` and `live/reports.cpp` have no sleeps or
reconnect loops (reports arrive through the libiec61850 handler); `canvas_command.cpp`
(`control_mux_v1`) was already event-driven.

## 2. Evidence

Laptop Ryzen 9 5900HS, Windows 11 26300, 2026-10-20. Before = master 247e6d6 built in
`build/speed-n3-base`, after = this branch in `build/speed-n3`, both `-DSTUDIO_GATEWAY=ON`, each
tool from its own folder. Loopback 127.67.x.y, ports >= 20000, synthetic templates only; the
owner's app and `build/Release` were not touched; every process started was stopped. Scratch
scripts (not committed): `control_bench.py`, `control_wire.py`, `master_bench.py`, `sv_idle.py`,
`sv_threads.py`, `goose_idle.py`, `wakes.py` (context switches and cycles per process/thread from
`NtQuerySystemInformation`).

**Enhanced operate through `studio_control operate`** (studio_ied from the base build for both;
association kept open; 2 rounds x 100 operates alternating close/open per build, interleaved):

| ctlModel | before mean / p50 / p90 / max ms | after mean / p50 / p90 / max ms |
|---|---|---|
| direct-with-enhanced-security | 30.89 / 30.86 / 31.71 / 32.75 | **0.90 / 0.88 / 1.05 / 1.21** |
| sbo-with-enhanced-security | 30.88 / 30.84 / 31.55 / 32.45 | **1.12 / 1.08 / 1.35 / 1.69** |

The 400 result objects were identical between builds (`ok`, `selected`, `operated`,
`terminated: true`, `add_cause`). Saving: ~30 ms per enhanced operate (plan estimate 10-15 ms).

**Wire bytes of studio_control** (`control_wire.py`: recording proxy 127.67.4.1:20401 ->
studio_ied 127.67.4.2:20402, sbo-with-enhanced close + open + `close_all`, UtcTime masked): one
association each, client-to-server 891 bytes and server-to-client 1405 bytes, **WIRE IDENTICAL**.
`native_wire_compare.py BASE NEW` (studio_ied, studio_probe): 5 MMS streams identical after
masking (9570, 198, 746, 198, 758 bytes), 4 reports each, 14 GOOSE frames identical: **WIRE
IDENTICAL** (studio_ied sources are unchanged by this lane).

**studio_live SV listener** (VMnet1 host-only adapter, 10 s per phase; runs new, base, new, base;
publisher = `studio_sv_test_publisher` 4000 frames/s, 9-2LE, 1 A / 63.5 V):

| phase | before wakes/s, Mcycles/s | after wakes/s, Mcycles/s |
|---|---|---|
| process, no listener | 0, 0 | 0, 0 |
| listener on an idle adapter | 176.9-184.9, 15.7-16.3 | **4.6-5.0, 0.75-0.77** |
| listener with 4000 frames/s | 289-299, 117.6-121.9 | **113-138, 66.3-70.4** |

Per thread on an idle adapter: before two pollers at 81 and 84 wakes/s plus the supervisor
(3.5); after the capture loop **0** and the supervisor 4.3 (its 500 ms report tick). Reports
were the same in all runs: `receiving`, rate 3896-4014, lost 0, invalid 0, RMS IA-IC 1.0 A,
VA-VC 63.5 V, IN/VN 0, smpSynch 2, then `timeout` after the publisher stopped (asdus 45520-45560).

**studio_live GOOSE inbox** (`goose_start` with no subscriptions, 10 s idle): inbox thread
before **16.4** wakes/s, after **0**; supervisor 6.0 / 5.7 (250 ms tick). 50 datagrams after
the idle phase woke the process 56 / 53 times (each read at once).

**studio_live Stop** (`stop_bench.py`: GOOSE listener + SV listener on VMnet1, 10 sessions per
run with the stop moment shifted against the ticks; runs after, before, after):

| request | before mean / max ms | after mean / max ms |
|---|---|---|
| `goose_stop` | 102-157 / 205-320 | **0.7-0.9 / 0.8-1.6** |
| `sv_stop` | 327-388 / 475-498 | **0.6-0.8 / 0.7-2.3** |
| `stop` (both listeners running) | 467-471 / 677-683 | **1.1-1.2 / 1.3-1.6** |

**studio_master IEC 104 connect failure**:

| case | before | after |
|---|---|---|
| refused: closed port 127.67.0.9:28999, t0 10 s | 10080-10266 ms | **70-101 ms** (5 runs) |
| unreachable: 192.0.2.1:2404 (TEST-NET-1), t0 5 s | 5398-5415 ms | 5073 ms |

Error text identical in every run ("No answer from 127.67.0.9:28999 within 10 s. Check the
address and port, and that the device accepts this PC as its master."), and the same two
`master_link` events (`connecting`, then `closed` "The connection to ... closed"). Raw socket
probe on this laptop (`conntest.c`): vendor wait 10014 ms; writable+except only 2028 ms
(Windows' SYN repetition after RST); first attempt without SYN repetition 15 ms; an unanswered
SYN without repetition fails after ~1.0 s with `WSAETIMEDOUT`, which is why a second, default
socket continues until t0.

**Protocol suite** (`studio/tests/protocol` copied with 127.50.0.x -> 127.67.0.x and ports
12xxx/15xxx/18xxx -> 22xxx/25xxx/28xxx, copy deleted after; `STUDIO_NATIVE_BIN` = after build
with `studio_goose.exe` built from this branch; gateway, master, converter and GOOSE engine
tests included): **79 passed, 9 skipped** (GOOSE share/conflict and SV need
`STUDIO_GOOSE_ADAPTER`), before and again after the Stop change. `test_sv_listen.py` with `STUDIO_GOOSE_ADAPTER` = VMnet1: 4 passed on
after and on before. `tests/integration/test_tools_api.py` remapped (127.67.0.236:28136): 2
passed. Go: `go test ./internal/native/... ./internal/app/... ./internal/api/toolsapi/...` with
`STUDIO_NATIVE_BIN` = after build: ok (composed timing/parity tests skip without their switches).
`check_file_lengths.py`: PASS.

## 3. Proposed ADR (number to assign at merge)

**ADR-00xx: Refused SCADA-master connects fail at once**

*Problem.* lib60870's Windows `Socket_connect` waits for the socket to become writable only;
Windows signals a failed non-blocking connect through the except set, so a refused connection
(no IEC 104 server on the port, firewall RST) was reported only after the full t0 (10 s). Windows
also repeats the SYN for about 2 s after an RST.
*Decision.* `studio/native/master/socket_60870.c` compiles the vendor `socket_win32.c` unmodified
with `Socket_connect` renamed and provides a replacement in the lib60870 target (pinned by
SHA-256 in `cmake/master_socket_support.cmake`): wait on writable + except; first attempt without
SYN repetition; refusal reported at once; an unanswered SYN continues on a default socket until
the connect timeout; every other failure is reported at the connect timeout as before.
*Compatibility.* Same IEC 104 bytes, error text and events; only the time of a refusal changes
(10 s -> < 0.1 s). On the TCP level an unreachable host now sees one extra SYN from a second
source port after ~1 s. The gateway (slave) never connects; test masters linking lib60870 get the
same connect. Windows 8+ (`SIO_TCP_INITIAL_RTO`); without it the single default socket is used and
a refusal is reported after Windows' own ~2 s.

The SV and GOOSE listener changes need no ADR beyond this log (no contract change; same frames
reach the same libiec61850 code, threadless mode is a vendor API).

## 4. Limitations and open points

- **libiec61850 MMS client receive thread**: each open association in studio_control and
  studio_live keeps ~64 wakes/s (the vendor `connectionHandlingThread` sleeps 10 ms between
  polls, 15.6 ms on the default timer; measured 65 wakes/s, 5.7 Mcycles/s with one idle
  association, same before and after). It does not show in operate latency (0.9-1.1 ms per
  enhanced operate), so it is a CPU/idle-wake item only. Removing it needs a vendor/HAL change
  (event-driven client thread or a high-resolution `Thread_sleep`), i.e. lane N1's `hal/**` and
  an ADR; not done here.
- The master text "No answer ... within 10 s" is kept for a refusal, as required; a clearer
  "refused" text would be a separate UI-text change.
- SV: the libiec61850 receiver now runs on the header capture's thread; a stop waits for at most
  one Npcap read timeout. Both captures read the same adapter as before (no extra handle).
- Real IEDs and a real 104 outstation were not available; refused/unreachable cases were measured
  on loopback and a TEST-NET address, SV on the host-only VMnet1 adapter with the test publisher.
- Integrator: link this log from `streams/README.md`, write the change record and ledger entry,
  and rebuild studio_control, studio_live and studio_master into the app folder only with the
  owner's app closed and the owner's OK. No EXE was built.

## How to reproduce

```powershell
cmake -S studio/native -B studio/native/build/speed-n3 -G "Visual Studio 16 2019" -A x64 `
  -DIEC_SOURCE=<repo>/.local/native/libiec61850 -DJSON_SOURCE=<repo>/.local/native/json `
  -DNPCAP_SDK=<repo>/.local/native/npcap-sdk -DLIB60870_SOURCE=<repo>/.local/native/lib60870/lib60870-C `
  -DOPENDNP3_SOURCE=<repo>/.local/native/opendnp3 -DOPENDNP3_DEPS=<repo>/.local/native/opendnp3-deps -DSTUDIO_GATEWAY=ON
cmake --build studio/native/build/speed-n3 --config Release --parallel 4 --target studio_control studio_live `
  studio_master studio_ied studio_probe studio_gateway studio_sv_test_publisher studio_test_iec104_master
python studio/tools/native_wire_compare.py <base>/Release <new>/Release
```
