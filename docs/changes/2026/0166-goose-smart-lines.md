# 0166: GOOSE smart lines on the SLD

Date: 2026-10-22. Owner-approved design. Decision [ADR-0103](../../decisions/ADR-0103-goose-smart-lines.md),
spec [signal layers](../../specs/signal-layers.md) LAYER-24..28. Frontend only.

One breaker move is now one change on the SLD: a brief outline on the breaker, one chip
"Open · sent to 4 IEDs" (or "Open · 4 of 4 IEDs got it" after a real stNum increase) and one
rounded orthogonal line per receiving IED that climbs a free gap to a shared lane above the
drawing and drops onto the receiver's mark, drawing itself behind a moving dot. The raw
per-link labels are gone from the canvas; click a line for its link, the chip for the publisher.
Senders outside the project keep their dot beside the receiver.

Files: `equipment/signalRoute.ts` (routing), `signalOrigin.ts` (symbol lookup),
`signalChanges.ts` (coalescing), `signalPlan.ts` (where each change is drawn),
`signalLayerStore.ts` / `signalPulses.ts` (kept signals, cause, `equipment_state`),
`copy/signalLayers.ts` (chip words), `features/diagram/signals/RouteLine.tsx`, `ChangeLines.tsx`,
`ChangeChip.tsx`, `SignalOverlay.tsx`, `signal-layers.css`; design `layout.md`, `interactions.md`.

Evidence: new `studio/tests/frontend/gooseSmartLines.test.ts` (8 tests: origin lookup, 400 ms
coalescing, gated contact kept, chip words expected/observed, no box crossing, rounded corners,
shared lane, stagger and cap, reduced motion); `gooseVisibility.test.ts` updated for the removed
label rule. Frontend 381/381 with `node --experimental-strip-types --no-warnings --test`;
`tsc --noEmit` clean; `vite build` succeeds; `check_file_lengths.py` and `verify.py` pass.

Limitations: not checked in the running app or a browser yet (no screenshots; geometry and
wording are unit-tested only). "Got it" is the listener's view of the block's new stNum on the
network, not proof each subscriber processed it. Routes avoid equipment boxes, not wires, tags
or chips; a drawing with no free gap gets a route marked not clear. Switch words need the
backend's `equipment_state` for that node; otherwise the signal's own words are shown.
