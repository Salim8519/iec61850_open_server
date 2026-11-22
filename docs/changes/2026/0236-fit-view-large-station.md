# 0236 Fit view shows the whole of a large station; changelog lists 0.1.35 and 0.1.36

Date: 2026-11-22. Owner: "the button of sld when user clicks adjust zoom (the one under plus and minus)
the sld centers, not zoom out or in to show all equipments in the sld, why, test, if test ok, then fix
on same version 0136"; "in the website we skipped version 0135 by mistake". Layers: frontend
(`features/diagram`), website changelog. No contract, engine or design change.

## Cause

The SLD canvas had `minZoom={0.2}`. React Flow 12.11.6 `getViewportForBounds` clamps the fitted zoom to
minZoom and then centres the bounds, so a drawing that needs less than 0.2 is only centred. The merged
Al Kamil 400 + 132 kV SLD (7024 x 5880) needs about 0.1 on a 1665 x 822 canvas.

The changelog's newest heading used `{SITE.version}`; setting the site to 0.1.36 relabelled the 0.1.35
notes as 0.1.36, so 0.1.35 disappeared.

## What changed

- `features/diagram/zoomFloor.ts` (new): `zoomFloor(fitted)` is 0.2, or 0.9 x the zoom that fits the
  drawing when that is smaller. `Canvas.tsx` computes it from the fitted nodes' bounds (the same nodes
  and padding as Fit view) and the canvas size, and passes it as `minZoom`.
- `website/src/pages/changelog.astro`: the 0.1.35 heading is fixed text; a 0.1.36 section lists
  changes 0231 to 0236.

## Evidence

- `tests/frontend/zoomFloor.test.ts` with React Flow's own `getViewportForBounds`: the old floor
  clamps the Al Kamil SLD to 0.2 and it does not fit (bug reproduced); with the new floor it fits.
  All 514 frontend tests pass; `tsc --noEmit` clean.
- Browser, fresh engine serving the built frontend, merged Al Kamil project: zoomed in to 0.171, the
  fit button went to 0.097 and all 371 symbols were inside the canvas; the window title showed
  "Al Kamil 400-132kV - ScadaBench".

## Limitations

- At very low zoom the symbols are small; the zoom-tier styles (`useZoomTier`) are unchanged.
