# ADR-0095: Universal point kinds, edit pin and compact lists

Accepted: 2026-10-21. Owner request after [ADR-0094](ADR-0094-higher-contrast-controls.md):
colour and type must be universal ("double points apply to all, same for other kinds of
tools"), lists must show more points, and editable values must say so. An app-wide check
found that 21 lists displayed types in 10 different ways, with 5 separate type tables and no
shared colour. Three tools also repeated the same hand-picked colours. The owner chose
"Badge + edge" and one-line rows in a clickable mockup.

## Decision

- **One classifier**: `frontend/src/points/pointKind.ts` maps protocol type information to one
  kind: SP single point, DP double point, IS integer status, MV measured value, CT counter, SC
  single command, DC double command, SE set point, CM command (class unknown), CF setting, TX
  text/other, AT quality/time helper, UN unknown. The inputs are IEC 61850 FC + basic type
  (+ data class), IEC 104 type ids, gateway point kinds, master groups and commands
  (IEC 104, DNP3, Modbus), and SNMP types. Kinds never come from reference names. An unknown
  type shows UN with the raw type; it is never guessed.
- **One look**: `ui/KindBadge.tsx` (letters + symbol; the hover text gives the full name and
  the raw type) and `kindRow()`, which gives a typed row a 5 px left edge in the kind colour.
  Colours exist only as `--kind-*` tokens: teal SP, violet DP, slate IS, sky MV, indigo CT,
  magenta commands, stone CF, grey TX/AT/UN. Green, amber and red stay reserved for status.
  Commands use a white badge with a double border. Helper attributes (q, t) and unknown types
  use a dotted badge.
- **One filter**: `ui/kindFilter.ts` adds one "All types" list to the shared FilterBar (a
  dropdown, so the filters keep to one row; chips stay possible with `select = false`). Each
  entry shows the kind's symbol, letters and count, and kinds with no rows are hidden. Device
  data points get their counts from the engine (`kind_counts=1`); the FC chips were removed
  there, and the FC stays on every row.
- **One edit pin**: `ui/EditableText.tsx` shows changeable text with a ✎ pin and a dashed
  line, and edits in place (Enter saves, Esc cancels). `AutoValue` marks values that change
  by themselves with "⟳ auto" and no pin.
- **Compact lists**: `table.compact-rows` gives one line per point. Tool headers, device
  headers, tabs, toolbars and filters share rows. Rare actions move into menus; no function
  is removed.
- **Full view**: `ui/FullView.tsx` lets one big list fill the window under a slim dark bar
  (place, facts, the main start/run button, *Exit full view (Esc)*). The *Full view* button sits
  at the end of the tab row (`TabBar end`). It is used by Devices data points, the gateway,
  master and converter, SNMP simulate and read, and IED Explorer. On/off is remembered per list
  in the backend preferences (list `full-view`). The layer is z-index 45, so menus and help
  bubbles still open above it; Esc while typing does not leave it.
- **One line per row**: the shared `.type-cell` rule is scoped to `td.type-cell`, and the edit
  pin's empty state is `is-empty`, not the global `.empty` (which added 48 px padding). The
  gateway shows the path without the IED name and ends a long path in … (full path on hover).
- **Data classes (Go only)**: `GET /api/v1/ieds/{ied_id}/data-classes` returns
  `{"classes": {object ref: cdc}}`, read from the IED's stored SCL (`scl.DataClasses`,
  cached per IED and source). The frontend uses it to tell counters (BCR) and single/double
  commands (SPC/DPC) apart. `GET /ieds/<id>/signals` accepts Go-only `point_kind=<kind>` and
  `helpers=hide` with the same rules, and `kind_counts=1` adds `kind_counts` to the answer
  (`signalindex/point_kind.go`).
- **Shared colours instead of page colours**: `--warning-text`, `--error-line` and
  `.confirm-box` replace the hex values copied into the gateway, explorer, GOOSE and devices
  styles.

## Compatibility

No project data or IED model changes. Requests without the new parameters are answered
byte-identically, and the Python-parity golden test still passes. The frozen Python engine
answers 404 to the new route and ignores the new parameters, so classes stay unknown and the
type chips fall back to unfiltered lists there. Remembered filter choices for removed chip
groups are ignored (shared `restore`). The diagram drawing is unchanged.
