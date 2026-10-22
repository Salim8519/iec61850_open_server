# 0167: Typed IEC 61850 commands: INC, ENC, APC, BAC, BSC, ISC with Select, Operate and Cancel

Date: 2026-10-22. Decision [ADR-0104](../../decisions/ADR-0104-typed-iec61850-commands.md). Go engine and
native workers; the frozen Python engine and its goldens are unchanged.

## What changed

- **SCL reader** (`engine/internal/scl/controls_extended.go`, `controls_extended_read.go`): typed
  controls read from the stored SCL with the builder's rules; `builder.go` only gained the hook and a
  field that `Import` never emits. Shape per entry: `{ref, status, cdc, ctl_val_kind, ctl_val_ref,
  value_kind, model, sbo_timeout_ms, min, max, step, labels, trans_ind}`.
- **Simulated IEDs** (`native/worker/controls/typed_*`): start key `extended_controls`, capability
  `typed_controls`, op `control_checks`; feedback per class, AddCause 7/9/10/11/26, test flag,
  originator in `control_requested`/`control_applied`/`control_test`.
- **Client** (`native/control/typed_session.*`, `typed_value.cpp`): `typed_select`, `typed_operate`,
  `typed_cancel` (capability `typed_commands`); the select is kept per endpoint/ref until operate,
  cancel or sboTimeout. `canvas_command.cpp`, `canvas_mux.cpp` and `control_session.cpp` are untouched.
- **Go**: package `typedcmd` (values, options, plain-English outcomes), `runtime/typed_command.go`
  (`SelectTyped`, `OperateTyped`, `CancelTyped`, the contract other workstreams build on),
  `typed_checks.go`, `typed_selects.go`, routes in `runtimeapi/typed_routes.go`, Explorer
  `tools/explorer/command.go` + `POST /tools/explorer/command`, `tools/control/typed.go`,
  `native/control_typed.go`. `native.IedRequired` now includes `typed_controls`.
- **Frontend**: IED Explorer command panel uses the typed path: Close/Open or On/Off, list pick for
  `Mod`, number box for INC/ISC/APC (limits shown), ▲ Higher / ▼ Lower for BSC/BAC, Select → countdown →
  Operate / Cancel selection for SBO models, folded "Command options" (test, interlock check,
  synchrocheck, sent from, sender name).
- Sample IED `studio/tests/fixtures/typed_controls/SampleTyped.scd` (every class in every model).

## Evidence

Recorded so far (worktree build, `cmake --build studio/native/build --config Release --target
studio_ied studio_control`):

- `go test ./internal/scl/ -run "Extended|Corpus"`: pass (24 typed controls in the sample, exact
  shape; Python corpus goldens still byte-identical).
- `go test ./internal/tools/control/ -run Typed`: 4/4 pass on real `studio_ied` + `studio_control`:
  6 classes × 4 control models (select where needed, operate, positive CommandTermination for
  enhanced, feedback read back, originator `rig-test`/orCat 2 in `control_applied`); limits (INC 11 →
  AddCause 26, BSC stops at 63, ISC 18 refused, ENC 9 refused, wrong kinds refused before sending);
  cancel then operate → "Press Select first."; select timeout 3 s → `select_expired`; test flag
  acknowledged without change; interlock block → AddCause 10, synchro block → 11, blocked refuses.
- `go test ./internal/runtime/ -run Typed`: 2/2 pass (coordinator API on a running sample IED,
  "Select expired after 3 s. Press Select again.", plain-English refusals).
- `go test ./internal/api/runtimeapi/`: pass (stopped Flask goldens unchanged; typed routes; typed
  operate of a breaker over HTTP). `./internal/typedcmd`, `./internal/tools/explorer -run Command`: pass.
- Full `CGO_ENABLED=0 go test ./...` (before the frontend step): 61 packages ok. Failures not caused by
  this change: `internal/gateway` build "Access is denied" (antivirus); `internal/native` worker tests
  bind 127.63.0.x:21011-21023, inside this PC's Windows excluded TCP range 20924-21023; `launcher`,
  `licensing/hostkey`, `netcfg` (PowerShell 20 s timeout) do not depend on any changed package.

- Frontend: `node --test --experimental-strip-types tests/frontend/*.test.ts` 378/378; `npx tsc
  --noEmit` clean; `check_file_lengths.py` PASS. `verify.py` with `GOTMPDIR` inside the worktree: all 7 checks PASS, including the
  whole Go engine suite (the first run found the
  five new routes missing from the Go-only allow list of `app/routes_test.go`; fixed with
  `runtimeapi.GoOnlyRoutes` and `explorer.CommandRoute`).

### Independent clients (libiec61850 v1.6.1, examples built with its own CMake, `BUILD_EXAMPLES=ON`)

- **A. libiec61850 `ControlObjectClient` → `studio_ied` with SampleTyped.** A 60-line evidence driver
  calls the library's own `ControlObjectClient_create/select/selectWithValue/operate/cancel` (no
  ScadaBench client code; kept out of the repository). One fresh worker per run (see finding).
  Every class in every model was accepted and read back over MMS: `Mod.stVal` 3, `ISCSO1.stVal` 7,
  `AnOut1.mxVal.f` 42.5, `TapChg.valWTr` `{6,false}` (5 → higher), `TapSet.valWTr.posVal` 9,
  `BndSet.mxVal.f` 2.5; 24/24, with 12 CommandTermination+ for the enhanced models. Refusals over the
  wire: INC 11 → addCause 26; Select, Cancel, Operate → 18 (object not selected); Select, wait 3.5 s
  (sboTimeout 3 s), Operate → 18; test flag → acknowledged, `stVal` stays 0; interlock block +
  interlock check → 10, the same block without check → operated (stVal 2); synchro block + synchro
  check (direct enhanced APC) → 11.
- **B. Unmodified `client_example_control` → `studio_ied` with `simpleIO_control_tests.cid`.** SPCSO1
  direct operate ok and stVal 1; then "failed to select SPCSO2" and SPCSO3/4/9 "not found". The
  baseline worker of the main checkout gives exactly the same output, so this is not caused by this
  change (finding below).
- **C. ScadaBench `studio_control` typed steps → unmodified `server_example_control`.** `hello` lists
  `typed_commands`; SPCSO1 direct-normal operate ok; SPCSO2 sbo-normal select (expires_ms 2000) then
  operate ok; SPCSO3 direct-enhanced operate ok, terminated true; SPCSO4 sbo-enhanced select then
  operate ok, terminated true (its sboTimeout is not readable, so the client counts the default 30 s);
  SPCSO2 and SPCSO4 select then cancel → canceled true; operate after cancel with `require_select` →
  "Press Select first."

### Findings (pre-existing, not fixed here)

- `studio_ied` refuses new associations (MMS error 5, connection rejected) after six clients that each
  connected, sent Oper and disconnected; reads alone do not trigger it. Same with the baseline worker
  (`studio/native/build/Release` of the main checkout). Studio's own clients keep one association,
  so the app is not affected, but a SCADA that reconnects per command would be.
- With the simpleIO model the unmodified example client fails at SPCSO2's select and the association
  is unusable afterwards; same with the baseline worker. Needs its own diagnosis.

## Limitations

- APC/BAC use `AnalogueValue.f` only; Edition 1 `setMag` and `operTm` are not supported.
- Over MMS an enumeration has no names, so Explorer names values only for `Mod`; other ENC objects
  get a number box. Project IEDs carry their labels through `GET /controls/typed`.
- The simulated tap changer reaches each position at once (`transInd` never true).
- `orCat` 0 ("not supported") cannot be sent: 0 means "not set" and sends 3 (remote).
