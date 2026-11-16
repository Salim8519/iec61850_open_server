# ADR-0138: Real IEDs read only until "Allow commands", device control check

Accepted: 2026-11-11. Owner decisions in this session:
- "Monitor real device" must mean what it says: reading only, unless the user turns
  commands on for that IED;
- the switch lives per IED in Devices; IED Explorer is not touched;
- the gateway keeps its own "Allow SCADA commands to real IEDs" switch, and a SCADA
  command reaches a real IED only when both switches are on;
- the user makes sure the IED answers on the network (ping); ScadaBench adds no PC
  address and scans nothing for a real IED;
- on connect, compare the device's own control settings with the CID, and test command
  reading against every vendor file available.

## Context

ADR-0057 let the gateway pass SCADA commands to real IEDs, on by default. The role name
"Monitor real device" promised read-only, so a user watching a live bay could still trip a
breaker through the gateway without knowing it. The native control client already uses the
device's own control model at send time, but nothing showed the user when the device and
the CID disagree, or that the device is in Local.

## Decision

- **`live.allow_commands`** (Boolean, default `false`) belongs to each IED's real-device
  settings (`netset.ValidateLive`). Only a Boolean is accepted. It is stored only while it is
  `true`, so every existing project and settings answer stays byte-identical, and a missing
  key means off.
- **Gateway.** `Refusal` refuses a real IED's command while its `allow_commands` is off, with
  a sentence naming the IED and the switch. `MonitorOnly` marks those points monitor-only,
  so the native worker already answers negatively (104 negative ACT_CON, DNP3
  NOT_AUTHORIZED, Modbus exception). The gateway switch keeps working as before: both must
  be on.
- **Send from Devices** (owner, same day: "make the ied has its own allow switch to send
  commands"). While a real IED's Allow commands is on and its live session is connected,
  Devices › Data points shows Close…/Open… (DPC) or On…/Off… (SPC) on each command of the CID.
  Each opens a confirmation naming the IED, the object and that it is a REAL device; only
  "Yes, send" calls the Go-only `POST /live/{id}/operate` with `confirm: true`. The service
  (`internal/realcmd`) refuses an unconfirmed command, a non-real IED, the switch off, a command
  not in the CID, a non on/off value and a session that is not live, each in words. It sends
  through the shared `studio_control` client with the device's own control model (originator
  station), then reads the feedback point read only for up to 3 s. Events
  `real_command_requested` and `real_command_applied`/`real_command_failed` carry one
  correlation id and keep request, applied (device answer) and observed (feedback) apart. The
  gateway switch does not apply here. Typed commands (steps, set points) are not offered yet.
- **Unchanged.** IED Explorer (connects by typed address, its own confirmation), SLD and
  Test refusal of real IEDs (LIVE-04), `studio_live` read-only (LIVE-01).
- **Device control check** (`acquisition/control_check.go`). After the model comparison, the
  live session reads, read-only, `ctlModel` and `sboTimeout` (FC CF) of every control
  object in the CID and `Loc.stVal` (FC ST) of each logical device's LLN0 and of each
  control's own logical node where the CID has one. Each control is classed `match`,
  `differs` or `unreadable`; an unreadable value is reported, never replaced by the CID
  value. Local positions are listed. The result is `controls` in the session status, read
  again with Refresh health. Devices shows it under the connection.
- **Vendor command reading.** A Go test walks every public fixture plus the local corpus
  named in `STUDIO_SCL_CORPUS` (`vendor_cids/`, `test_cids/`; never committed) and checks
  that every imported control has a valid model (1 to 4), a feedback reference inside the
  model and a non-negative select timeout, and logs counts per manufacturer. Synthetic
  fixtures cover ctlModel set on the instance (`DOI`) and given as a number.

## Compatibility

- Projects keep their format number and their bytes; a project without `allow_commands`
  reads as off. A project that relied on gateway pass-through to a real IED must now turn
  the IED's switch on once. This is the intended change.
- Python is frozen (ADR-0076): the switch exists in the Go engine only, as the product runs.
- No new dependency and no native change. The only MMS write is the confirmed command through
  `studio_control`; `studio_live` stays read only.

## Evidence and limits

Tests are listed in change 0218. No third-party IED was available: the device check is
proven against `studio_ied.exe` as a stand-in device and unit tests only. Vendor support is
claimed only for reading commands from SCL files, not for operating those devices.
