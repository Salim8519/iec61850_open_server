# ADR-0066: Subscribers use received GOOSE; workers report MMS client access

Accepted: 2026-10-18. Specs: [GOOSE](../specs/goose.md), [signal layers](../specs/signal-layers.md).
Change: [0094](../changes/2026/0094-goose-mms-signal-layers.md). Amends GOOSE-08 of ADR-0030.

**Problem**

Change 0091 showed GOOSE travelling between two PCs and IED to IED, but a simulated subscriber
ignored what it received (only LGOS supervision was written). A simulated bay unit therefore
never reacted to its partners. Separately, the MMS server worker reported a connection without
the client's address and had no way to tell who reads what, so "who is reading this IED" could
not be answered.

**Decision: GOOSE inputs**

- A simulated subscriber applies received GOOSE only to the data its own `ExtRef` binds. The
  binding is read from SCL at import (dataset position of the input, the input's logical
  device) and checked at Run:
  - `intAddr` must name exactly one runtime-editable attribute of the subscriber: the full
    reference, the reference inside the input's logical device (`GGIO3.Ind001.stVal`), or a
    unique `/LN.DO.DA` match. Vendor-specific or free-text `intAddr` values (`Op1`, packed
    `LD|IED|…` hints) are listed as "shown but not applied", never guessed.
  - values are converted by the target's SCD type (Boolean, Dbpos from the bit string, integers);
    a structured member is applied only when the publisher's own file gives its layout.
- Frames drive inputs only when the listener says `receiving` with no problem. A confRev,
  dataset or goID mismatch, an unreadable frame or a dataset size different from the SCD is
  refused with a `goose_input_refused` event and the subscriber is not changed. A timeout keeps
  the last applied values. GOOSE seen on the wire but not declared is never routed.
- The path is the existing one: `studio_live` receives (all GOOSE crosses the real adapter in
  lab mode, also IED to IED on one PC), `acquisition/goose_inputs.py` turns an accepted frame
  into ordinary acknowledged samples, and the subscriber's native worker applies them and sends
  its own reports. The worker stays the owner of its MMS model; the coordinator stays the owner
  of simulation sources. Each application emits one `goose_input` event per stNum change.
- Projects imported before this change are bound from their stored SCL at Run.

**Decision: MMS access events**

- `studio_ied` now reports the client's `ip:port` on `connection`, `report_control` and control
  events (`ClientConnection_getPeerAddress`).
- Reads are observed by wrapping the read-access check libiec61850 already installs on its MMS
  server: the original check runs first and its answer is returned unchanged; only successful
  reads are counted. `IedServer_setReadAccessHandler` was rejected because installing it changes
  the error a client gets for a nonexistent object (protocol test compares both builds).
  This uses the private `MmsServer` layout of the pinned libiec61850 commit and is confined to
  `native/worker/server/read_hook.c`; a library upgrade must re-check it.
- Counts are aggregated per client per logical node, plus control writes and enabled report
  blocks, and sent as one `mms_access` summary at most once per second, only when something
  changed. Never one event per read. The backend keeps the client list per IED, publishes it to
  the live view only (not the run log), and removes a client when it disconnects.
- ScadaBench's own command client (`studio_probe`) is recognised for 20 s after ScadaBench
  sends a command and is shown as ScadaBench, not as an external reader or operator. Control
  events carry `external: true` only for other clients.
- No credentials or models are logged; the summary carries addresses, node names and counts.

**Compatibility**

- Workers that cannot observe reads still start; they lack the `mms_access` capability and the
  view simply shows no readers. No project format change.
- Data-attribute writes (SP, CF, DC…) are not counted yet: adding per-attribute write handlers
  in libiec61850 would change its write-access semantics.
