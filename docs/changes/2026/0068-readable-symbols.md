# 0068: Switching symbols readable from a distance

Date: 2026-10-15. Owner request: the CB/DS/ES state is hard to see unless you zoom in.
Decision: [ADR-0045](../../decisions/ADR-0045-readable-switching-symbols.md).

## Change

- `equipment/symbols/SwitchingGlyphs.tsx` (new) draws the switching symbols:
  - the CB is filled with a whole conductor when closed, and hollow with a broken
    conductor when open;
  - the DS/ES blades are long, swing 40° and have an IEC contact bar;
  - the Unknown/Invalid cross is larger.
- `equipment/symbols/EquipmentSymbol.tsx`: uses the glyphs and takes its stroke from `--sym-stroke`.
- `equipment/symbols/switch-motion.css`: pivots are set inline, and opacity has a transition.
- `equipment/readability.ts` (new): zoom bands and confirmed-change detection.
- `features/diagram/useZoomTier.ts` (new) and `Canvas.tsx`: a `zoom-near|mid|far` class.
- `equipment/EquipmentNode.tsx` and `equipment-node.css`: a change pulse, and thicker strokes per zoom band.
- `studio/tests/frontend/symbolReadability.test.ts` (new).

## Evidence

**Tests:** frontend 90 passing, including 3 new readability tests. `tsc --noEmit` is clean.

**Browser** (dev session, saved project, stopped):
- The real components were rendered for Closed/Open/Intermediate/Invalid at 1.0, 0.6 and 0.35 scale.
- At 0.35, open and closed are clear from geometry alone: a hollow box with a broken
  conductor versus a filled one, and a swung blade versus an upright one.
- An earlier draft had a white "open" blade that almost disappeared. Its outline was
  thickened, and the browser check confirmed it is visible.
- The canvas switched `zoom-far` → `zoom-near` while zooming. The pulse animation
  attached to a card. The console showed no errors.

## Limitations

- The pulse was checked by setting the attribute in the browser, not by a live run (the
  saved project was not run while the owner was away). The trigger logic is unit-tested.
- Wire widths do not change with zoom (energization line styles are unchanged).
- Owner visual acceptance is pending.
