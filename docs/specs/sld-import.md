# Spec: SLD import from the SCD

Decision: [ADR-0035](../decisions/ADR-0035-sld-import-from-scd.md). Change: [0058](../changes/2026/0058-sld-import.md).
Owner decisions (2026-10-14): busbars are drawn as `JUNCTION` nodes, no retired
kind is reintroduced, and the project format stays at 12.

## Purpose

An engineer loads a whole-substation SCD and gets a correct, operable single-line
diagram in one step instead of drawing it by hand. Breakers and switches that the
SCD links to an IED are bound to that IED's position signal and control, so the
diagram works immediately with simulated or real IEDs.

## Inputs (IEC 61850-6)

Read from the original SCL stored in the project (`scl_sources`). The file is
never modified.

| Element | Used for |
| --- | --- |
| `Substation`, `VoltageLevel` (`Voltage`), `Bay` | Grouping, labels, layout rows and columns |
| `ConductingEquipment@type`, `@name` | Equipment kind and label |
| `Terminal@connectivityNode` (fallback: `substationName`/`voltageLevelName`/`bayName`/`cNodeName`) | Connections |
| `ConnectivityNode@name`, `@pathName` | Junctions and wires |
| `LNode` (`iedName`, `ldInst`, `prefix`, `lnClass`, `lnInst`) under equipment | Signal and control binding |
| `esld:*` layout attributes (namespace `https://transpower.co.nz/SCL/SSD/SLD/v0`) | Positions when present |
| `PowerTransformer`, `GeneralEquipment`, `Function`, other equipment types | Listed as not drawn |

## Behaviour contract

**Equipment mapping**

| SCL | Studio | Notes |
| --- | --- | --- |
| `CBR` | `CB` | |
| `DIS` | `DS` | |
| `DIS` with a terminal whose `cNodeName` is `grounded` | `ES` | The grounded terminal gets no junction |
| `CTR` | `CT` | |
| `VTR` | `VT` | |
| `ConnectivityNode` with ≥3 drawn terminals | a row of `JUNCTION`s, one per distinct terminal position, chained by wires | Busbars render as a bar |
| `ConnectivityNode` with 1 drawn terminal | one `JUNCTION` | Shows where undrawn equipment connects |
| `ConnectivityNode` with exactly 2 drawn terminals | one direct wire | No junction is drawn |
| Every other type (`IFL`, `SAR`, `CAP`, `REA`, `GEN`, `PowerTransformer` …) | not drawn | Listed in the preview with the reason |

**Bindings from `LNode`**
- An `LNode` resolves to an IED reference using the IED's `ldName` when present,
  otherwise `iedName + ldInst`, then `prefix + lnClass + lnInst`.
- **State** comes from the `XCBR`/`XSWI` `Pos` of the equipment (direct mode).
- **Control** prefers a `CSWI` `Pos` DPC on the same equipment, because real IEDs are
  operated through CSWI. If there isn't one, it uses a controllable `XCBR`/`XSWI` `Pos` DPC.
- A binding is created only if the IED is loaded in the project and the reference
  exists with the required CDC. Every binding passes the normal diagram validation
  (`validate_diagram`).
- When a binding can't be created, the equipment is drawn unbound, with a plain reason
  such as "IED P1 is in the SCD but not loaded" or "No Pos signal found under XCBR1".
  An IED the project can never load (no server, several servers, or its import fails) says
  why instead: "IED P1 is in the SCD but cannot be loaded: …" (Go engine, ADR-0114).
- `iedName="None"` (an unassigned `LNode`) never produces a binding.

**Layout**
- When every drawn item has `esld` coordinates, they are used: scaled so that symbol
  cards don't overlap, and snapped to the 20 px grid.
- Otherwise a deterministic layout is used: voltage levels as rows, bays as columns,
  and equipment stacked from the busbar side down by connection order.
- The same input always produces an identical diagram: same node ids, positions and
  wires.
- Node ids are `scl-` plus the first 16 hex digits of the SHA-256 of
  `source_id + "/" + SCL path`, so they satisfy the node-id pattern.
- Terminal counts use **drawn** equipment only; a transformer winding on a node is not
  counted (it is listed as not drawn).
- Wire sides follow geometry: a mostly-vertical wire runs from the upper card's bottom
  terminal to the lower card's top terminal; a mostly-horizontal wire runs from the
  left card's right terminal to the right card's left terminal.
- A junction row sits on the drawn busbar height when the SCD has one, otherwise in
  the widest vertical gap between its members. A card the row runs through gets its
  junction beside it, never on top of it.
- Control comes only from a `CSWI` or `XCBR`/`XSWI` `LNode` named on the equipment.
  When the equipment names only a non-controllable `XCBR`, control stays unbound
  with the reason "choose the control in Properties"; no `CSWI` is guessed.

**Preview and apply**
- Preview never changes the project. It returns the proposed diagram plus four lists:
  drawn, not drawn (with reason), unbound (with reason) and diagnostics.
- Apply has two modes:
  - **Replace** clears the current diagram. It needs an explicit confirmation that names
    how many items will be removed.
  - **Add beside** places the import to the right of the current drawing. It skips any
    node whose id is already on the diagram and reports it as "already on diagram".
- Apply goes through the existing revision-guarded diagram save and the undo history.
  One undo restores the previous diagram.
- If the result would exceed the diagram limits (2,000 nodes, 5,000 wires), it is
  refused with a diagnostic, never truncated.
