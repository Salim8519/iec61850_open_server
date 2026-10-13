# ADR-0024: Relay panel and signal-card usability

Accepted: 2026-09-29. Refines ADR-0022 at the owner's request: the relay
worked but was hard to read and use.

- Canvas: the relay renders a fixed-width panel (header with name and signal
  count, one row per signal). Values are text-first chips whose marker shape
  also encodes the state (filled 1, hollow 0, dashed no value), so colour is
  never the only cue. Rows and chips ellipsize instead of overflowing.
- Names: a label still equal to its exact reference is displayed shortened by
  removing only the IED-name part of the logical device
  (`IED1Master/LLN0.Mod.stVal` -> `Master/LLN0.Mod.stVal`). The LD instance
  is kept, so refs of one IED remain distinct; unmatched refs are unchanged.
  Stored labels and bindings are not rewritten. New signals default to the
  description, else this short form.
- Properties: compact cards replace the tall per-row editor. The signal type,
  functional constraint and Triggerable/Display-only tag show while stopped.
  While running, the card reads the same live equipment store as the canvas
  (fixing values that appeared only after a change event), offers the
  existing set editor, one Pulse button towards the opposite Boolean value,
  and Resume automatic.
- Adding: an explicit "Add signals" action opens the picker inline; it stays
  open for several adds and closes with Done.

Compatibility: no project format, contract, endpoint or native change. The
separate "Pulse 1"/"Pulse 0" buttons are replaced by the single named pulse;
setting either value remains available through the set editor. Rotation is
no longer offered for relays because a panel has no symbol orientation.
