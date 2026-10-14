# 0065: Live colouring

Date: 2026-10-14. Owner request: live (energised) colouring of the SLD.
Specification: [Live colouring](../../specs/energization.md) (POWER-01..06).
Decision: [ADR-0042](../../decisions/ADR-0042-live-colouring.md).

## Change

**Backend**
- `domain/energization.py`: sections and states.
- `simulation/energization_states.py`: publisher.
- `simulation/equipment_states.py`: `positions()` and an `after_publish` hook.
- `coordinator.py`: wiring.
- `api/equipment.py`: snapshot.

**Frontend**
- `equipment/energization.ts`: store, styles, edge projection, sentence.
- `useStudio.ts`: event and snapshot.
- `Canvas.tsx`: styled wires.
- `EnergizationLegend.tsx` and `energization.css`.
- Sheet "Power" line.
- Activity summary in `eventDetails.ts`.

**Tests**
- `tests/unit/test_energization.py`
- `tests/protocol/test_energization_runtime.py`
- `tests/frontend/energization.test.ts`

## Evidence

**Tests**
- 4 unit and 1 native protocol test (real MMS operate → `energization` events
  live → dead, published only on change).
- 3 frontend tests: distinct patterns, 3:1 contrast and stable edge objects.

**Browser** (isolated 8799 instance, scratch data; the owner's session was untouched)
- Setup: a grid Infeed, a busbar, a Line feeder and a Transformer bay linked to the
  *Mapped* test IED.
- With Q1/Q0 closed:
  - wires down to Q9 were red solid;
  - the parts past the unlinked Q9 were amber dotted "Maybe live";
  - the legend named both styles.
- After an external open of Q1, the feeder turned grey dashed. The transformer bay stayed
  live, correctly fed back from its live transformer Infeed.

## Limitations

- Junction dots are not coloured; their wires are.
- Earthed and maybe-earthed are not distinguished.
- The owner's visual acceptance is pending.
