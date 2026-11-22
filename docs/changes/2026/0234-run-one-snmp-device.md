# 0234 Run or stop one SNMP device (0.1.36)

Date: 2026-11-22. Owner: "please can u add in each snmp devices a button to run one snmp devices
instead of all? currently there is not button". Decision: [ADR-0146](../../decisions/ADR-0146-run-one-snmp-device.md).
Layers: Go SNMP worker and service (`internal/snmp/agent`, `worker`, `service`), `api/snmpapi`,
frontend SNMP simulate page.

## What changed

- `snmp/agent/engine_one.go` (new): `StartOne`, `StopOne`; `snmp/worker/handlers.go`: `agent_start`,
  `agent_stop`.
- `snmp/service/device_run.go` (new): `StartDevice`, `StopDevice`; `api/snmpapi/devices.go`:
  `POST /devices/{id}/start` and `/stop`.
- Frontend: `snmpApi.startDevice` / `stopDevice`; `DevicePanel.tsx` Run / Stop button with a
  tooltip naming the device; copy in `copy/snmp.ts`.

## Evidence

- `TestRunOneDevice` with the real rebuilt `studio_snmp.exe`: Run A starts only A; Run B keeps A
  running; Stop A keeps B running; Stop B (the last) stops the engine. All `internal/snmp/...` and
  `api/snmpapi` tests pass. Frontend `tsc --noEmit` clean.

## Limitations

- Not yet clicked through in the packaged app; the release build of 0.1.36 follows.
