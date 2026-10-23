# ADR-0112: GOOSE IED components on the SLD

Accepted: 2026-10-23. Owner-approved design in chat (decisions 1-7 below). Change
[0183](../changes/2026/0183-goose-ied-components.md). Spec: [signal layers](../specs/signal-layers.md)
LAYER-29..38. Builds on [ADR-0065](ADR-0065-sld-signal-layers.md), [ADR-0073](ADR-0073-goose-engine-go.md)
(GOOSE inside the app), [ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md) (Go only),
[ADR-0103](ADR-0103-goose-smart-lines.md) (smart lines) and [ADR-0111](ADR-0111-fit-to-window.md).
**Supersedes** ADR-0103 decision 3 (destination = the receiver's "G" mark or name tag), the corner
marks of change 0105 and the left name-tag column, and ADR-0065's "tags are never saved" for GOOSE:
IEDs are now saved diagram components.

## Problem

GOOSE lines ended on small "G" marks inside the corner of whatever equipment happened to be bound to
an IED, or on name tags in a column left of the drawing. A breaker could look like it received a
trip, an IED bound to nothing floated outside the drawing, and nothing on the SLD could send GOOSE:
an engineer had to find the IED's data point and change it by hand, guessing which values were in
the message.

## Decision

1. **IED component.** New Go-only diagram kind `IED`: one whole IED, at most one per IED per drawing
   (a second is refused in plain words), no terminals, fixed 200 px card. It shows the full IED
   name (wrapped, never cut), *Simulated · running / stopped* or *Real device · watched*, its GOOSE
   in words ("Sends 2 messages · gets 1", "No GOOSE"), and, while the layers are on, its GOOSE
   problems and "◉ n read". It reuses the omittable `ied_id`; it has no state or command binding.
   Removing an IED from the project removes its component.
2. **Lines.** One change runs from its **origin** through the **sender's IED component** to **every
   receiving IED component** (ADR-0103 routing reused, `routeVia`). No origin: it starts at the
   sender's component. No sender component: from the origin straight to the receivers. A line
   never ends on equipment. A sender with no place at all (outside the project, or no component
   and no trigger) flies one dot into each receiving component (changes 0097/0102 unchanged).
3. **Origin = a user action, never a guess.** Only a component the user acted on within 1.5 s
   counts: a GOOSE send (equipment link or IED component, exact refs), a point panel row (exact
   ref), or an operate / test / button / tap command on equipment whose own bindings name the
   changed value. A change from a generator, script, real IED or outside client starts at its
   sender's component (supersedes ADR-0103 decision 2's "binding names it" rule).
4. **Missing receivers** are named, not drawn: a dashed chip "also TR5 · not on drawing" under the
   sender, and a toolbar list "N IEDs not on the drawing ▾" with *Add* per IED (pattern of the
   outside senders list).
5. **Add all GOOSE IEDs.** One click in Add › *IEDs and GOOSE*: Go proposes one component for every
   project IED that sends or receives GOOSE and has none yet, in one evenly spaced row 60 px above the
   drawing (spread over its width), or continuing an existing row of components; the browser inserts
   them in one diagram edit (one Undo step). Tidy up leaves IED components where they are.
6. **Sending GOOSE.** (a) From the IED component's sheet: every message, its values in dataset order
   with functional constraint, and On/Off, Open/Closed or a number box for each value the drawing may
   change; real IEDs are watch only. (b) From equipment: an optional *GOOSE* section (IED → message →
   value; a breaker, disconnector or earth switch sends a position (Dbpos) or one BOOLEAN, or open
   and closed as two BOOLEANs) saved as the omittable `goose_link`; *Send Open/Closed*, *On/Off* or
   a number while running. (c) Point panel rows whose value is in a message show a "GOOSE" tag and
   are origins. Every send is ONE sample batch on the simulated IED's actor: libiec61850 publishes
   pending GOOSE when the data model lock of the batch is released, so stNum rises exactly once.
7. **Animation and colours** of ADR-0103 stay (dot, reveal, stagger, 16-line cap, chip words,
   reduced motion, GOOSE colour `--accent`).

## Contracts (Go engine only; the frozen Python engine refuses the kind and the routes)

- `kind: "IED"` accepted by the Go node kinds before the Python list, so the unknown-kind words and
  the Python golden vectors are unchanged; `goose_link` omittable on every kind but `JUNCTION`,
  `INFEED`, `POINT_PANEL` and `IED`: `{ied_id, gocb_ref, mode: one|contacts, one, open, closed}`,
  checked in plain words against the IED's stored messages (`goosemsg.Link.Check`).
- Membership is functional-constraint aware and comes from the stored `fcda_refs` and member FC
  (`XCBR1.Pos [ST]` holds `Pos.stVal/q/t`, never `Pos.ctlModel [CF]`); one catalog per stored IED
  record, cached (≈30 ns to reuse, ≈13 µs to build).
- Routes: `GET /ieds/{id}/goose/messages`, `GET /goose/members`, `GET /diagram/goose-ieds`,
  `POST /equipment/{node}/goose` (see [API](../reference/API.md)). A send logs
  `goose_send_requested` and `goose_send_applied` or `goose_send_failed` with node, IED, block,
  refs, values and one correlation id; the run log adds the run id. Refusals (real device, skipped,
  GOOSE switched off, not running, no link, wrong value) write nothing and log nothing.

## Compatibility

Projects without IED components or links are byte-identical (both fields omittable; goldens pass).
A project saved with IED components opens only in the Go engine (as for `TAP_TRANSFORMER`).
Without Go's `/goose/members` (older engine) expected pulses keep the dataset prefix rule. The
frontend learns sizes from React Flow's measured nodes (the overlay used an 80×60 fallback before,
which mattered little for symbols but routed a 200 px card through its neighbours).

## Limitations

"Got it" is still the app's own GOOSE listener seeing the new stNum (LAYER-07), not each receiver's
application. Messages with structured members (whole data objects) list every leaf; only BOOLEAN,
Dbpos and numbers are sendable. The GOOSE colour stays the shared accent; no new colour token.
