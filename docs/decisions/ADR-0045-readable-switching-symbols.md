# ADR-0045: Switching symbols readable from a distance

Accepted: 2026-10-15. Owner request: the CB, DS and ES symbols are unclear from a distance,
and state changes are barely noticed unless you zoom in. They must be clear but professional.

**Problem**
- The CB state was a 12-unit contact that moved sideways by 6 units inside a 22-unit body,
  which is about 3 px at half zoom.
- The DS and ES blades were 20 units long and swung 35°. A blade painted white on a white
  card almost disappeared.
- Strokes were 2.2 units at every zoom, so they dropped below 1 px when zoomed out.

**Decision**
- `symbols/SwitchingGlyphs.tsx` draws the switching symbols (same 54 px symbol and 60-unit
  viewBox, so the 108×120 card size, Tidy up and bay templates are unchanged):
  - CB body is 32 units. Closed: the body is filled and a whole conductor runs through it.
    Open: the body is hollow and the conductor is broken. This follows the SCADA
    convention that filled means closed and hollow means open.
  - DS and ES use an IEC contact bar on the fixed side and a 34- or 30-unit blade that swings
    40° when open (20° Intermediate). A painted blade has a heavy outline, so a white
    "open" blade stays visible.
  - The Unknown/Invalid cross is 16 units and never resembles Open.
- Every symbol takes its stroke from `--sym-stroke` (3 px). The canvas adds a zoom band
  class (`equipment/readability.ts`, `features/diagram/useZoomTier.ts`):
  - near is zoom ≥ 0.85;
  - mid (stroke 4) is 0.5–0.85;
  - far (stroke 5.5) is below 0.5.

  Only stroke width changes, so nodes are never remeasured and wires never move.
- A confirmed CB/DS/ES position change (between Open, Closed, Intermediate and Invalid)
  pulses a blue ring twice, for about 2 s. With reduced motion it shows a static ring for 2 s.
  The first value after load or run start does not pulse. The pulse is an attention cue
  only. The state is still shown by geometry, words and colour.

**Compatibility:** display only. There is no project, contract, SCL or MMS change, and
user state colours (ADR-0026) still apply. The default symbol stroke grows from 2.2 to
3 for all kinds, and the trip relay and push button now share that stroke.
