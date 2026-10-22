# 0172: SLD tap-changer transformer (RTCC)

Date: 2026-10-22. Owner request: a power transformer with an on-load tap changer on the SLD: Raise /
Lower, the tap position, Auto/Manual and a simulated voltage regulator, all logic in Go.
Decision: [ADR-0109](../../decisions/ADR-0109-sld-tap-changer-transformer.md). Built on the point
panel and GOOSE smart lines (0165, 0166, not yet merged), next to change 0167 (typed controls,
ADR-0104) and change 0171 (SLD select-before-operate), both built in parallel.

## Change

- New diagram kind `TAP_TRANSFORMER` (Add > Power in and drawing > Tap-changer transformer). The
  retired `TRANSFORMER` (ADR-0033) and the drawing-only infeed transformer (ADR-0041) were checked
  first: the first is still deleted by the format 12 migration, the second has one terminal and no
  data, so a new kind was cheaper and safer than extending either. Two-winding only.
- Canvas card: HV and LV terminals, "Tap 9 / 1–17", an Auto/Manual badge (icon, word and border:
  solid for Auto, dashed for Manual), and while running the LV voltage, the regulator state
  ("Out of band · waiting · 12 s") and ▲ Raise / ▼ Lower. Disabled buttons say why.
- Sheet: Operate section (Raise, Lower, Auto, Manual), then plain-English sections with hints: where
  the tap position comes from (whole-number ST: BSC `valWTr.posVal`, INS/INC/ISC `stVal`), where
  raise and lower go (one BSC/ISC step control, or two SPC commands), the tap range, optional
  Auto/Manual status and command, optional LV voltage, and the simulated voltage regulator (target,
  band, delay, step %, rated HV/LV, source voltage). Signals use the shared search picker.
- Go (all logic): contract `project/contract/tap_changer.go`; cross-checks
  `project/validation_tap_changer.go`, `validation_tap_avr.go`; polled signals of real IEDs
  `acquisition/watched_tap_changer.go`; pure domain `runtime/tapchanger/` (ratio and LV voltage,
  regulator decisions, raise/lower routing); runtime `runtime/tap_changer.go` (node, live rows,
  operate entry), `tap_changer_route.go` (BSC/ISC via `OperateTyped`, SPC pair via the canvas
  command path, tap feedback), `tap_changer_mode.go` (Auto/Manual), `tap_changer_avr.go` (the
  regulator every 0.1 s, events); energisation through the transformer
  `runtime/plant/energize_transformer.go`.
- `runtime/typed_command.go`: exactly the typed command API of change 0167 with temporary adapters
  ("Step commands ... are not available yet"), marked `// Replaced by ADR-0104 at merge`.
