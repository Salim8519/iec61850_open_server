# Lane M: SLD measurement list

Branch `worktree-agent-aeb979b2914ecedcd`, reset to master 91c4fbb. Decision:
[ADR-0090](../../../decisions/ADR-0090-sld-measurement-list.md). Go engine + React only; the
Python backend is unchanged (frozen, ADR-0076). No EXE was built.

## What the owner gets

Add > Measuring > **Measurement list**: one panel with many measured values, each with its own
name, scale, offset, unit and decimals. In the sheet: pick an IED, search, click Add per value or
**Add all N shown** (numeric points only), then rename, reorder and scale. Tick several rows and a
bulk bar sets scale (quick × 1 / × 0.001 / × 1000), offset, unit or decimals for all of them at
once; fields where the ticked rows differ say "mixed". One sentence explains scaling: "Shown value
= raw value × scale + offset. Example: raw 11000 × 0.001 = 11.00 kV." The canvas panel shows
"label   value unit"; no value, stopped IED and bad quality are words + border pattern (+ icon).

## Files

Go (`studio/engine/internal`):

| File | Change |
|---|---|
| `project/contract/diagram.go` | kind `MEASUREMENT_LIST`, omittable `measurements` (cap 50) |
| `project/contract/models.go` | `omittable` field, `Measurement` model |
| `project/contract/scalars.go` | `coordinate` becomes `bounded(100000)`; scale/offset use `bounded(1e9)` |
| `project/validation_measurements.go` (new) | row checks, settle, removal helpers |
| `project/validation.go`, `validation_state.go`, `removal.go` | hooks |
| `acquisition/watched.go` | rows are watched points of a real IED |
| `runtime/plant_states.go` | rows publish `<node>:<row>` raw values |
| `project/vectors_deviations_test.go` (new), `vectors_test.go` | documented literal-message deviation |

React (`studio/frontend/src`): `api/types.ts`, `equipment/kindInfo.ts` (name, setup item,
`defaultLabel`), `registry.ts`, `symbols/EquipmentSymbol.tsx` (panel + gauge glyph),
`measurementDisplay.ts` (pure rules), `MeasurementPanel.tsx` + `measurement-panel.css` (canvas),
`useSignalInfo.ts` (cached signal description for the quality point), `stateBindings.ts` +
`signalLayout.ts` (signal layers), `tidy/tidy.ts`, `EquipmentNode.tsx`;
`features/diagram/MeasurementEditor.tsx`, `MeasurementRow.tsx`, `MeasurementBulkBar.tsx`,
`DraftField.tsx`, `measurement-editor.css`, `EquipmentSheet.tsx`, `AddMenu.tsx`, `Canvas.tsx`,
`DiagramPage.tsx`; `components/SignalSearchPicker.tsx` (optional `chooseAll`).

## Evidence

- `go test -trimpath -p 2` (GOMAXPROCS=4): `internal/project/...`, `internal/acquisition`,
  `internal/runtime/...` pass; new tests `measurements_test.go` (defaults and byte stability,
  contract errors, cross-checks, node data, removal), `watched_measurements_test.go`,
  `plant_measurements_test.go`. All 1629 golden vectors pass (6 kind-literal messages mapped by
  `pythonRecorded`, nothing else). Whole module: only `internal/launcher`
  `TestLeaseSharedWithPython` fails here (needs STUDIO_PYTHON; temp-folder lock), unrelated.
- `python studio/tools/verify.py`: file lengths, architecture, documentation, history,
  specifications and GOOSE engine PASS. Its Go step fails only in the Python-oracle tests
  (`TestRoutesAgainstFlask`, `TestFlaskParity`, `TestAgainstCPython`, ...: "No module named
  'flask' / 'waitress'"): this worktree has no `studio/.venv`. The same packages pass under
  plain `go test ./...` above.
- Frontend: `npm run check` clean; `tests/frontend/measurementList.test.ts` 7/7 (display maths,
  words for no value/stopped/text, bulk edit, typed-setting errors, reorder/new row, setup item,
  signal layers); all 235 frontend tests pass.
- Browser (Go dev server on 18765 with a scratch data folder, Vite on 15190, sample IED only,
  its port moved to 20102 and GOOSE publish off so nothing met the owner's app on 8765/5173):
  added a list, "Add all 2 shown" added GGIO1.AnIn1/AnIn2, bulk × 0.001 + kV, row 2 set to × 1,
  A, 1 decimal; saved diagram holds both rows with their settings. Running, values written
  11042.5 and 412.37 showed "11.04 kV" and "412.4 A" on canvas and in the sheet. Screenshots at
  1366x768 (pane-scaled) in the lane's scratchpad (`lane-m-screens/0..4-*.jpg`).

## Limitations

- The frozen Python engine refuses a project containing a measurement list (literal error on
  `kind`); its startup recovery keeps the autosave and opens empty. Not fixable without a Python
  change; documented in ADR-0090.
- Scaling is edited while stopped (all diagram edits need a stopped run).
- No test buttons on a measurement list (set values from the Test page or the IED).
- The picker's numeric filter also lists CO/CF integers (ctlNum, sboTimeout); they are allowed
  (numeric) but rarely useful. Quality and "IED stopped" states were checked by unit tests, not
  in the browser (the sample IED has no bad-quality source and a single IED cannot be partly
  stopped).
