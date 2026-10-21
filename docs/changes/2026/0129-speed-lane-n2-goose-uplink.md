# 0129: Speed lane N2 merged: event-driven GOOSE uplink, no 1 ms timer, idle CPU 0

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 3, lane N2. Branch
`speed-n2` (7525999, 793b2c1) merged into `master` (2030e69). Stream log:
[speed-n2](../../architecture/go-engine/streams/speed-n2.md). Applies to both engines once
`studio_goose.exe` is rebuilt in the app folder (not done: the owner's app uses it).

## Change

- `studio/goose_engine/internal/uplink`: non-blocking Npcap handle with its receive event
  (`pcap_setnonblock`, `pcap_getevent`); the adapter thread (locked OS thread) waits on stop, send
  and receive events with no timeout, drains in batches of 256; `Send` wakes it at once.
- `cmd/studio_goose/timer_windows.go` and `timer_other.go` removed: the engine no longer raises
  the machine-wide 1 ms timer (its timers are 100 ms-5 s; GOOSE retransmission timing is in the IED
  workers). The 100 ms share check runs only while sharing waits for the listen window.
- Bug fix: `npcap.Open` read the filter error text after `pcap_close`; any non-Ethernet adapter
  (e.g. Npcap loopback) crashed the engine. Now reported as uplink `error`, engine keeps running.
- IPC protocol and capabilities unchanged (`goose_engine_v1`); frames byte-identical and in order.

## Evidence

- Whole process 10 s idle: no adapter 103-116 -> 0 context switches/s; adapter open 10.5 % of a
  core and 5679 switches/s -> 0 CPU, 0 switches/s (VMnet1, receive-only).
- Publish to `pcap_sendpacket` hand-off, 2000 frames: before (with 1 ms timer) p50 544 us, p99
  2.2 ms, 768 idle wakes/s; after p50 19 us, p99 248 us, 0 idle wakes.
- Fake-handle tests (burst order, 3000 frames unchanged, allow gate, prompt Close, lost adapter,
  send failure once), share resume tests; `go vet` (Windows, Linux), gofmt, file lengths clean.
- Integrator on `master`: only `studio/goose_engine` and the stream log; inventory note updated;
  `verify.py` (runs the engine's go test): see ledger line.

## Limitations

- Receive wake not yet observed with real GOOSE frames through the driver (no traffic in the run);
  owner rig test with a VM publisher pending. Adapters that can never open are still retried every
  2 s (backoff would change state-event timing: owner decision). p99 248 us above the 0.1 ms goal.
