# 0099: Standard commands only, and Operate reaches lab-mode IEDs

Date: 2026-10-18. Decision: [ADR-0069](../../decisions/ADR-0069-standard-commands-and-listening-endpoint.md)
(supersedes ADR-0018).

Owner report: with ScadaBench (0.1.10) in a VMware VM and GE Viewer + Concentrator on the host,
the Viewer said "Command Sent OK" but CB Y755 would not close, and ScadaBench's own Operate
button said "MMS control failed or command termination was not received".

## Diagnosis (VM and host evidence)

| Test | Wire / log | Result |
| --- | --- | --- |
| Standard profile, Viewer CLOSE (`case-4-standard-close.pcapng`) | 21:38:26.687 `SBOw` ctlVal=true → success, Pos stays 01; 21:38:35.656 `Oper` true → success, CmdTerm+; Pos → 10 | closed |
| CONCEN profile, same CLOSE (`case-5-concen-close.pcapng`) | 21:41:08.327 `SBOw` true, 21:41:10.558 `Oper` true, both success, CmdTerm+; Pos stays 01 | not closed |
| VM run logs, owner's case-1/case-2 | both ran `compatibility=concen_inverted_sbow` (`act=close` for a requested Open); CB-2 shared CB-1's control, so the profile could not be changed alone | explains "normal SBO" failing |
| ScadaBench Operate in the VM | port 102 listens only on 192.168.230.20/.41; no control event logged; probe at an unbound 127.x → `error 5` after 10.7 s, at 192.168.230.20 → ok in 0.33 s | loopback dialled |

The Viewer itself also blocks a direction unless the UR FlexLogic PROCEED output is true
(`Master/GGIO1.Ind008` CB close, `Ind012` CB open, DS/ES in `Ind015`-`Ind069`). In the project
the close output was a 1 s self-resetting push button; that is a project setting and a known
limitation, not part of this change.

## Changes

- Native worker: ctlVal applied as given, movement on `Oper` only, no originator check; the
  deferred-task thread used only by the profile is removed; capability `standard_controls`;
  a start request with `compatibility` is refused.
- Backend: `ControlBinding` is `{ied_id, ref}`; project format 15; `_standard_commands`
  migration drops the profile for every version with one notice per CONCEN binding; shared-profile
  validation and `simulation/control_compatibility.py` removed.
- `Supervisor.endpoint(ied_id)` records the endpoint each worker confirmed; Operate (`/equipment/
  {id}/operate`, `/controls/operate`) and gateway commands to simulated IEDs use it; failure
  text names the address.
- Frontend: "External client behavior" choice and its warning removed; event details show plain
  `ctlVal`.
- Probe: `sbo-compat*` removed; optional originator argument for `sbo-*`.

## Evidence

| Check | Result |
| --- | --- |
| Lab mode on real sockets (VMnet8 alias 192.168.98.27:18190, dev backend + rebuilt worker) | loopback 127.50.0.2 → `error 5`; Operate Close `{"ok": true}` in 0.14 s, position 2; Operate Open → 1; `control_applied` peer 192.168.230.1 |
| New/changed tests | `test_command_endpoint.py` (4), `test_native_worker.py` (CONCEN origin gets standard Open on Oper; profile refused; no-op keeps `.t`), `test_mapped_equipment_control.py` (CONCEN origin Close closes), format-14 migration notice |
| Python suite (unit, integration, protocol, packaging) | 504 passed, 11 skipped; `verify.py` PASS |
| Frontend | 156 passed; `tsc` clean |
| Native | all targets rebuilt in `studio/native/build` without errors |

## Limitations

- The VM still runs packaged 0.1.10; it needs a new build to get this fix (no EXE built).
  Retested on the VM with test build 0.1.12 in [0100](0100-test-build-0.1.12-vm-retest.md).
- GE Viewer PROCEED outputs are not simulated; latching push buttons reset on IED restart.
- Captures stay in the working folder (customer model names); they are not committed.
