# 0183 GOOSE IED components: lines from IED to IED, GOOSE sent from the SLD

Date 2026-10-23. Go engine + frontend + docs. Decision [ADR-0112](../../decisions/ADR-0112-goose-ied-components.md),
spec [signal layers](../../specs/signal-layers.md) LAYER-10, 15, 22, 25, 29-38 and [GOOSE](../../specs/goose.md)
GOOSE-28. The frozen Python engine is unchanged and refuses the new kind and routes.

## Problem (owner request)

GOOSE lines ended on small "G" marks in the corner of whichever equipment was bound to an IED, or on
name tags left of the drawing, so a breaker could look like it received a trip. Nothing on the SLD
could send GOOSE: the user had to find the data point and guess which values were in the message.

## Change

- **Go** (`engine/internal/goosemsg/`, new, pure): FC-aware message catalog from the stored dataset
  (`fcda_refs` + member FC, `XCBR1.Pos [ST]` → `Pos.stVal/q/t`, never `Pos.ctlModel [CF]`), cached
  per stored IED record; `Link` check and plan (open/closed → Dbpos 1/2, a switch's BOOLEAN
  closed=true, contacts as two BOOLEANs in one plan, on/off, numbers; whole numbers for INT);
  plain-word refusals (real, skipped, GOOSE off, not running); messages and members answers;
  `Propose` for *Add all GOOSE IEDs*.
- **Contract**: Go-only kind `IED` (reuses `ied_id`; one per IED; no state/command binding; pasted
  copies lose the IED); omittable `goose_link` on every kind but JUNCTION/INFEED/POINT_PANEL/IED;
  removing an IED removes its component and every link to it. Unknown-kind words and golden
  vectors unchanged.
- **Runtime** (`runtime/goose_send.go`): `POST /equipment/{node}/goose` plans the writes and sends
  them as ONE `apply` batch with source `goose_send` (pauses a generator on those points), logging
  `goose_send_requested` → `goose_send_applied` / `goose_send_failed` with one correlation id.
- **Routes** (Go-only lists updated, `app/routes_test.go` checks them): `GET /ieds/{id}/goose/messages`,
  `GET /goose/members`, `GET /diagram/goose-ieds`, `POST /equipment/{node}/goose`.
- **Frontend**: IED kind everywhere (Add › *IEDs and GOOSE*, drop, registry, symbol, setup items,
  tidy excludes it, clipboard); `IedCard` (words; live problems and readers through a keyed store
  so traffic re-renders only that card); IED sheet (choose IED, messages with values in dataset
  order, On/Off, Open/Closed or number sends, watch only for real, received messages in words);
  equipment *GOOSE* section (picker, Send buttons while running); point panel "GOOSE" tag;
  *Add all GOOSE IEDs* and "N IEDs not on the drawing ▾" (one edit, one Undo, drawing saved first).
- **Overlay**: "G" marks, name tags and `IedTag.tsx` removed; origins only from trigger events
  (`gooseTriggers.ts`); `planChanges` routes origin → sender component (`routeVia`) → receivers,
  never onto equipment; `MissingChip`, `FlyLines`, `IedFlash`; expected pulses use Go's exact
  leaves (prefix rule kept for an older engine); boxes use React Flow's measured sizes
  (`useMeasuredBoxes`: the old 80×60 fallback routed a line through a 200 px card); a panel row's
  position change reads "Line disconnector Open", not "… 2".

## Evidence

- GOOSE stNum once (app's own listener `studio_live`, separate process): demo drawing on the public
  `scd/model_substation.scd`, breaker Q0 linked to FEED1 `Feed1SwiPosCtl` XCBR1: *Send Open* →
  run log `goose_send_requested` (id 185) → `signal` source `goose_send` → `goose_send_applied`
  (same correlation id) → `goose_status` st_num 2, sq_num 0; `/goose/links` BUS1 and BUS2 1 → 2.
- Independent of Go: `tests/protocol/test_goose_send_batch.py` starts studio_ied + studio_goose +
  studio_live, sends XCBR1 and XSWI2 open in one `update_batch`: stNum rises exactly once
  (2.5 s of frames after it), both values in the first frame (2 passed with test_goose_inside).
- Browser (worktree build, Go engine, port 8881): Add all GOOSE IEDs placed 6 components in one row
  above the SCD drawing, one Undo; on the demo drawing sends from the breaker link, from the FEED1
  component and from a point panel row each drew origin → FEED1 → BUS1 with "✓ got it", chip
  "… · 2 of 2 IEDs got it" and "also BUS2 · not on drawing"; Add BUS2 from the toolbar list went to
  the next slot, Undo removed it; GOOSE picker saved `goose_link` on Q1; no "G" marks; no failed API
  request (one 401 on the first page load before the session).
- Screenshots (1366×768 and 1920×1000 viewports, shown scaled to the pane):
  `%TEMP%/claude/…/scratchpad/goose-ied-shots/` (see the ledger entry).
- Go: goosemsg 11 tests + 2 benchmarks (catalog build ≈12.7 µs / 77 allocs, reuse ≈30 ns / 0 allocs),
  project, runtime, runtimeapi, liveapi, projectapi, app route lists; full `go test ./...` and
  verify.py results in the ledger. Frontend 473/473 (new `gooseIedComponents.test.ts`), tsc clean.

## Limitations

- "Got it" is the app's own listener seeing the new stNum, not each receiver's logic.
- Only BOOLEAN, Dbpos and numbers are sendable; quality and time leaves are listed, not sent.
- The model substation has no BOOLEAN pair, so *open and closed as two values* was checked in Go
  and frontend tests, not in the browser. *Who is reading* on the card was checked in tests only.
- At 1366×768 the SLD toolbar wraps to a second row when "N IEDs not on the drawing" shows (no text
  is cut; before, it wrapped only with a sheet open); the GOOSE colour stays the shared accent. Python engine refusal of the kind is by its unchanged validator, not re-run here.
