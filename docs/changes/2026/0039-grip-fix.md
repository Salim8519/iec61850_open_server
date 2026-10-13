# 0039: Resize grip pinned outside the scroll flow

Date: 2026-09-20. The inspector resize grip lived inside the scrolling
panel, so it scrolled away with long content and users had to scroll
back to the top to grab it.

The Inspector root is now a non-scrolling flex wrapper carrying the
fixed width; the grip is a full-height flex sibling that is always
visible, and only the inner panel scrolls. Drag math, clamp range,
reset-on-load and keyboard support are unchanged. The grip edge line
is now faintly visible by default with touch dragging enabled.

Evidence:

- `npm run check` passes; 9 frontend node tests pass; backend unit
  suite: 106 pass; `verify.py` passes.

Limitations: visual acceptance pending (grip reachability at the
bottom of a long scrolled panel).
