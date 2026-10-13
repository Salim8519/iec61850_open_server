# ADR-0023: Terminal-free push button with latching default and self-reset

Accepted: 2026-09-20.

The PUSH_BUTTON canvas equipment stops being a wire-path object and
gains a third behavior, without new endpoints or native changes:

- A push button renders no terminals and cannot be wired. The project
  format bumps to 10; loading an older project strips every edge that
  touches a push button with a visible notice. Preservation of user
  data yields here because terminals no longer exist to attach to, and
  the owner confirmed no live project uses button wiring.
- Clicking a button on the canvas while running presses it and selects
  it in one gesture through the existing `/equipment/{id}/operate`
  path; while stopped a click only selects for editing.
- New buttons default to latching Boolean status. Self-resetting mode
  drives one Boolean status point to the opposite of its configured
  rest value (0 or 1) and returns it automatically after the
  configured delay through the backend Boolean pulse engine, which
  already runs while automation is paused and restores the previous
  acknowledged value.
- The contract gains `self_resetting`, `push_rest_value` and
  `push_reset_ms` (100–60000 ms); validation accepts state mappings
  for both status modes and command bindings only for momentary SPC.

Rejected alternative: a frontend-only timer that writes back after a
delay — it would race newer writes and bypass the revision-guarded
pulse restoration; and keeping button terminals — a button is an
operator control, not an electrical path.
