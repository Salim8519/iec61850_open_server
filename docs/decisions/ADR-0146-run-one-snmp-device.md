# ADR-0146: Run or stop one SNMP device

Accepted: 2026-11-22. Owner, 2026-11-22, for 0.1.36: "add in each snmp devices a button to run one
snmp devices instead of all? currently there is not button". Change
[0234](../changes/2026/0234-run-one-snmp-device.md).

## Problem

SNMP devices could only start and stop all together ("Start devices" / "Stop devices"). The worker's
`agents_start` replaces the whole running set, so starting one device through it would restart
every other device (new engine boots, a restart a SCADA manager can see).

## Decision

1. **Worker.** `studio_snmp` gets `agent_start {device}` and `agent_stop {device: id}`
   (`agent.Engine.StartOne` / `StopOne`): one device starts, restarts or stops; the others are
   untouched. `agents_start` / `agents_stop` are unchanged.
2. **Engine.** `POST /tools/snmp/devices/{id}/start` and `/stop` (`Service.StartDevice` /
   `StopDevice`): the same device checks, lab-card placement and status as Start. Stopping the last
   running device is Stop (auto start off, the card released). Starting one device does not change
   auto start.
3. **Screen.** Each device card has a Run / Stop button next to its views; the page's Start devices
   and Stop devices stay.

## Compatibility

Projects and the SNMP section are unchanged. The worker and engine ship together, so an older
`studio_snmp` never meets the new routes in a release.
