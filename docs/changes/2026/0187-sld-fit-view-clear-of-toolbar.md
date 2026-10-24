# 0187: SLD Fit view keeps the drawing clear of the floating toolbar

Date 2026-10-24. Frontend only. Found while recording the new tutorial videos
([0188](0188-tutorial-videos-v3.md)); the owner approved fixing app bugs found on the way.

## Problem

The SLD toolbar floats over the top of the canvas (two rows while running: buttons, then the
GOOSE messages and Who is reading layers). Fit view, Tidy up, *Add all GOOSE IEDs* and *Bring it
here* fitted the drawing with one padding ratio on every side, so the top row of the drawing sat
under the toolbar. After *Add all GOOSE IEDs* the new IED components, and the GOOSE lines and
"got it" chips drawn above them, were hidden behind the toolbar on the demo station at 1600×900.
*Add* had the same blind spot: a new card went to the middle of the whole canvas and could land in
the strip under the toolbar.

## Change

- New `features/diagram/fitPadding.ts`: `toolbarSpace` measures the toolbar's lower edge in the
  canvas; `clearOfToolbar(ratio, space)` keeps the ratio on the left, right and bottom and puts
  at least the toolbar plus `TOOLBAR_GAP` (64 px: the GOOSE lane, 36 flow units above the top row,
  and its chips) at the top; `fitPadding(ratio)` measures at call time.
- `Canvas.tsx`: the toolbar height is measured in a layout effect and followed with a
  `ResizeObserver` (it wraps to more rows at small widths); the canvas fit options and the
  bottom-left Fit view button use the padding. `strayNodes.fitOptionsFor` takes any padding.
- `GooseIedsTool.tsx`, `TidyButton.tsx`, `StrayNotice.tsx`: their `fitView` calls use
  `fitPadding`.
- `useVisibleSpot.ts` (where *Add* puts a new card): centres on the canvas below the toolbar and
  counts the toolbar strip as taken, so `freeSpot` never picks a place under it.
  No other behaviour, contract or engine change.

## Evidence

- `tests/frontend/fitPadding.test.ts` (2 tests) and the Fit view stray test in
  `pushToPanel.test.ts` pass; `tsc --noEmit` clean.
- Browser, own build of master with this change (Go engine, demo station, 1600×900, running):
  before, Fit view put the IED row and the FEED1 → BUS1/BUS2 GOOSE line under the toolbar; after,
  the row starts below the toolbar and the line with "XCBR1.Pos Closed · 2 of 2 IEDs got it" and
  both "✓ got it" chips is fully visible. The same frames are in the goose-sld tutorial video.

## Limitations

- Checked at 100 % only; a window small enough for Fit to window below 100 % was not tried.
- Panning by hand can still move symbols under the toolbar, as before.
