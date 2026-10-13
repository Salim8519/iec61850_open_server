# ADR-0025: Properties cards, setup checklist and canvas clarity

Accepted: 2026-09-29. Owner request: make the app easy to use with clear
colours and boundaries, improve every object's Properties sidebar, and avoid
engine mistakes.

- Properties: a header card shows the category-tinted symbol, the friendly type
  (for example "Circuit breaker · XCBR" instead of `CB`), a Ready / Setup needed
  checklist derived from existing mappings, and icon actions (hidden while
  running). Sections are bordered cards with a tinted title band and worded
  tags. Titles follow the object (Position, Measured value, Measurement point,
  Trip state, Button state, Command). While running, the Operate/Live card is
  placed first.
- Mapped signals show the description or IED-shortened reference and the IED
  name; the previous box collapsed the name to zero width.
- Canvas: equipment cards get a stronger outline and a category accent bar
  (library colours, never state). The state line is a pill whose border style
  also encodes the tone: dashed not mapped, dotted awaiting, solid live.
  Unmapped trip relays and push buttons read "Not mapped" like other objects.
- With one project IED, pickers preselect it locally; no data is written until
  the user picks a point.
- Engine safety: diagram updates equal in persisted fields (id, type, position,
  data; wire endpoints) are applied to the view but not saved. This removes
  no-op revisions caused by React Flow measurement on load and selection.

Compatibility: frontend only. No backend, contract, project format, endpoint or
native worker change. Setup status is derived, not stored.
