# 0042: Relay panel and signal-card usability

Date: 2026-09-29. Owner report: the relay works but is not friendly, and its
look feels harsh and hard to distinguish. Decision: ADR-0024.

Observed before (screenshots of the live project): canvas rows at 10px spilled
outside the node with full raw references as names and near-identical values;
the Properties editor used ~140px per signal with a red remove box on every
row, no type or triggerability until running, and "Add signal" hidden in a
collapsed disclosure. While running, the cards showed no value until a change
event because they read a different value map than the canvas.

Changes (frontend only):

- `RelayPanel.tsx` + `relay-panel.css`: panel header, overflow-safe rows,
  text-first value chips with shape markers.
- `RelayStatusRow.tsx` + `relay-signals.css`: compact cards, quiet tools,
  type/FC/Triggerable tags, single named Pulse, Resume automatic.
- `RelayStatusEditor.tsx`: inline "Add signals" picker that stays open;
  no focus stealing during multi-add; values from the shared live store.
- `relayDisplay.ts`: IED-aware short refs, labels, value text, pulse target;
  `liveContext.ts` holds the live-value and IED-name contexts.
- Shared `SignalSearchPicker` shows IED-shortened refs (full ref on hover)
  and accepts a button label ("Add" for relays).

Evidence:

- 6 new `relayDisplay.test.ts` tests; frontend node tests: 29 pass.
- Backend unit + integration: 124 pass; `npm run check` and build pass;
  `verify.py` passes.
- Headless-Chrome screenshots of the dev session (stopped and running) and
  of an isolated copy (own data dir, IED on 127.50.0.9) while running: Set
  True on a Boolean showed `1` on canvas and card, the pulse button then
  offered False, and Resume automatic appeared.

Limitations: UI-only; no new MMS evidence (initiation endpoints unchanged).
Owner visual acceptance pending.