- If the project has several SCL sources containing a `Substation` section, the user
  picks one. Sources without one are not offered.

**Interface** (Diagram place; UI-01 unchanged)
- The action "Build from SCD" appears in the Diagram place.
- When no loaded file has a `Substation` section, the action is disabled with a hint:
  "This file has no substation layout. Ask for the SCD from the system integrator."
- The preview explains each list in plain English (beginner-guidance rules) and shows
  counts before details.

## Out of scope

- Drawing transformers, lines, generators, capacitors or reactors.
- Phase-level `SubEquipment`.
- Writing a `Substation` section into SCL.
- Syncing diagram edits back to SCL.
- Multi-SCD merging.
- Function/sub-function views.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| SLD-01 | Preview never changes the project, its revision or the stored SCL. | `studio/tests/integration/test_sld_import_api.py::test_preview_does_not_change_project` |
| SLD-02 | `CBR`, `DIS`, `CTR` and `VTR` map to `CB`, `DS`, `CT` and `VT`. A `DIS` to `grounded` maps to `ES`. | `studio/tests/unit/test_sld_layout.py::test_equipment_kind_mapping` |
| SLD-03 | Every element that isn't drawn appears in *not drawn* with its type and a reason; nothing is dropped silently. | `studio/tests/unit/test_sld_layout.py::test_unsupported_equipment_is_listed` |
| SLD-04 | A connectivity node with exactly two drawn terminals becomes one wire; any other count becomes `JUNCTION`s. | `studio/tests/unit/test_sld_layout.py::test_connectivity_nodes_become_wires_or_junctions` |
| SLD-05 | A terminal whose node can't be resolved produces a diagnostic and no wire. | `studio/tests/unit/test_substation_scl.py::test_unresolved_terminal_is_diagnosed` |
| SLD-06 | State binds to `XCBR`/`XSWI.Pos`; control prefers `CSWI.Pos` DPC over a controllable `XCBR`/`XSWI.Pos`. | `studio/tests/unit/test_sld_bindings.py::test_cswi_control_is_preferred` |
| SLD-07 | A binding is created only for a loaded IED and an existing reference with the right CDC; otherwise the equipment is unbound with a reason. | `studio/tests/unit/test_sld_bindings.py::test_missing_ied_leaves_equipment_unbound` |
| SLD-08 | Every generated diagram passes `validate_diagram` unchanged. | `studio/tests/unit/test_sld_layout.py::test_generated_diagrams_validate` |
| SLD-09 | `esld` coordinates are used when present, snapped to the grid, with no overlapping cards. | `studio/tests/unit/test_sld_layout.py::test_esld_coordinates_are_used_without_overlap` |
| SLD-10 | The same input always produces the identical diagram (ids, positions, wires). | `studio/tests/unit/test_sld_layout.py::test_layout_is_deterministic` |
| SLD-11 | *Add beside* never duplicates a node id already on the diagram. | `studio/tests/unit/test_sld_layout.py::test_add_beside_skips_existing_ids` |
| SLD-12 | Results above the diagram limits are refused, never truncated. | `studio/tests/unit/test_sld_layout.py::test_oversized_station_is_refused` |
| SLD-13 | Apply is one undo step and uses the revision-guarded save. | `studio/tests/frontend/sldImport.test.ts::apply is a single undo step` |
| SLD-14 | The action is disabled with a plain hint when no source has a `Substation` section. | `studio/tests/frontend/sldImport.test.ts::disabled without substation section` |
| SLD-15 | No `CSWI` is guessed from naming; control comes only from `LNode`s written on the equipment. | `studio/tests/unit/test_sld_bindings.py::test_csw_is_never_guessed_from_naming` |
| SLD-16 | Partial drawing coordinates switch the whole station to the automatic layout, with a note. | `studio/tests/unit/test_sld_layout.py::test_partial_coordinates_use_automatic_layout` |
| SLD-17 | An IED that can never be loaded is named with the reason, not as "not loaded" (Go engine). | `studio/engine/internal/project/sld/bindings_test.go::TestBindSaysWhyAnIedCannotBeLoaded`, `studio/engine/internal/scl/vendor_station_test.go::TestUnloadableIedsSayWhy` |

## Acceptance evidence

Recorded in change [0058](../changes/2026/0058-sld-import.md):
- `scd/model_substation.scd` uses its `esld` layout: 46 items drawn (12 CB, 18 DS,
  12 CT, 4 VT), 44 junctions and 90 wires. The 8 IFL, 2 SAR and 2 transformers are
  listed as not drawn.
- `scd/open_substation.scd` uses the automatic layout. QA1 and QB1 bind their position
  to `IED1_XCBRGenericIO/XCBR1.Pos.stVal` and `XSWI2.Pos.stVal`. Their command stays
  unbound, because the SCD names only `XCBR1`/`XSWI2`, which aren't controllable in this
  file.
- A browser walkthrough of preview, add beside, replace with confirmation, and apply.
- Pending: the owner's visual acceptance, and real vendor SCDs that carry `LNode`s
  with `CSWI`.

## Answered questions

1. Partial `esld` coordinates → **automatic layout for the whole station**, with a
   note. (Owner delegated the proposed answers, 2026-10-14.)
2. Bay and voltage-level names on the canvas → **not in this feature**; no label
   primitive is added, so the contract is unchanged. Names appear in the preview and
   in junction labels.
