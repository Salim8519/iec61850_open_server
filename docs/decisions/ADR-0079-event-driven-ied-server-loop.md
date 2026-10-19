# ADR-0079: Event-driven simulated IED server loop

Accepted: 2026-10-20 (workstream G of [ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md);
owner asked for the code that really affects speed). Evidence and measurements:
[g-native](../architecture/go-engine/streams/g-native.md), change
[0120](../changes/2026/0120-native-stream-g-event-loop.md). Numbers 0077 and 0078 are held by the
proposed runtime ADRs in the [c-runtime](../architecture/go-engine/streams/c-runtime.md) log.

## Problem

The single-threaded libiec61850 server blocks ~10 ms per idle connected MMS client on every
request (41-58 ms per read with four idle clients) and delays operate, reports and GOOSE
retransmission by the same amount. With one active and two idle clients, GOOSE MinTime bursts
reached 23.9 ms gaps (4 frames past the 12 ms TAL, against ADR-0070). Twenty idle workers used
31-43 % of one core polling every 1 ms.

## Decision

- `studio_ied` runs the vendor single-threaded server threadless (`IedServer_startThreadless`)
  from its own loop. One thread still does all MMS work.
- The loop sleeps on the server sockets plus a loopback wake socket until the next deadline:
  the vendor's own control/report/setting-group schedule (read from private fields of the
  pinned commit, as `read_hook.c` does under ADR-0066), a GOOSE retransmission at send time +
  MinTime/MaxTime (learned from the Ethernet HAL send path), or a wake from the IPC thread
  (updates, GOOSE enable, GoEna writes). A 50 ms safety wake bounds anything not modelled.
- The vendor `socket_win32.c` is compiled unmodified under renamed symbols
  (`hal/socket_studio.c`); only the loop thread uses its wait/poll modes. Every other thread
  and executable keeps the vendor behaviour.
- `cmake/event_loop_support.cmake` pins the private headers and `socket_win32.c` by SHA-256 and
  requires the single-threaded build, so a libiec61850 upgrade stops configuration until the
  loop is reviewed.
- Threaded MMS was rejected: reports leave only from each connection thread's 10 ms tick and its
  accept thread still polls every 1 ms.

## Compatibility

- No IPC, project, API, capability or event change; start reply, `timeBeginPeriod(1)`, data-model
  locking, the 5-connection limit and the stop sequence are unchanged.
- MMS server-to-client byte streams (browse, read, operate, report GI + 3 changes) are identical
  after masking UtcTime/BinaryTime; the first 14 GOOSE frames are identical after masking `t`.
- Python and Go engines both use the new worker; no engine change is needed.
- Windows only (the product is Windows only). Enhanced SBO is ~0.1 ms slower on average (noise).
