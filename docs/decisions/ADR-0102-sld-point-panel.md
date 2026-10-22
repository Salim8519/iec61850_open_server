# ADR-0102: SLD point panel

Accepted: 2026-10-22. Owner request: an SLD bay with hundreds of separate push buttons (alarms,
double points, commands, measurements of one protection IED) is unusable; one small component
per bay must open a searchable list where each point is set or operated in one step. Same shape
as [ADR-0090](ADR-0090-sld-measurement-list.md) (terminal-less panel, Go-only fields) and the
point look of [ADR-0095](ADR-0095-universal-point-kinds.md).

## Problem

A push button binds one signal. A bay of a protection relay has about 100 to 250 points, so the
diagram needs the same number of symbols, each with its own mode and colours, and nobody can
find the one they want. The relay and measurement-list panels show rows, but only read them
(relay rows may trip) and cannot hold commands or double points.

## Decision

A new diagram kind `POINT_PANEL` (Add > Protection & control > Point panel), a terminal-less
panel. The canvas shows one small fixed-width card per bay; click or Enter opens a popup with
the points; the sheet (right side) edits the name and the IED.

### Contract (Go engine, `studio/engine/internal/project/contract/point_panel.go`)

- `data.ied_id`: the one IED of the panel (`null` until chosen; text otherwise). A point panel
  holds the points of exactly one IED; a point of another IED is refused.
- `data.points`: at most `PointPanelLimit` = 300 rows (a saved panel stays near 100 KB and
  below the 500 points a real IED session may poll). Row:
  `{id, label, group, type, source, control, unit, scale, offset, decimals, pinned, healthy}`.
  - `id` identifier, `label` 1-100 characters, `group` 0-100 characters (default "").
  - `type`: `status`, `double`, `command` or `value`.
  - `source` `{ied_id, report_ref, ref}` for status, double and value rows; `control`
    `{ied_id, ref}` for command rows. The other one is `null`.
  - `unit` 0-16 characters, `scale` and `offset` in [-1e9, 1e9] (default 1.0 and 0.0),
    `decimals` 0-6 (default 2), `pinned` (default false): value rows only.
  - `healthy`: optional value the "All healthy" action writes: `true`/`false` for status,
    0 to 3 for double, a number (the raw value the IED holds) for value; none on commands.
- Both fields are *omittable*: they are dumped only when present, a fresh node of another kind
  loses them, and a new point panel always gets `points: []` and `ied_id: null`. Documents
  without a point panel keep their exact bytes. Extra keys are refused.
- Cross-checks (`validation_point_panel.go`): other kinds refuse panel fields; a panel takes no
  state mapping; the IED exists (after it is chosen); unique row ids; every signal exists on
  that IED with the type the row needs: status = BOOLEAN, double = Dbpos, value = numeric
  (INT8..INT128, INT8U..INT32U, FLOAT32, FLOAT64) with functional constraint MX; every command
  control exists on the IED and is SPC or DPC; value-only settings and pins on other types are
  refused; scale 0 is refused. Messages are plain English and say what to change.
- IED removal empties the panel of that IED (`ied_id` null, no points; the panel stays) and
  counts the panel as a binding.
- A real IED polls the points' signals (`acquisition/watched_point_panel.go`; commands have no
  value). While running, each signal row publishes its raw value as `equipment_state`
  `<node>:<row>`, as relay and measurement rows do; display = raw x scale + offset, rounded to
  `decimals`, then `unit` (browser only; IED values and the wire are unchanged).

### Requests (no new route)

Everything goes through the existing `POST /api/v1/equipment/{node_id}/test`, so the Flask
route table (and its parity test) does not change. For a point panel node the body is:

| Body | Effect |
| --- | --- |
| `{"action":"set","row":"<id>","value":true/false}` | status: writes the simulated IED's BOOLEAN |
| `{"action":"set","row":"<id>","value":0..3}` | double: writes the Dbpos position |
| `{"action":"set","row":"<id>","value":<number>}` | value: writes the raw number (whole for integer signals) |
| `{"action":"send","row":"<id>","value":true/false}` | command: Operate through the canvas command path (SPC true = ON; DPC true = Close, false = Open) |
| `{"action":"healthy"}` | writes every point that has a healthy value, in chunks of 64 |

Status, double and value writes use the sample writer of the push buttons (no command is
sent); commands use `commands.operate` with the same refusals as the canvas Operate. A real
IED is refused (ADR-0067); a stopped run answers "Run the IEDs to change points". Events
`point_panel_requested`, `point_panel_applied`, `point_panel_failed` carry a correlation id,
the IED, the reference, the point type and the value (one request and one result per batch for
"healthy"), and distinguish what was asked from what the IED then held.

## Compatibility

- Projects without a point panel: unchanged bytes, unchanged behaviour. Push buttons,
  measurement lists, relays and the Python parity results are not touched.
- The frozen Python engine does not know the kind (no Python change, ADR-0076): it refuses a
  project that contains a point panel when it opens it, keeps the autosave in its recovery
  folder and starts empty. The Go engine is the shipped one (ADR-0083); remove point panels
  before switching a data folder back to the Python engine.
- The text of the unknown-kind error is unchanged and does not list `POINT_PANEL`
  (`kindChoice` accepts the panel first and leaves the old literal list for everything else), so
  the Python golden vectors and `vectors_deviations_test.go` stay exactly as they are. A later
  change may add the name to that message together with one `pythonRecorded` line.
- `POST /equipment/{node}/test` for other kinds is unchanged.

## Limits (kept visible to the owner)

- Status, double and value points only; integer or enumerated statuses are not offered.
- A command to a real device is refused, as in Operate. Commands change the status through the
  native server, not through feedback targets.
- The panel has no terminals and does not draw IED marks on the diagram; a changed point
  republishes its row like a relay row, and the panel changes no other symbol.
- Up to 300 points per panel; make another panel for more.