- Frontend: `equipment/tapChanger.ts` (words only), `tapCommands.ts` (`sendTapCommand`, the one seam
  where change 0171's `useSelectOperate` plugs in), `tapContext.ts`, `TapTransformerBody.tsx`,
  `tap-transformer.css`, `copy/tapChanger.ts`, `features/diagram/tap/` (sheet, fields, operate,
  command hook). Optional `extended_controls` on the IED type.

Existing files changed (one addition each):

| File | Why |
| --- | --- |
| `project/contract/diagram.go`, `point_panel.go` | accept the kind; omittable `tap_changer` field |
| `project/validation.go`, `removal.go` | call the tap changer checks, settle defaults, IED removal |
| `acquisition/watched.go` | poll the transformer's signals on real IEDs |
| `runtime/coordinator.go`, `lifecycle.go`, `cadence.go` | hold, reset and advance the regulators |
| `runtime/plant_states.go` | live rows and regulator states in `GET /equipment/states` |
| `runtime/equipment_control.go` | dispatch `raise`/`lower`/`auto`/`manual` to the transformer |
| `runtime/sample_writer.go` | write source `tap_changer` pauses a generator on the LV voltage point |
| `runtime/plant/energize.go` | the transformer is a power item; feed through after the infeeds |
| `api/types.ts` | kind, `tap_changer`, optional `extended_controls` |
| `equipment/registry.ts`, `kindInfo.ts`, `symbols/EquipmentSymbol.tsx`, `EquipmentNode.tsx`, `tidy/tidy.ts` | name, symbol, card, tidy size |
| `features/diagram/AddMenu.tsx`, `Canvas.tsx`, `DiagramPage.tsx`, `EquipmentSheet.tsx` | add tile, defaults, command context, sheet sections |
| `docs/design/layout.md`, `interactions.md`, `reference/API.md`, `FILE-MAP.md` | the card, the buttons, the actions and events |

## Demo station

The demo and sample SCL (`scd/open_substation.scd`, `scd/model_substation.scd`, the marketing demo)
have no tap changer (no YLTC/ATCC, no BSC/ISC), so the demo was not changed. To try it on any
simulated IED: add a Tap-changer transformer, choose a writable whole-number status (ST) as the tap
position, "Two single commands" with two SPC controls, and tick "Simulate the voltage regulator";
set the source voltage to e.g. 140 kV and press Auto: the regulator lowers the tap until the LV
voltage is within the band. A real tap changer IED with a BSC uses "One step control" after change
0167 is merged.

## Evidence

- Go (`CGO_ENABLED=0`): `runtime/tapchanger` 13 tests (tap -> LV voltage incl. a 140 kV source,
  limits, in band, above band lowers after the delay, below band raises, back in band cancels, max/min
  limit logged once and never commands, Manual and unknown tap do nothing, zero delay, settings read;
  routing BSC higher/lower, ISC new tap and refusal without a known tap, SPC pair, limits, no command);
  `project` 4 tests (exact default bytes, other kinds drop the field, 21 refusals with plain words,
  BSC + ENC and SPC pair bindings valid, IED removal leaves a valid transformer); `runtime` 6 tests on
  the fake workers (SPC-pair raise/lower moves the simulated tap 9 -> 10 -> 9 with requested/applied
  events sharing a correlation id; raise at 17 refused; manual step in Auto refused; BSC step answers
  "not available yet" and logs a failure; the regulator lowers 9 -> 5 on a 140 kV source in four
  `step` decisions and writes ~11.1 kV to the LV point; a real IED is never commanded and the
  regulator logs `blocked: real_ied`); `runtime/plant` 3 energisation tests (LV side live, an
  earthed LV side does not earth HV, dead source). CPython energisation vectors still identical.
- Full `go test ./...`: all packages pass except `internal/gateway` and `internal/store/modelcache`
  (antivirus "Access is denied" on the test executables), `internal/launcher` (needs STUDIO_PYTHON;
  temp lock cleanup) and `licensing/hostkey` (rate-limit timing test); none is touched by this change.
- Frontend: `node --test` 401/401 (8 new in `tapChanger.test.ts`), `tsc --noEmit` clean, vite build;
  `check_file_lengths.py` PASS; `verify.py` PASS (lengths, architecture, docs, history, specs, GOOSE
  engine and Go engine tests).
- Browser (worktree `studio_server.exe` on 127.0.0.1:8793, own data folder, 1366x768, OpenSCD demo
  station `scd/open_substation.scd`; IED2_PTOC alone on port 20793 because other sessions held :102):
  Add > Tap-changer transformer gave the card "Awaiting tap" / Manual; the tap picker offered only
  the 9 whole-number ST signals; LPHD1.PhyHealth.stVal as tap position and GGIO1.SPCSO1/SPCSO2 as
  the raise/lower pair. Running: Raise on the card and in the sheet moved the tap 0 -> 1 -> 2
  ("Raise done · tap 2", LV 9.90 -> 10.04 kV); Lower was disabled at the lowest tap with the reason.
  Auto (3 s delay): the regulator raised 3 -> 8 by itself and stopped "Voltage in band" at 10.86 kV;
  Manual then Lower gave tap 7 (10.72 kV). The run log holds `avr_decision` (`mode_auto`,
  `out_of_band`, `step`, ...) with run_id, node_id, ied_id and correlation_id, and the
  `tap_command_*` events. No text was cut at 1366x768.
- Found in the browser and fixed: choosing "Two single commands" was refused until both commands
  were chosen, so the mode jumped back. An incomplete command now saves; the regulator and Raise /
  Lower stay off until it is complete (`tapCommandReady`, `commandReady`, `Route` "not complete").
  Also "Tap not mapped · 1–17" wrapped mid-range (now "Tap not mapped") and "Manual" was repeated.

## Limitations

- BSC/ISC step commands and ENC mode commands answer "not available yet" until ADR-0104 is merged.
- No select-before-operate on the SLD until change 0171; commands are a direct operate.
- Two-winding only; no parallel control, line drop compensation, tap-change time or transInd.
- The regulator's own Auto/Manual is per run (starts Manual); the source voltage is a setting.
- The frozen Python engine refuses a project that contains a tap-changer transformer.
