# ADR-0026: Configurable state colours for two-state equipment

Accepted: 2026-09-29. Owner request: a closed breaker must be obvious without
zooming — green when closed, white when open by default, changeable by the
user, and the same idea for other objects.

- Contract: `EquipmentData` gains optional `active_color` and `inactive_color`
  (`#rrggbb`, validated). `None` means the kind default. They are display-only:
  simulation, native workers and SCL export never read them. No project format
  bump or migration: older projects load unchanged.
- Kinds and defaults: CB/DS/ES Closed `#16a34a` / Open `#ffffff`; TRIP_RELAY
  Operated `#dc2626` / Reset `#ffffff` (respecting state polarity); status-mode
  PUSH_BUTTON Pressed or Latched `#16a34a` / Released `#ffffff`. Momentary SPC
  buttons, CT, VT, sources, relays and junctions have no two-state fill.
- The canvas card fills only for a confirmed state from the live value.
  Intermediate uses a dashed amber card, Invalid a dashed red card, and
  unmapped or awaiting objects stay neutral. Text/symbol colour is chosen for
  the higher WCAG contrast (navy or white). The state pill text and symbol
  geometry remain authoritative, so colour is never the only cue.
- Properties "Appearance" card: native colour pickers with live preview that
  commit once per pick, Reset to defaults, and Apply to all of the same kind.
  Editing follows the normal diagram rules (stopped only, undoable, saved).

Compatibility: older Studio builds reject diagrams that contain the new fields
(`extra="forbid"`); projects saved without custom colours are unaffected.

Amendment (2026-09-29, same day, owner feedback "the whole object got
coloured"): the card is no longer filled. The state colour paints only the
part that shows state — breaker body (with contrasting contact), disconnector
and earth-switch blade as an outlined bar, trip-relay body, push-button cap —
and a matching dot in the state pill. Light colours keep a slate outline so a
white "Open" blade stays visible. Intermediate/Invalid mark the card border
only (dashed amber/red). The Appearance card previews the painted symbol.
Contract fields and defaults are unchanged.
