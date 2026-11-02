# ADR-0134: Listen-check messages name what Windows answered; the lab card's DNS step may fail

Accepted: 2026-11-03. Owner: a friend's laptop showed two errors with no usable log ("why our app
failing?"), then "set a plan to fix ... without generalizing the issue to all", "make sure the app is
not behaving different than now". Change [0211](../changes/2026/0211-clear-listen-and-lab-card-failures.md).
Builds on [ADR-0128](ADR-0128-lab-service-one-call.md) and the endpoint preflight of the Go engine
([ADR-0074](ADR-0074-go-engine.md)).

## Problem

1. Creating the lab card failed with `Set-DnsClient ... HRESULT 0x80041010` (WMI: invalid class,
   `MSFT_DNSClient`). On that PC Windows has no working DNS Client settings class (disabled or
   removed DNS Client, a damaged WMI repository or a reduced Windows build). The card script stops
   at the first error, so the whole setup failed over a setting the card does not need, and the
   IPv6 step after it never ran.
2. Run said "C06_BCU cannot listen on 127.50.0.2:102 because another program is using it
   ([WinError 10013] ...)". The preflight said "another program is using it" for every bind error:
   10013 (Windows refused), 10048 (in use) and 10049 (address not usable, for example an unplugged
   card) alike, and never said which program. Neither failure was written to a log.

## Measured before deciding

The preflight test-binds with `SO_EXCLUSIVEADDRUSE`; the IED (libiec61850) listens with
`SO_REUSEADDR`. A holder on port 102 was started in each way on the owner's PC and both binds
tried on 127.50.0.2:102:

| Holder | Preflight | IED bind | A client to 127.50.0.2 reaches |
| --- | --- | --- | --- |
| 0.0.0.0, exclusive | 10013 | 10013 | nothing |
| 0.0.0.0, default or reuse | ok | ok | the IED |
| 127.50.0.2, default or exclusive | 10048 | 10013 | nothing |
| 127.50.0.2, reuse | 10048 | ok | the other program |
| 127.0.0.1, any | ok | ok | the IED |

The preflight refuses exactly where the IED could not listen or would silently lose its clients to
another program. Its decision stays as it is.

## Decision

1. **The check is unchanged; its message says what Windows answered** (`runtime/listen_problem.go`).
   The program holding the port is looked up in the Windows TCP listener table
   (`internal/portholder`, read only) and named with its process id:
   - 10013 with a holder on every address: "*program* holds port 102 on every address of this PC and
     lets no other program use it";
   - 10013 otherwise: Windows refused it; a program holds the port for itself or security software
     blocks it;
   - 10049: the address is not usable on this PC right now; check that the card is enabled and its
     cable is plugged in;
   - any other error keeps the old sentence, naming the program when one is found. With no holder
     found, 10048 reads word for word as before.
2. **The run log keeps the failure.** A refused check emits `listen_check_failed` (IED, endpoint,
   Windows error, holder process id, program and address) before Run answers.
3. **Only the lab card's DNS step may fail.** `Set-DnsClient -RegisterThisConnectionsAddress $false`
   runs inside `try/catch` (`labnet/card_script.go`); its reason is added to the install result
   message, which the network journal records, and the IPv6 step still runs. Every other step still
   stops the setup. A PC where that class is missing has no working DNS Client settings to register
   the card's addresses with.

## Compatibility

- Go-only difference from the frozen Python backend: message text of a failed listen check and the
  new run-log event; Python keeps "another program is using it". The check, its socket options and
  every successful Run are unchanged.
- On a PC where every card step works, the card is set up by the same commands in the same order.
- `netcfg.Preflight` (not used by Run) keeps its Python-identical text.
