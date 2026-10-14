# ADR-0042: Live colouring computed by the backend, shown by line style

Accepted: 2026-10-14. Owner request: live (energised) colouring on the SLD.
Specification: [Live colouring](../specs/energization.md).

**Decision**
- **Computation:** a pure `domain/energization.py` turns the diagram plus authoritative
  switch positions into sections. It uses union-find over wires; switches join their
  two sides only when Closed, and unknown positions give "maybe live".
- **Publishing:** `simulation/energization_states.py` publishes one `energization` event
  when the result changes, and the same snapshot is served with `/equipment/states`.
- **Why the backend:** AGENTS.md forbids duplicating authoritative simulation logic in the
  browser, and the interlock hints (ADR-0043) need the same sections.
- **Display only:** nothing is written to an IED. Nothing is coloured when stopped.
- **Styles:** every state has its own line pattern and width as well as its colour
  (never colour alone), and a legend names them in words.

  | State | Line |
  | --- | --- |
  | Live | red solid, 4 px |
  | Maybe live | amber dotted |
  | Dead | grey dashed |
  | Earthed | green dash-dot |
  | Earth fault | purple long dash, 5 px |

  The colours sit in `equipment/energization.ts`, like the wire colour in `Canvas.tsx`,
  and are tested at 3:1 or better against the canvas.

**Compatibility:** additive. The event is new, and `/equipment/states` gains an
`energization` field. Unchanged wires keep their object, so live updates don't
re-render the canvas.
