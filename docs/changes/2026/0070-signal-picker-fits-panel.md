# 0070: Signal picker fits the properties panel

Date: 2026-10-17. Owner report: in the equipment properties panel ("Position comes
from"), users could not read the full point paths, and the right side of the picker
was cut off.

## Cause

`SignalSearchPicker` put the search box (320px) and a fixed 140px report filter on one
row. In the 380px sheet (340px usable) that row was wider than the panel. Its grid
track grew to 470px, so the Connection and IED boxes, the report filter, the end of
each path and the **Choose** button all went past the clipped right edge. Paths were
also forced onto one line with an ellipsis.

## Change

- `components/SignalSearchPicker.tsx`:
  - the search box and the report filter are now stacked full-width lines;
  - the root is a `signal-search-picker` single-column grid.
- `components/signal-search-picker.css`:
  - the picker and the list use `minmax(0, 1fr)` columns, so they never grow past
    their panel;
  - point paths and descriptions wrap (`overflow-wrap: anywhere`) instead of being cut;
  - the Choose button and the "Added ✓" badge do not shrink;
  - the pagination row may wrap.
- Paths keep the short form without the IED-name prefix (the IED is chosen above),
  as the owner confirmed. The shared picker also serves the relay signal adder and
  Test points, so they get the same fix.

No design token, contract or dependency changes, so no ADR is needed.

## Evidence

- **Checks:** `verify.py` passes; `tsc --noEmit` is clean; frontend tests: 95 passing.
- **Layout check:** a page with the real stylesheets (`tokens`, `base`, `workspace`,
  picker) and the sheet → section → stack → picker markup, measured in Chrome:
  - at 380px and 330px sheet widths, nothing passes the panel edge, horizontal
    overflow is 0 and every Choose button is visible;
  - long paths wrap to 2–3 lines.
- **Old layout on the same page:** 111px overflow, with the Connection box 470px wide
  in a 339px space. This reproduces the reported screenshot.

## Limitations

- The open session was the packaged `SubstationStudio.exe` on port 8765, so dev mode
  could not start and the EXE was not rebuilt. The packaged app will show the fix only
  after the next requested build.
- Owner visual acceptance in the real app is pending.
