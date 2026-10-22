# 0169: SCADA master gets the most-used commands (Go engine only)

Date: 2026-10-22. Extends the SCADA master ([ADR-0061](../../decisions/ADR-0061-scada-roles-master-converter.md))
as a Go-only addition ([ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)):
new behaviour reaches users with the Go engine. The frozen Python backend is unchanged, and the
Python parity replay still passes (the new kinds are accepted in Go but never listed in the frozen
"not available" message, and the cyclic route is Go-only). No new ADR: this adds command types and
one route within the existing master design and native IPC, all backward compatible.

## What changed

- `native/master/iec104_command_objects.cpp/.hpp`: new `regulating` kind building C_RC_NA_1 / C_RC_TA_1
  (Lower = 1, Raise = 2); the QU qualifier is now threaded instead of hard-coded 0.
- `native/master/client_104.cpp`: the command path carries the qualifier and an Activate/Deactivate
  cause (COT 6/8). A deactivate sends one object and waits for deactivation confirmed (no termination).
- `native/master/iec104_decode.cpp`: C_RC_NA_1/C_RC_TA_1 type names and the "deactivation" cause (COT 8).
- `native/master/client_dnp3.cpp`: CROB rebuilt with latch on/off, pulse on/off, trip/close plus count
  and on/off pulse times; new `analog_output_int16` (g41v2).
- `engine/internal/api/toolsapi/master.go`, `master_command.go` (new): `commandKinds` kept as the frozen
  Python baseline (for the refusal message); `goOnlyKinds` adds `regulating` and `analog_output_int16`;
  the command builder is shared by the one-shot and cyclic routes; QU (0-31), deactivate, and CROB
  count/on-time/off-time are validated. New routes `GET/POST /tools/master/cyclic` and
  `POST /tools/master/cyclic/stop` (GET not a push topic, so parity is unaffected).
- `engine/internal/gateway/service/master_cyclic.go` (new), `master.go`, `master_connect.go`: a cyclic
  sender repeats one command on a 100-600000 ms timer, optionally stepping the address (+1) and toggling
  an on/off value each shot; every shot goes through `Master.Command` (so it is logged and warned);
  Disconnect stops it.
- `frontend/src/api/master.ts`, `copy/master.ts`, `points/pointKind.ts`,
  `features/tools/gateway/master/MasterCommandForm.tsx`, `MasterCyclic.tsx` (new), `master.css`: the
  command form adds the regulating step (Lower/Raise), QU, Activate/Deactivate, CROB pulse on/off with
  count and on/off times, and a "Repeat automatically" panel (interval, Address +1, Toggle, Start behind
  the same REAL-device warning, Stop). Command order keeps "most used on top".

## Evidence

- Go unit/sequencing: `master_cyclic_test.go` (address steps 100/101/102, value toggles 1/0/1; bad
  interval and not-connected refused; `toggledValue` cases); `go test ./internal/gateway/service/...` ok.
- Go end-to-end against ScadaBench's own 104 outstation (`studio_gateway`): `master_commands_extra_test.go`
  — QU > 31 refused (400); deactivate of a double gets "deactivation confirmed", no termination;
  `regulating` accepted and framed; cyclic refuses 10 ms, runs at 200 ms, Stop ends it. Run with
  `STUDIO_NATIVE_BIN` pointing at the rebuilt `studio_master.exe`.
- Parity: `TestRoutesAgainstFlask` passes (frozen Python routes unchanged).
- Frontend: 373/373 node tests; `npx tsc --noEmit` clean. `check_file_lengths.py` PASS.
- Native: `studio_master.exe` rebuilt (Visual Studio 16 2019, x64, Release).

## Limitations

- Our master and ScadaBench's outstation share lib60870, so the 104 checks are not independent. The
  outstation does not implement C_RC, so the regulating step was only confirmed to leave as a valid
  C_RC frame, not accepted end to end; the same applies to QU semantics. No third-party SCADA/RTU was
  available.
- DNP3 "Direct Operate - No Ack" is not offered: opendnp3 3.1.2's command processor exposes only
  select-and-operate and direct-operate.
- DNP3 CROB timing and g41v2 were not exercised against a live outstation here (no DNP3 device wired in
  this run); validation and framing are covered.
- `internal/gateway` Go tests are blocked by the local antivirus ("Access is denied" on fork/exec); run
  with `-trimpath` as `verify.py` does.
