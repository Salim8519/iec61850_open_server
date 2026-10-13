# Change 0005: scenarios, IED deletion and refreshed colors

Date: 2026-09-09.

## Behavior

- Saved, typed scenario sequences support timed set-value and check-value steps.
  Sequences can start with Run or explicitly. Pause freezes their simulation clock.
  A failed step stops its sequence and emits an explicit failure event. A scenario
  write suspends an existing generator for that signal, like a manual override.
- The sequence editor offers reference search, IED selection, editable draft
  steps, saved sequence management and runtime status. Structural editing is
  locked during a run. Native-model assertions are not independent MMS evidence.
- Delete IED requires a stopped simulation and an impact confirmation. Equipment
  and wires remain; references to that IED, sources and dependent scenario steps
  are removed. Empty resulting sequences are removed. Before changing the project,
  a recovery .subsim archive is written under data/recovery. Original files remain
  untouched. Open Project can restore the complete pre-deletion state.
- Shared color tokens now provide navy navigation, blue selection surfaces and
  teal/violet library accents, following the user's request and ADR-0002.

## Verification

Full suite: 30 tests pass. This includes native scenario pause/resume and assertions,
scenario ordering/failure behavior, deletion confirmation/stopped guards, reference
cleanup, recovery round-trip and the existing MMS/import/packaging checks.
Frontend production build passes. Length, architecture and documentation checks pass.

The scenario integration test exposed a native report-event field collision that
previous narrow worker tests did not cover. Native kind/id/timestamp fields are
now namespaced before application event emission. Consumer exceptions are logged
without killing the native request/reply reader. This is a runtime correction,
not evidence that every native event path has been exhaustively tested.

Visual browser acceptance is still pending. CSV replay, control/interlock scenarios,
full load acceptance, LAN setup and clean-machine verification remain separate gates.
The user's final hibernation instruction is recorded in the acceptance checklist;
no hibernation occurs at this partial milestone.
