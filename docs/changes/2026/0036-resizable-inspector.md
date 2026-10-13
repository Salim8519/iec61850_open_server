# 0036: Resizable properties panel

Date: 2026-09-20. Long IEC 61850 references pushed inspector rows past
the fixed 320 px panel, forcing sideways scrolling.

The inspector now has a drag grip on its left edge: pull to widen or
narrow between 280 and 640 px. Double-click (or Home key) resets to
the 320 px default; arrow keys step 16 px. The width is session state
only and resets on every load. Rows inside the panel now wrap and
flex children can shrink, so the default width no longer scrolls
sideways; dragging wider is comfort, not necessity.

Evidence:

- `npm run check` passes; backend unit suite: 106 pass;
  `verify.py` passes.

Limitations: visual acceptance pending (drag range, narrow-window
media queries, canvas reflow at 1366x768).
