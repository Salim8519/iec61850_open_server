# 0184 Smart point picker: type chips, objects, suggestions and one complete kind table

Date 2026-10-23. Go engine + frontend. Decision [ADR-0113](../../decisions/ADR-0113-smart-point-picker.md),
spec [point-picker](../../specs/point-picker.md). No protocol code changed (only how points are listed
and classified), so no MMS client check applies.

## Problem (owner request)

With a screenshot of the CB sheet ("Position comes from → Two points: open and closed contacts", points
like `System/ConInGGIO1.Ind106.stVal`): finding a point in the SLD pickers is still a hassle. The "All
types" dropdown was easy to miss, offered Double point for an open contact, had no counts, and a type
alone cannot find Ind106 among hundreds of GGIO inputs. The owner approved a drawing and asked that the
filter cover all IEC 61850 standard points and follow the kind colouring.

## Change

- Go `signalindex`: complete kind table (`point_kind.go`, no regular expressions); kinds cached per index
  and data-class version (`kind_cache.go`); logical nodes built on first use (`logical_nodes.go`); one
  pass for matches, kind counts and node counts (`scan.go`); suggestions (`near.go`); new Go-only
  parameters `annotate`, `ln`, `ln_counts`, `near` (`query.go`, `search.go`); the route loads data
  classes for them and overlays live values on `suggested` too (`liveapi/signal_routes.go`).
- Shared vectors `studio/tests/fixtures/point_kinds/iec61850.json` (786 cases) read by a Go test and a
  frontend test; `points/pointKind.ts` mirrors the Go rules and `kindOfDataClass` covers every class.
- Picker split (`components/signalPicker/`): `preferredKind`, `pickerAnswer`, `useSignalSearch`,
  `KindChips`, `ObjectSelect`, `HelperToggle`, `SuggestionBlock`, `SignalPickRow`; words in
  `copy/signalPicker.ts`; `signalQuery.ts` (`smart`, `helpers`, `ln`, `near`); `pickerMemory.ts` keeps
  type, object and the quality/time switch. Callers pass the component's name and slot: state mapping
  and contacts, measurement list, relay rows, tap changer fields. The point panel adder gets the object
  select on the search row. `ControlMapping.tsx` is unchanged (it lists SPC/DPC controls, not points).

## Decisions where the brief left room

- Final kind table: see ADR-0113. SV and BL win over the helper rule (`subQ` is CF, not AT, so it is
  not hidden as a helper); a counter class needs a number (SEC `sev` is IS); a status/measurand class
  never overrides the attribute's type; unknown FCs and `Struct` stay UN; INT64U/INT128U stay numbers
  as before; settings and description classes refine ST/MX/OR attributes too.
- `ln_counts` keys come out sorted like every Flask-shaped answer; the browser lists them in natural
  order (GGIO2 before GGIO10) and drops the logical device when there is only one.
- Suggestions add `suggested_by` so a row can say "in its description" or "in its name"; words are
  letters and digits (an underscore separates words).
- The active chip keeps its chip at 0 and the list says "No single points (on/off) match. Pick another
  type above or All types." instead of switching type by itself.
- "Show quality and time" appears only where the slot can list helpers (no basic-type limit).
- Frontend tests are `node --test` (the repository's runner), not vitest.

## Evidence

- Go before the change: `signalindex` and `liveapi` tests pass. After: same packages pass, including
  the Python-parity golden test `TestSignalsMatchPythonRoute`; new tests for the table, `annotate`, `ln`
  and `ln_counts`, `near`, unchanged answers and the kind cache. Full `go test -trimpath -p 2 ./...`
  (CGO_ENABLED=0, STUDIO_PYTHON = the main checkout's venv): 72 packages ok, 0 failed (gateway,
  modelcache and launcher passed this time). `verify.py`: 7/7 PASS. Two earlier runs failed for the
  environment, not the change: the worktree had no `studio/.venv` (no Flask for the Python oracles; a
  junction to the main checkout's venv fixed it), then `licensing` `TestScenariosMatchPython` saw a
  short-lived `.tmp` file once (passed alone twice and in the other runs; the package is unchanged).
- Benchmarks, 90 000 signals, 8 interleaved runs each (median): Search 3.78 → 3.69 ms (1242 allocs both),
  SearchAndEncode 4.29 → 4.40 ms (noise, 1362 allocs both), Build 91.4 → 93.6 ms (270 529 allocs and
  38.9 MB both), type filter with counts (`point_kind=SP&helpers=hide&kind_counts=1`) 87.8 → 2.90 ms and
  8.3 → 4.2 MB per request; new full picker request 6.3 ms, 5.5 MB.
- Frontend 469/469 (460 before), `tsc --noEmit` clean.
- Browser (Go server built from this branch, port 8882, scratch data, public `scd/model_substation.scd`
  and `scd/open_substation.scd`, never Run): CB open contact on FEED1: only "Single point 28" + "All
  types 28", reason line, objects "CILO1 (2) … XSWI4 (3)", XCBR1 gives 3 rows; one position point: Double
  point preselected (8) beside Single point (28), "Suggested for XCBR1" lists `LD1/XCBR1.Pos.stVal`
  "Mentions XCBR1 in its name", Choose marks it Chosen in both places; search "3" gives 7 of all types;
  measurement list MMXU1: Measured value preselected (66), suggestions from MMXU1; relay rows: All types
  139 with 7 kinds in two columns, "Show quality and time" adds Quality / time 88; tap position: Integer
  status preselected; point panel adder: object select beside the search. No console errors except the
  expected 401 of a tab opened before its session. No clipped picker element at 1366×768 (fit 0.95) or
  1920×1000. Description suggestions over HTTP: `near=enumerated` → `XSWI2.SwTyp.stVal` "Enumerated
  status", `suggested_by: description`.

## Limitations

- The public samples have no description that names an SLD component, so the description suggestion was
  shown over HTTP and in Go tests, not in a screenshot.
- Search stays one substring (`q`), so "106" and "Ind106" work but "GGIO1 106" does not.
- Not checked in the packaged EXE (not built, by rule) or with the frozen Python engine in the browser
  (the fallback is covered by tests).
