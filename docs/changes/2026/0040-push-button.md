# 0040: Terminal-free push button, latching default, self-reset

Date: 2026-09-20. Push buttons leave the wire path: no terminals, click
to press on the canvas, latching default, plus a self-resetting mode.

Buttons render no anchors; format 10 strips button wires on load with a
notice. Clicking a button while running presses it through the existing
operate path and selects it; while stopped it only selects. New buttons
default to latching Boolean status. Self-resetting drives the mapped
Boolean to the opposite of its rest value (0/1, configurable) and
returns it after the configured delay via the backend pulse engine.

Evidence:

- `npm run check` passes; 9 frontend node tests pass; backend unit
  suite: 107 pass; protocol mapped-control suite: 4 pass, including the
  new press-then-auto-rest capture; `verify.py` passes.

Limitations: visual acceptance pending (terminal-free symbol, canvas
press feel, three-mode behavior editor).
