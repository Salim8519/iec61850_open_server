# Spec: Smart point picker and point kinds

Decision: [ADR-0113](../decisions/ADR-0113-smart-point-picker.md) (extends [ADR-0095](../decisions/ADR-0095-universal-point-kinds.md)).
Change: [0184](../changes/2026/0184-smart-point-picker.md).
Owner request (2026-10-23): finding a point in the SLD pickers is still a hassle; the type filter must
cover all IEC 61850 standard points; follow the kind colouring.

## Behaviour contract

- **Kinds.** Go (`signalindex/point_kind.go`) and the browser (`points/pointKind.ts`) give every
  standard functional constraint × basic type, and the data classes that refine them, the kind of the
  one table in `studio/tests/fixtures/point_kinds/iec61850.json`. UN only for vendor types.
- **Engine (Go only, `GET /ieds/<id>/signals`).** `annotate=1` adds `point_kind` to each item;
  `ln=<LD/LN>` filters one logical node; `ln_counts=1` adds `{"LD/LN": n}` counted without `ln`;
  `kind_counts=1` counts without `point_kind`; `near=<text>` adds `suggested` (at most 8 matching
  items whose description, then reference, holds the text as a whole word, case ignored, each with
  `suggested_by`). Without these parameters the answer is byte-identical to Flask's.
- **Picker.** Every SLD picker (state, contacts, measurements, relay rows, tap changer) and the Test
  and gateway pickers show, under the search: type chips with the shared badge and the engine's count
  (only kinds with rows, the chosen one always, "All types (n)"), the slot's type chosen for the user
  with one line saying why, an object (logical node) select with counts, then the report select.
  "Show quality and time" (off) appears where helpers can be listed. Points that mention the
  component's name come first, each saying where the name was found. The point panel adder has the
  same object select. Chips wrap into columns inside a narrow sheet; nothing is cut at 1366×768.
- **Older engine.** An answer without `kind_counts` (frozen Python engine) gives the old picker: All
  types, the old type dropdown, the browser's own kind per row, no objects and no suggestions.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| PICK-01 | Go and the browser answer every case of the shared kind table; every standard FC and basic type is in it and none is UN. | `studio/engine/internal/acquisition/signalindex/point_kind_vectors_test.go::TestPointKindMatchesTheSharedTable`, `studio/tests/frontend/pointKindVectors.test.ts::kindOf61850 answers every case of the shared IEC 61850 kind table`, `studio/tests/frontend/pointKindVectors.test.ts::every standard data class has a kind when a list knows only the class` |
| PICK-02 | A request without the Go-only parameters is answered as Flask does; each new parameter only adds its own key. | `studio/engine/internal/acquisition/signalindex/signalindex_test.go::TestSignalsMatchPythonRoute`, `studio/engine/internal/acquisition/signalindex/picker_params_test.go::TestPickerParametersOnlyAddTheirKeys` |
| PICK-03 | `annotate=1` gives each item Go's kind; kinds are computed once per index and data-class version. | `studio/engine/internal/acquisition/signalindex/picker_params_test.go::TestAnnotateGivesEachItemItsKind`, `studio/engine/internal/acquisition/signalindex/picker_params_test.go::TestKindsAreCachedPerDataClassVersion` |
| PICK-04 | `ln` filters one logical node; `ln_counts` counts with every filter but `ln`; `kind_counts` keeps `ln`, drops `point_kind`. | `studio/engine/internal/acquisition/signalindex/picker_params_test.go::TestLogicalNodeFilterAndCounts`, `studio/engine/internal/acquisition/signalindex/point_kind_test.go::TestPointKindFilter` |
| PICK-05 | `near` suggests at most 8 points that mention the text as a whole word, descriptions first, every other filter applied. | `studio/engine/internal/acquisition/signalindex/picker_params_test.go::TestNearSuggestsPointsThatMentionTheName`, `studio/engine/internal/acquisition/signalindex/picker_params_test.go::TestNearSuggestsAtMostEight` |
| PICK-06 | Each slot opens on its kind: contacts SP, one position point DP, measurements MV, tap position IS, number command SE, a named command kind. | `studio/tests/frontend/signalPicker.test.ts::each slot opens on the type it takes` |
| PICK-07 | Chips show only kinds with rows (the chosen one always) in the kinds' order, with the shared badge, a tick when chosen and wrapping columns. | `studio/tests/frontend/signalPicker.test.ts::chips show only the types that have rows, the chosen one always, in the kinds order`, `studio/tests/frontend/signalPicker.test.ts::the picker shows suggestions first, keeps the old type list for older engines and says why a type is chosen` |
| PICK-08 | Objects are listed in natural order with counts; one logical device shows the node alone. | `studio/tests/frontend/signalPicker.test.ts::objects are listed in natural order with their counts; one logical device shows the node alone` |
| PICK-09 | The smart picker hides quality and time by default and asks the engine for counts, kinds, object and suggestions; a plain list sends none of them. | `studio/tests/frontend/signalPicker.test.ts::the smart query hides helpers by default and asks for counts, kinds, object and suggestions` |
| PICK-10 | Rows use the engine's kind; without the new fields the picker behaves as before (All types, old dropdown, browser kind). | `studio/tests/frontend/signalPicker.test.ts::rows use the kind from the engine; an answer without it (older engine) falls back to the browser rule`, `studio/tests/frontend/signalPicker.test.ts::the picker shows suggestions first, keeps the old type list for older engines and says why a type is chosen` |
| PICK-11 | Every SLD picker passes the component's name and its slot; the point panel adder has the object select; rows keep the badge and edge. | `studio/tests/frontend/signalPicker.test.ts::every SLD picker passes the component name and its slot`, `studio/tests/frontend/deviceKinds.test.ts::every point list of the area shows the kind badge and the shared row edge` |

## Evidence

Speed (90 000 signals, `signalindex/bench_test.go`): type filter with counts ~88 ms → ~2.9 ms per
request; full picker request ~6.3 ms; Search, Build and their allocations unchanged. Browser check on
the public samples at 1366×768 and 1920×1000 in change 0184.
