# ADR-0100: Typed network addresses get the planner's checks (Go engine only)

Accepted: 2026-10-22. Owner decision in chat: fix it in the Go engine only, because the Python backend
is frozen ([ADR-0074](ADR-0074-go-engine.md), [ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md)); make the answer
instant. Change [0163](../changes/2026/0163-typed-addresses-checked.md).

## Problem

In Settings › Network › "Put my IEDs on a network" a user may type an address for any row. The planner
checked only the addresses it picked; every valid typed address was forced to `ok` ("Address typed by
you."). Typing an address another IED already used, this PC's own address, or an address on another
adapter showed "OK: Free" and Apply stayed on; Run then failed with "port in use" or the adapter got
an address it already had.

## Decision

1. Typed addresses run through the same checks as planned ones, in memory (no extra ping, ARP or
   Windows call): `duplicate` (another row holds it, including a second typed row and the adapter's own
   address), `used_by_other_adapter`, `already_on_adapter`. Rows nobody typed for hold their address
   first, then typed rows in list order. A planned pick that a typed address replaced no longer leaves
   a stale `duplicate`; adapter-strategy picks step aside for typed addresses. Spaces around a typed
   address or prefix are ignored.
2. Apply stays refused while any row is blocking (unchanged `ready` rule).
3. The browser shows a duplicate or own-PC clash at once (`typedAddressCheck.ts`, same words as the
   engine) and turns Apply off until the engine's answer belongs to what is typed; the preview wait for
   typing is 300 ms instead of 500 ms.

## Compatibility

Intended difference from Python for these four cases only; Python still answers "ok". No generated
parity vector or HTTP golden file contains such a case, so none changed. Projects and saved settings are
unchanged.
