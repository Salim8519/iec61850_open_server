# ADR-0089: Refused SCADA-master connects fail at once

Accepted: 2026-10-20 ([speed plan](../architecture/go-engine/speed-plan.md) phase 3, lane N3;
owner: "top notch network performance"; bug hunt [0126](../changes/2026/0126-bug-hunt-q-frontend-fixes.md)
found a 10 s wait). Evidence: change [0136](../changes/2026/0136-speed-lane-n3-native-waits.md), log
[speed-n3](../architecture/go-engine/streams/speed-n3.md).

## Problem

lib60870's Windows `Socket_connect` waits for the socket to become writable only; Windows signals a
failed non-blocking connect through the except set, so a refused connection (no IEC 104 server on
the port, firewall RST) was reported only after the full t0 (10 s). Windows also repeats the SYN
for about 2 s after an RST.

## Decision

`studio/native/master/socket_60870.c` compiles the vendor `socket_win32.c` unmodified with
`Socket_connect` renamed and provides a replacement in the lib60870 target, pinned by SHA-256 in
`cmake/master_socket_support.cmake` (same pattern as ADR-0079/0088): wait on writable + except;
first attempt without SYN repetition; a refusal is reported at once; an unanswered SYN continues on
a default socket until the connect timeout; every other failure is reported at the connect timeout
as before.

## Compatibility

Same IEC 104 bytes, error text and events; only the time of a refusal changes (10 s -> < 0.1 s).
On the TCP level an unreachable host now sees one extra SYN from a second source port after ~1 s.
The gateway (slave) never connects. Windows 8+ (`SIO_TCP_INITIAL_RTO`); without it the single
default socket is used and a refusal is reported after Windows' own ~2 s. The SV/GOOSE listener
and studio_control wait changes in the same lane change no contract and need no ADR.
