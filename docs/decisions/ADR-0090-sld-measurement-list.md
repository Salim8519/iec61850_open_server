# ADR-0090: SLD measurement list

Accepted: 2026-10-20 (owner: "in the add components, can u add measurement list, just like
protection relays, sometimes user wants to modify multiple measurements, some have different
scaling, make the component nice and easy to work with please"). Lane M; log
[lane-m-measurement-list](../architecture/go-engine/streams/lane-m-measurement-list.md).

## Problem

CT, VT and Analog source each show one point as the raw value (`String(value)`): no units, no
scaling, one symbol per value. Watching a feeder's voltages, currents and power meant many
symbols and mental arithmetic (11000 V shown as "11000").

## Decision

A new diagram kind `MEASUREMENT_LIST` (Add > Measuring > Measurement list), a terminal-less
panel like the protection relay (interactions.md relay rules apply: one row per entry, entries
edited while stopped).

Contract (Go engine, `studio/engine/internal/project/contract`):

- `data.measurements`: at most 50 rows `{id, label, source: {ied_id, report_ref, ref}, scale,
  offset, unit, decimals}`; `id` is an identifier, `label` 1-100 characters, `scale` and
  `offset` finite numbers in [-1e9, 1e9] (default 1.0 and 0.0), `unit` 0-16 characters
  (default ""), `decimals` 0-6 (default 2). Extra keys are refused.
- The field is *omittable*: it is dumped only when present, and a fresh node of another kind
  loses an empty `measurements` key. Documents without measurement lists keep their exact
  bytes; a new measurement list always gets `measurements: []`.
- Cross-checks: only a measurement list accepts rows ("Only a measurement list accepts
  measurements"); unique row ids; the bound signal must exist and be numeric (INT8..INT128,
  INT8U..INT32U, FLOAT32, FLOAT64, any FC); a list takes no state mapping and no command.
- IED removal drops the rows of that IED (the list stays) and counts the list as a binding.
- A real IED polls the rows' points with their quality/time points (watched values).
- While running, each row publishes its raw value as `equipment_state` `<node>:<row>`, as
  relay rows do. Display = raw × scale + offset, rounded to `decimals`, then `unit`; the
  browser applies it (a display concern; IED values and the wire are unchanged).

Frontend: canvas panel (title, "N measurements", rows "label   value unit"; no value "—" dashed,
stopped IED "Stopped" dashed, bad quality a double border, warning icon and the words "Bad
quality"); sheet editor with a numeric-only multi-add picker ("Add all N shown"), compact rows
(name, scale, offset, unit, decimals, value now, reorder, remove) and a bulk bar for ticked rows
(quick scale × 1 / × 0.001 / × 1000, scale, offset, unit, decimals).

## Compatibility

- Projects without a measurement list: unchanged bytes, unchanged behaviour.
- The frozen Python engine does not know the kind (no Python change, ADR-0076): it refuses a
  project that contains a measurement list when it opens it (pydantic literal error on `kind`),
  keeps the autosave in its recovery folder ("could not be reopened") and starts empty. The Go
  engine is the shipped one (ADR-0083); remove measurement lists before switching a data folder
  back to the Python engine.
- Pydantic's message for an unknown kind now also names 'MEASUREMENT_LIST'. The Python golden
  vectors predate the kind; `vectors_deviations_test.go` maps exactly this text back and nothing
  else.
- Scaling is edited while stopped, like every diagram edit (`PUT /project/diagram` requires a
  stopped run).
