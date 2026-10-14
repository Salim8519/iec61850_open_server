# 0058: Build the single-line diagram from the SCD

Date: 2026-10-14. Owner request: build roadmap features A to D. This is feature A.
Specification: [SLD import](../../specs/sld-import.md). Decision:
[ADR-0035](../../decisions/ADR-0035-sld-import-from-scd.md).

## Change

**Backend**
- `scl/substation.py` reads `Substation`/`VoltageLevel`/`Bay`/`ConductingEquipment`/
  `ConnectivityNode`/`Terminal`/`LNode` and the `esld` coordinates.
- `domain/sld_layout.py` handles kind mapping, junction rows, wires, ids, the coordinate
  layout and the limits. `domain/sld_auto_layout.py` provides the grid layout and
  `domain/sld_bindings.py` the `LNode` bindings.
- `projects/sld_import.py` and `api/sld.py` provide `GET /project/substation-sources`
  and `POST /project/diagram/from-scl/preview`.

**Frontend**
- A **Build from SCD** button in the Diagram toolbar opens a dialog:
  - choose the file and Add beside or Replace;
  - the preview shows counts first, then the *Needs a link*, *No symbol yet* and
    *Notes about the file* lists;
  - Replace requires a confirmation checkbox;
  - Apply is one undo step.
- `equipment-node.css` renders junctions at their intended 40 px.

**Answered spec questions** (the owner delegated the proposed answers):
- Partial coordinates switch the whole station to the automatic layout.
- No bay or voltage-level labels on the canvas.

## Evidence

**Tests**
- `test_substation_scl.py`, `test_sld_layout.py`, `test_sld_bindings.py` and
  `test_sld_import_api.py`: 19 tests passing.
- `sldImport.test.ts`: frontend at 61 passing.

**Browser** (an isolated instance with scratch data; the owner's running session was not
touched)
- `model_substation.scd`, coordinate layout: two 50 kV busbars, two feeders, two
  transformer bays and two MV busbars with three feeders each. That is 46 items, 44
  junctions and 90 wires, with no overlapping cards. VTs on a busbar get their junction
  beside them.
- `open_substation.scd`, automatic layout: QA1 and QB1 bind their position to IED1_XCBR
  and show *Awaiting value*. Their command is unbound with a plain reason.

## Limitations

- Transformers, lines, arresters, generators and phase `SubEquipment` aren't drawn.
- No bay or voltage-level labels on the canvas.
- Not yet tried on real vendor SCDs with `CSWI` `LNode`s.
- The owner's visual acceptance is pending.
