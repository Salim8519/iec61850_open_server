# Speed lane N2: GOOSE uplink without polling

Branch `speed-n2` from master a41879c ([speed plan](../speed-plan.md), phase 3, N2). Owns
`studio/goose_engine/**` only. No user-visible change: same IPC operations, replies, events,
capabilities (`goose_engine_v1`, so Python's `REQUIRED_CAPABILITIES` still passes), and the
same frames in the same order, byte for byte (the engine forwards the worker's slices; it
never builds or re-times a frame).

## What changed

| Before | After |
|---|---|
| Adapter thread loops on `pcap_next_ex` with a 1 ms read timeout, then drains the send queue | Handle is non-blocking (`pcap_setnonblock`); the thread, locked to its OS thread, sleeps in `WaitForMultipleObjects` on stop, send and Npcap's receive event (`pcap_getevent`), with no timeout |
| A published frame waits for the next poll | `Send` queues the frame and signals the send event at once (one `SetEvent` per burst: an atomic "pending" flag skips it while a wake is already on its way) |
| One received frame per pass | Every wake drains all waiting frames and all queued sends, in batches of 256 so neither direction starves the other; a full batch loops again without waiting |
| `timeBeginPeriod(1)` for the whole process lifetime (`cmd/studio_goose/timer_windows.go`) | Removed: nothing in the engine needs it (below) |
| Share check ticks every 100 ms for the life of the engine | Ticks only while sharing waits for the listen window on an open adapter; SetUplink and every uplink state change wake it. Same 100 ms granularity, same 3 s window |

Files: `internal/uplink/uplink.go` (lifecycle), `pump.go` (wait loop), `signals_windows.go`
(events), `signals_other.go` (non-Windows stub), `internal/npcap/npcap_windows.go`
(non-blocking + `Event()`), `internal/engine/share.go` (tick), `cmd/studio_goose/main.go`.

Why the wait cannot miss a frame: Npcap's `NPF_Read` clears the receive event **before** it
checks and copies the buffer, and the tap path sets it on every enqueue when buffered bytes
>= MinToCopy (0 here) and also when the buffer is full (checked in nmap/npcap
`packetWin7/npf/npf/Read.c`). The pump drains with `pcap_next_ex` until it returns 0, so the
last read cleared the event before finding the buffer empty; any later frame sets it again.
Sends: the pump clears "pending" before draining the queue, so a frame queued after the drain
always signals the event.

## Bug found and fixed

`npcap.Open` read the filter error text after `pcap_close` (through a null handle) and
**crashed the engine** whenever the GOOSE filter could not be compiled, i.e. on any adapter
that is not Ethernet. Seen with master's binary on the Npcap loopback adapter:
`panic: invalid memory address ... npcap.(*Handle).lastError`. Now the text is read first;
the same adapter reports uplink `error` "GOOSE filter failed: no VLAN support for BSD
loopback" and the engine keeps running. Test: `TestErrorTextOfAClosedHandleDoesNotTouchNpcap`.

## Evidence (Ryzen 9 5900HS laptop, Windows 11, Go 1.25.6, GOMAXPROCS=4)

Whole process, `studio_goose.exe` master (before) vs `speed-n2` (after), 10 s idle after
`hello`/`open`, CPU time from `GetProcessTimes`, context switches summed over the process's
threads (`Win32_PerfRawData_PerfProc_Thread`). The adapter run is **receive-only**
(`share=false`, nothing sent) on the VMware host-only adapter VMnet1, no GOOSE traffic.

| Scenario | Before CPU (1 core) | Before ctx switches/s | After CPU | After ctx switches/s |
|---|---|---|---|---|
| No adapter (Inside the app) | 0-0.16 % | 103-116 | 0.000 % | 0.0 |
| Adapter open, idle (VMnet1) | **10.5 %** | **5679** | 0.000 % | 0.0 |
| Non-Ethernet adapter (Npcap loopback) | crash | - | 0.000 %, uplink `error` | 78 (unchanged 2 s reopen loop) |

Adapter thread in-process (`TestMeasureBeforeAfter`, `STUDIO_GOOSE_MEASURE=1`): a fake
device with the Npcap event semantics; "before" is the old loop reproduced in the test
(1 ms `WaitForSingleObject` on the receive event, exactly Packet.dll's wait inside
`pcap_next_ex`). Latency is `Send` call to hand-off to the device (where `pcap_sendpacket`
runs), QueryPerformanceCounter, 2000 frames 2 ms apart.

| Mode | Idle wakes/s (10 s) | p50 | p99 | max |
|---|---|---|---|---|
| Before: 1 ms poll + `timeBeginPeriod(1)` (shipped) | 768 | 544 us | 2.21 ms | 3.91 ms |
| Before: 1 ms poll, no timer request | 64 | 8.44 ms | 15.9 ms | 19.8 ms |
| After: event wait, no timer request | **0.0** | **19 us** | 248 us | 3.8 ms (one outlier) |

Real Npcap, new code (`TestRealAdapterIdleWakes`, `STUDIO_GOOSE_ADAPTER=` VMnet1 GUID,
receive-only): 0.0 wakes/s over 10 s, states `[open]`; `Close` returned at once, which also
shows `pcap_next_ex` really is non-blocking on the driver.

Go timers without the 1 ms request (`TestMeasureGoTimers`): `time.Sleep(1ms)` p50 1.10 ms,
p99 1.72 ms without it vs 1.09 / 1.71 ms with it (max 3.6 vs 1.7 ms). Go 1.23+ sleeps on
high-resolution waitable timers. The engine's own timers are 100 ms (share check), 2 s
(reopen), 1 s rate windows and 5 s storm notices, so nothing there needs 1 ms. ADR-0070's TAL
rules concern retransmission, which the native IED workers own (MinTime/MaxTime, their own
`timeBeginPeriod`, lane N1); the engine only forwards frames and does not change their timing
except by forwarding them sooner.

Tests: `go test -p 2 -trimpath ./...` passes (uplink: order of 1291 frames heard in bursts
up to 3x batch, 3000 frames sent unchanged and in order through a full queue, nothing sent
before AllowSend or after it is switched off, 0 wakes over 1 s idle, Close of an idle
thread < 200 ms, lost adapter reported, send failure reported once then recovery, median
hand-off < 300 us; engine: sharing resumes after a lost adapter returns and waits for an
adapter that opens late). `go vet` passes for windows and linux; gofmt clean; every file
< 300 lines. Built with `go build -trimpath -ldflags "-s -w"` to
`studio/native/build/speed-n2/studio_goose.exe` (master's build for comparison in
`speed-n2/before/`); `studio/native/build/Release` untouched.

## Risks

- The real-driver receive wake with actual GOOSE frames is not exercised here (no GOOSE on the
  lab adapters, and this lane sends nothing). The wake rule is taken from Npcap's source and
  the fake mirrors it; the receive-only open, wait and close on a real adapter are verified.
- `pcap_setnonblock`/`pcap_getevent` are now required exports; any Npcap has them, a missing
  one is reported as "Npcap is too old" like the other functions.
- The 100 ms share check now starts its phase when sharing becomes pending instead of at engine
  start; sharing still opens 3.0-3.1 s after the adapter opened.

## Open items

- Owner/integrator: GOOSE host <-> VM retest (as 0103) with this binary, sharing on, to see
  received and sent frames through the real driver; then `STUDIO_GOOSE_ADAPTER={guid}
  go test -run TestRealAdapterIdleWakes -v ./internal/uplink` while the VM publishes.
- A failing adapter (missing, not Ethernet) is reopened every 2 s forever (about 78 context
  switches/s, inside `pcap_open`). Unchanged behaviour; a backoff would change state-event
  timing, so it is left for an owner decision.
- `docs/architecture/go-engine/inventory/processes-1.md` line 59 still lists the Go
  `timeBeginPeriod(1)`; the integrator should update it with the merge (outside this lane).
- Changes record and ledger entry are the integrator's (speed plan rules).
