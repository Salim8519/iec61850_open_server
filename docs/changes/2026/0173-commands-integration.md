# 0173 Commands everywhere: integration of 0165-0172

Branch `integration-0173` (from `release-0120`, 0.1.21). It merges, in this order: point panel
0165 + GOOSE smart lines 0166 (with compact stopped rows), master most-used commands 0169,
gateway Add points 0168, SLD tap-changer transformer 0172, typed IEC 61850 commands 0167,
SLD select before operate 0171 and gateway step / set point commands 0170. Nothing committed.

## Merge decisions

- ADR-0104's `runtime/typed_command.go` (agent A) replaces the stand-in adapters of 0170, 0171
  and 0172. 0171's `typedControl`/`controlModelName` helpers stay; 0172's duplicate
  `typedControl`, which read `extended_controls` from the IED payload (ADR-0104 never puts it
  there), was removed so the tap changer finds BSC/ISC controls in the stored SCL.
- ADR-0104's typed control lookup reads through 0171's `extendedControlsOf` hook (the same SCL
  cache), so runtime tests can inject typed controls.
- `CommandOptions` is ADR-0104's `typedcmd.Options` alias; 0171's method on it became the
  function `optionsObject`.
- One `runtimeapi.GoOnlyRoutes` lists ADR-0104's and ADR-0108's routes.
- Gateway spec rules of 0170 renumbered GW-15/GW-16 (0168 took GW-12..14).
- Index rows of every branch kept (append-only), ledger lines of 0167-0172 re-appended in
  number order with `history_log.py`; 0167's route text, rewritten by Git Bash on its branch,
  is written correctly in the re-appended line, so its separate correction note is not needed.

## Fixes found while merging

- **Typed gateway links were always refused.** 0170 checked a step / set point link against the
  IED's `controls`, which hold only SPC/DPC; ADR-0104 keeps BSC/ISC/APC/INC in the stored SCL.
  New `gateway.ExtendedControls` hook (wired by the app to `scl.CachedExtendedControls`) and
  `typedControlReason` look in both. 0170's tests had put BSC/APC into `controls`, which hid it.
- **Add points did not offer 0170's kinds.** The type catalog (`addpoints/catalog.go`) lists set
  point, step and tap position per protocol (set point is "Most used" on IEC 104); candidates
  include typed controls (BSC/ISC for a step, APC/INC for a set point; ENC/BAC refused in
  words); their status pairs as step position / float / integer; kind checks use
  `gateway.KnownKind` instead of the six Python kinds only.
- Tests that pinned a stand-in's "not available yet" answer now check the real typed path; the
  fake control worker answers `typed_select` / `typed_operate` / `typed_cancel`.
- **Typed controls never refreshed the drawing.** A BSC raise reached the IED (posVal 5 -> 6) but the
  SLD kept 5: the runtime's control feedback refused every non-Boolean ctlVal ("feedback failed").
  Typed values now skip only the switch-symbol mapping and still publish their status
  (`runtime/feedback.go`, `feedback_typed_test.go` fails without the fix).
- **Tap changer refused a real BSC on save.** 0172's project check read `extended_controls` from the
  IED payload; it now reads the stored SCL like 0171's panel check.
- **SCADA step / set point commands were refused at command time** ("no controllable object"):
  the gateway service looked the control up in `controls` only; `findControl` now also reads the
  typed list (`service/typed_refusal_test.go`). Found by the end-to-end run below.
- **Master cyclic routes** were missing from the app's Go-only route list (`MasterGoOnlyRoutes`).

## Owner decision: one click

Select before operate is protocol behaviour, not a second question for the user. The SLD switches
and point panel rows now send in one press (the engine selects and operates on the wire, as a
station HMI does). "Show Select before Operate step by step" under Command options (Go-only
preference `sld_sbo_steps`, absent from the default document so Python parity holds) brings back
the countdown with Operate / Cancel for testing. The IED Explorer stays step by step.

## Evidence

- Go build and vet clean; `go test` runtime, project, api, app, typedcmd, scl, gateway, addpoints,
  toolsapi pass (new: `gateway/typed_controls_test.go`, `addpoints/typed_candidates_test.go`).
- Frontend: tsc clean, node tests 422/422; file lengths PASS; specs PASS; history verify PASS.
- Native build of every program (VS 2019, 0 errors) and `studio_xsd`; full `go test ./...` with the new
  natives: 70/70 packages after fixes (netcfg live PowerShell test timed out once at 20 s under load,
  passed alone).
- Browser, 1366x768, own server on 8795, `SampleTyped.scd` (every class, every model):
  tap changer Raise x2 moved Tap 5 -> 7 live; panel APC setpoint 42 read back 42.00; SBO INC step by
  step selected on the wire and ran out after the IED's 3 s timeout; after the one-click change the
  same SBO INC sent 7 in one press and the IED (ctlModel 2) holds 7.
- Gateway end to end into `SampleTyped` (points planned by Add points, `studio_server` on 8795):
  - IEC 104, independent lib60870 master: C_RC 47 raise with select + execute (ACT_CON, ACT_TERM
    positive) moved posVal 5 -> 6; C_SE 50 55.5 with select + execute -> APC mxVal 55.5; C_SE 50 4 ->
    INC stVal 4. Before the `findControl` fix every one of them got a negative ACT_CON.
  - DNP3, independent opendnp3 master: CROB close (select-before-operate) = raise 6 -> 7, read back
    as analog input 0; g41 var 3 33.3 (SBO) -> 33.3; CROB trip direct = lower 7 -> 6.
  - Modbus TCP, raw-socket client: FC16 float32 77.25 -> APC 77.25; FC16 9 -> INC 9; FC6 into a
    two-register float point refused with exception 3.

## Limitations

- Panel value rows accept only measured (MX) values: whole-number status such as an INC stVal or a tap
  position cannot be shown in a panel yet.
- Enter in a panel number box did not open the confirmation in one browser run; the Send button did.
- ENC and BAC commands have no gateway kind yet (IED Explorer, point panel and SLD send them).
- The master Add points work (0174) is on its own branch and not in this merge yet.
