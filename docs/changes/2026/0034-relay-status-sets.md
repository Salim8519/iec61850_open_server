# 0034: Relay status sets

Date: 2026-09-17. User-requested rework: relays become a user-named set of
status signals that can be initiated, and are no longer connectable wire-path
objects.

A relay on the canvas is now a status panel: the Inspector lists named status
entries (name + one exact imported signal each, add/remove/rename while
stopped, at most 50 per relay). Any signal may be assigned; non-writable ones
display live values with initiation disabled and a "Display only" badge. While
running, each entry offers Set, a 1000 ms Boolean pulse and Resume automatic
through the existing `/signals/update`, `/signals/pulse` and
`/signals/release-override` endpoints — no new backend runtime code. Relay
nodes lost their terminals so they cannot be connected, and the relay symbol
lost its connector stubs.

Live values project through the equipment state store using composite keys
`nodeId:entryId`, so `/equipment/states`, event hydration and canvas rows work
through the unchanged plumbing. The project format is now 9: opening an older
project converts each relay's single direct state mapping into one named
status entry with the exact reference preserved, shows the migration notice
and keeps the pre-change backup. IED removal previews count relay status
bindings and removal strips only the entries bound to removed IEDs.

Evidence:

- 9 new unit tests in `studio/tests/unit/test_relay_statuses.py` pass:
  acceptance, display-only non-writable binding, unknown-signal errors,
  legacy mapping rejection, non-relay rejection, duplicate identifiers,
  format-8 migration conversion, removal counting/stripping and composite-key
  projection. Unit suite: 106 pass; integration: 17 pass.
- 17 frontend node tests pass, including 3 new `relayStatuses.test.ts`
  tests for relay binding collection and value keys.
- `npm run check` and the Vite production build pass.
- `studio/tools/verify.py` passes all checks.

Browser visual acceptance remains: migration notice on the current project,
adding and naming statuses, live panel rows while running, initiation of a
writable entry and the display-only badge.
