# ADR-0103: GOOSE smart lines on the SLD

Accepted: 2026-10-22. Owner-approved design ("smart lines") in chat. Change
[0166](../changes/2026/0166-goose-smart-lines.md). Builds on [ADR-0065](ADR-0065-sld-signal-layers.md),
changes 0097, 0102 and 0105. Spec: [signal layers](../specs/signal-layers.md) LAYER-24..28.
Frontend only; the Go engine, Python backend, native workers and the GOOSE engine are unchanged.

## Problem

The GOOSE layer drew one curve per link, from the publisher's corner mark to the subscriber's,
with a raw label such as "PUB → SUB · GGIO1.Ind1 off" for every pulse. One breaker move (two
contacts, several receivers) gave many overlapping long curves and unreadable text, and the curve
started at the IED's corner mark, which often sits on another symbol, so it looked as if the wrong
equipment had sent it.

## Decision

1. **One change per real change.** The store keeps every publisher signal that belongs to a sent
   dataset (even when the one-pulse-per-link-per-second gate silences it) and tags each pulse with
   its cause. Signals from one publisher that land on the same symbol within 400 ms are one change
   (`equipment/signalChanges.ts`). Pulses whose cause is unknown (only a new stNum) group per
   publisher in the same window. One line per receiving IED, never one per link and contact.
2. **Origin = the symbol.** The equipment whose own bindings (state mapping direct, single or both
   contacts; command; relay rows; measurement rows) name the changed reference, state first
   (`equipment/signalOrigin.ts`). It gets a brief outline. Without such a symbol the change starts
   at the publisher's existing mark or name tag.
3. **Destination = the receiver's existing mark** (corner "G" or name tag).
4. **Routing** (`equipment/signalRoute.ts`, pure): orthogonal, corners rounded (r 8). Leave the
   origin sideways into the nearest vertical gap free of equipment boxes, climb to one lane 36 px
   above the topmost box, run along it, drop down the receiver's free gap. Candidate gaps are just
   outside box edges, nearest first; a route that finds no free gap is still drawn and flagged
   `clear: false` (tests require clear routes for normal layouts). All receivers of a change share
   the climb and the lane.
5. **Animation.** Each line reveals itself behind a moving dot: per animation frame the dot's
   transform and a mask path's `stroke-dashoffset` are set (paint only, no layout). Lines of a
   change start 80 ms apart (at most 400 ms) and fade by the existing 2 s (`LINK_FADE_MS`).
   At most 16 lines move at once; the rest show at once. Reduced motion: no travel, the line
   shows whole with the dot at the receiver, then disappears. CSS keyframes stay opacity-only.
6. **Words.** One chip per change above its origin: "Open · sent to 4 IEDs" (dashed border, "→")
   or "Open · 4 of 4 IEDs got it" (solid border, "✓"), full words and receiver names on hover.
   Switch words come from the backend's own `equipment_state` (Dbpos) for that node, so the
   browser does not re-derive a position from contacts; otherwise the relay/measurement row and
   value, or the signal's own words. The receiving mark says "✓ got it" only after a real stNum
   increase and nothing for an expected change. Per-link canvas labels are removed. A click on a
   line opens that link in the side sheet; a click on the chip opens the publisher's links.
7. **Unchanged:** a sender outside the project keeps its one dot beside the receiver and the
   receiver's "← sender · change" line (change 0102, LAYER-20/22); problem marks, tags, readers.

## Compatibility

View-only change. No project, API, event or preference format changes; nothing is saved. The store
gains view-only fields (`signals`, `equipment`, `Pulse.cause`, `Pulse.change`) and now also reads
`equipment_state` events it already received. The "line" layer words "PUB → SUB · change" remain
as the line's hover text for outside senders only. Routes are computed from node positions already
held by the overlay and memoised, so GOOSE traffic still never re-renders equipment nodes.

## Limitations

"Got it" means the GOOSE listener saw the new stNum on the network for that block (LAYER-07
evidence), not that each subscriber's application processed it. Routing avoids equipment boxes,
not wires, tags or chips; extremely dense drawings may produce a route flagged not clear.
