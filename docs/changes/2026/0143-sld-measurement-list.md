# 0143: SLD Measurement list (many measurements, each with its own scaling)

Date: 2026-10-20. Owner: "in the add components, can u add measurement list, just like protection
relays, sometimes user wants to modify multiple measurements, some have different scaling".
Decision: [ADR-0090](../../decisions/ADR-0090-sld-measurement-list.md). Lane M (merged 59482ed),
log [lane-m-measurement-list](../../architecture/go-engine/streams/lane-m-measurement-list.md).

## Change

- Add › Measuring › Measurement list. Sheet: pick an IED, search its number signals, Add or
  "Add all N shown"; rows with name, live value, scale, offset, unit, decimals, reorder, remove;
  tick rows and set scale/offset/unit/decimals together (quick ×1 / ×0.001 / ×1000, "mixed"
  shown when ticked rows differ); one-sentence hint "raw 11000 × 0.001 = 11.00 kV".
- Canvas panel: "label  value unit" rows; no value "—", stopped "Stopped", bad quality with a
  double border, icon and words (never colour alone); empty state "Add measurements in the sheet".
- Go: `data.measurements` (≤ 50 rows), validation (number signals only, unique ids, plain
  errors), removal with its IED, polled points for real IEDs, live states; projects without a
  list save byte-identical bytes.

## Evidence

- Go project/acquisition/runtime tests pass incl. new contract, validation, removal, polled
  points; 1629 golden vectors pass (six unknown-kind messages mapped by a test-only helper).
- Frontend 235/235 in the lane (new `measurementList.test.ts` 7/7), `npm run check` clean.
- Browser on a scratch Go server (sample IED): raw 11042.5 and 412.37 shown as "11.04 kV" (×0.001)
  and "412.4 A" (×1, 1 decimal); screenshots in the session scratchpad.

## Limitations

- The frozen Python engine refuses a project containing a measurement list (keeps it in recovery).
- Scaling is edited while stopped (every diagram edit needs a stopped run); no test buttons on the
  list yet; the picker also offers numeric command/config attributes (ctlNum, sboTimeout).
