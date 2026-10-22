# 0170: Gateway tap positions, step commands, set points and command steps

Date: 2026-10-22. Decision: [ADR-0107](../../decisions/ADR-0107-gateway-step-setpoint-commands.md).
Spec: [gateway](../../specs/gateway.md) GW-15, GW-16 (renumbered at merge: 0168 took GW-12..14). Branch `gateway-kinds-0170` (from `release-0120`).

## What changed

- Go: three point kinds (`step_position`, `command_step`, `command_setpoint`) with validation
  (control class, signal type, protocol, format, word order), address spaces and widths, CSV
  `format,word_order` columns, own points, project storage. Routing table
  (`service/routing.go`) and executor (`service/typed_exec.go`): select, execute and cancel go
  to `SelectTyped`, `OperateTyped`, `CancelTyped`; a direct single/double execute keeps the old
  path; a single/double select falls back to ScadaBench's own confirmation while the typed API
  answers `not_available`. `runtime/typed_command.go` is the temporary adapter (replaced by
  ADR-0104 at merge), reached through `runtime/ports/gateway_typed.go`.
- Native `studio_gateway`: `typed_commands` capability and start flag; IEC 104 C_RC 47/60 and
  C_SE 48-50/61-63 decoding (`iec104_command_decode.cpp`), M_ST 5/32 tap positions, every step
  to the engine with `wire`; DNP3 CROB codes and functions and g41 var 1-3
  (`dnp3_commands.cpp`), tap position analog inputs, analog output status; Modbus FC6/FC16 set
  points (`modbus_writes.cpp`), tap position input register, FC3 read-back. Test masters gained
  step, set point, time tag and cancel steps (104) and CROB codes and analog outputs (DNP3).
- Frontend: names, type IDs and hints in every kind list (`copy/gatewayTypedKinds.ts`), badge
  kinds, Raise…/Lower… and a set point box in the command test row, own targets per command.

## Evidence

- Go: `CGO_ENABLED=0 go test ./...` with `GOTMPDIR` in the worktree: every package passes
  except `internal/launcher` `TestLeaseSharedWithPython` (needs `STUDIO_PYTHON`; TempDir lock
  clean-up), which this change does not touch. New tests: `gateway/typed_kinds_test.go` (5) and
  `service/routing_test.go` (5). The existing gateway wire tests pass with the new native
  program (`STUDIO_NATIVE_BIN` = this worktree's build).
- Native (pytest, independent lib60870/opendnp3 masters, raw Modbus):
  `test_gateway_typed_wire.py` 4/4, `test_gateway_wire.py` 6/6, `test_gateway_runtime.py`,
  `test_converter_runtime.py`, `test_master_wire.py` 7/7 (Python engine, unchanged channel).
- Frontend: node tests 377/377 (4 new; two kind-list expectations extended), `tsc --noEmit` clean.
- End to end on the composed Go server (port 18777, public `scd/open_substation.scd`, IEDs
  running, studio_test_iec104_master / studio_test_dnp3_master):
  - C_DC select + execute on `IED1_XCBR CSWI1.Pos`: 46/7, 46/7, 46/10; Commands tab "select
    confirmed by ScadaBench" (SelectTyped answered `not_available`), then "operated … is now
    closed (ON)" — the IED operated.
  - C_SE_NC 50 select + execute, C_SE_NB 49, C_SE_NA 48, cancel (COT 8 → 50/9), C_RC 47 select +
    execute on own points: all positive with ACT_TERM; the own value and tap position follow
    (spontaneous M_ME_TF 1.0 / 0.5, M_ST_TB 6); C_RC to IOA 50999 → 47/47 negative; C_SE_NB to a
    float set point → 49/44 negative, each with a plain Traffic line.
  - DNP3: CROB close SBO, trip direct, pulse_on SBO (step raise), g41v3 SBO and g41v2 direct all
    SUCCESS; tap position analog input 5 → 6.
  - A set point or step linked to a DPC is refused with "… is a DPC and cannot take a set point
    (it needs APC or INC)".

## Limitations

- The public demo SCD has no BSC/ISC/APC/INC controls, so a typed value to a real control was
  checked only with the fake typed port in Go tests; against the temporary adapter it answers
  "not available yet". With ADR-0104 merged this must be rerun on a model with a tap changer.
- No third-party SCADA; DNP3 direct operate without answer is decoded but not sent by any
  available master; M_ST transient bit always 0; no DNP3 g41 var 4; no Modbus step command.
- The set point format is edited through CSV or the API; the Add points dialog is rebuilt in a
  separate change. Not checked in the browser at 1366×768.
- The frozen Python engine cannot open a project that holds a new kind.
