# ADR-0043: Interlocking is friendly hints only

Accepted: 2026-10-14. Owner decision: "only hints, friendly". Specification:
[Interlock hints](../specs/interlock-hints.md).

Studio had no interlocking. The legacy demo has CILO logic, and the native worker has an
unused per-control `blocked` flag.

**Decision**
- `domain/interlock_hints.py` (pure) gives plain-language hints for CB/DS/ES from the
  energisation sections. They are published with the `energization` event and shown
  under Open/Close in an amber dashed box.
- Nothing is ever blocked. Studio's Operate stays enabled, external MMS commands are
  accepted as before, and the native `blocked` flag stays unused.
- An external command against a hint adds one `interlock_hint_observed` Activity entry.
  Studio marks its own Operate first so it is never reported as external.
- The wording never claims IEC interlocking (CILO): the box says the hints come from
  the drawing.

**Compatibility:** additive. MMS behaviour is unchanged; the protocol test confirms an
external command against a hint is still executed.
