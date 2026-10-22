# 0178: Bug hunt group 3 — Tools (gateway, master, converter, SNMP)

Date: 2026-10-23. Source: bug hunt 2026-10-23, tester C (verified), plus B16 from tester B. Fixes the
CONFIRMED and PARTLY items of group 3 with the smallest change a user notices only as "it works now": no new
prompts or steps, stored values and contract shapes unchanged, texts the Python engine shares left as they are.

## Items

| id | fixed | what changed |
| --- | --- | --- |
| C2 | yes | DNP3 outstation: output statuses (g10/g40) ONLINE from the start and sent as events (g11v2, g42v3/v4/v7, class 1) |
| C3 | yes | Master *New command* form: word order for the 2-register Modbus writes (FC16) |
| C4 | yes | Protocol converter with a Modbus device: *What to read* tab (the master's ranges) after *Device points* |
| C1 | yes | Smart Propose (Go-only path) adds step commands (BSC/ISC) with their tap position and set points (APC/INC) |
| C5 | yes | SNMP *Change* refused (no access, not writable, not authorized, read-only): hint about the write community |
| C6 | yes | Master *Regulating step* badge is SE, like the gateway's C_RC / command_step |
| B16 | yes | Point panel value rows read from a status (`…stVal`, `…posVal`) get IS instead of MV |
| C9 | yes | Gateway point list: the "sent" note and a pending confirmation are cleared when the protocol changes |
| C10 | yes | Wording: per-protocol pairing example, file vs paste message, "an IEC 104 type", type list joined by "; ", one meaning for word order |
| C11 | yes | Gateway list fits 1366×768 when stopped: Raise…/Lower… and × are whole again |
| C12 | yes | *Not in your list* shows the value and its quality on two lines ("OFF" / "good"), not "OFFgood" |
| C14 | yes | *Find registers*: Stop button (ends after the request on the wire, keeps what was found); result box no longer scrolls inside |
| C15 | yes | A refused connect says "Nothing is listening on host:port (the connection was refused)…" (IEC 104, DNP3, Modbus) |

## Details

**C2** (`native/gateway/dnp3_outstation.cpp`). Output statuses were class 0 and never written, so a master saw
RESTART (not ONLINE) for good and commanded values only on the next integrity poll. Now they are class 1 with
event variations, and `changed()` writes them: a command point added together with its status (the status
reference starts with the command reference + ".", e.g. `XCBR1.Pos` → `XCBR1.Pos.stVal`; set point → its
`mxVal`) follows that status with its flags (COMM_LOST while the source is not live); any other output status
starts ONLINE with 0 and then shows the last commanded value (as before, written by the command handler, now
also as an event). Single command/double command follow single/double statuses (a double is ON when closed).

**C15** (`native/master/client_104.cpp`, `client_dnp3.cpp`, `modbus_link.cpp`, `socket_60870.c`, `client.hpp`).
IEC 104: the replaced lib60870 `Socket_connect` already failed at once on WSAECONNREFUSED; it now records that in
`studio_connect_refused`. DNP3: opendnp3's failed-open counter (`numOpenFail`) is checked while waiting; a
refusal now answers in about 2 s (Windows repeats the SYN once) instead of 10 s. Modbus: `SO_ERROR` of the failed
socket. Every other failure keeps "No answer from … within N s". Text shared by both engines (native).

**C14** (`native/master/main.cpp`, `client_modbus.cpp`; Go `gateway/service/probe.go`, `master_ask.go`,
`toolsapi/master_points.go`; frontend `FindRegisters.tsx`, `ModbusRanges.tsx`, `master.css`). studio_master runs
`probe` on its own thread and answers its request id when it ends; the new op `probe_stop` sets a flag checked
before every request, and the answer then carries `"stopped": true` (only then, so the Python engine's answers are
unchanged). One mutex keeps a single request on the Modbus link, so a command or poll during Find registers waits
for the request on the wire instead of the whole probe. A disconnect or stop first stops and joins the probe.
Go-only route `POST /api/v1/tools/master/probe/stop` (listed in `MasterGoOnlyRoutes`) answers `{probe}`; the probe
state gains `"stopped"` with "Stopped by you after N requests." and the ranges found so far, which *Add to the read
list* takes like a finished run. The Find registers box no longer has the 40 % inner scroll of `.panel-messages`.

**C1** (`gateway/addpoints/smart_typed.go`, `smart.go`). `gateway.Propose` (Python parity, also the DNP3
auto-propose at Start) is unchanged. Only `addpoints.Smart` adds the typed controls of the proposed IEDs as the
kinds Add points offers (`CommandFits`); `withStatuses` adds the tap position or measured value and `readdress`
pairs addresses as for SPC/DPC. A kind the protocol does not carry (a step on Modbus) is left out. The preview
summary counts the typed commands; statuses are added later, as for SPC/DPC.

**C3** (`api/master.ts`, `master/wordOrder.ts`, `MasterCommandForm.tsx`). `MasterCommand.word_order` (`big` or
`little`, the stored values the engine already reads) with the same words as *What to read*: "High word first" /
"Low word first". Kept when the command type changes; shown only for FC16 writes.

**C4** (`converter/converterTabs.ts`, `ConverterTool.tsx`). The converter reads through the master service, so
the tab renders the existing `ModbusRanges`; when the device protocol changes away from Modbus the view falls
back to *Device points*.

**C10**. `copy/gatewayAdd.ts`: the status-pairing example per protocol (IEC 104 20003/50003, DNP3 "input index 3
for command index 3", Modbus "discrete input or register 3 for coil 3"). `toolsapi/add_points.go`: an uploaded
file without rows says "No rows were found in the file. Fill in the rows (below the title row) of the Points
sheet, save the file and choose it again." (paste keeps its text). `masterpoints/rows.go`: "is not an IEC 104
type" (a DNP3 / a Modbus) and the type names joined by "; " because names hold commas; `addpoints/rows.go` the
same "; ". Word order: stored values unchanged (`high_first`/`low_first` in the gateway, `big`/`little` in the
master); the Go-only gateway check now says "high_first (high word first) or low_first (low word first)", and
the master list paste also reads `low_first`. The Python-shared range text ("big (high word first) or little")
is unchanged.

**C11** (`gateway/PointRow.tsx`, `gateway.css`). Measured at 1366×768 with SampleTyped, gateway stopped: the
table was 904 px in an 816 px box. Up to 1440 px window width *Change* is its ✎ pin like the other cells (the
word stays in its hover text and accessible name), the IED path cell is at most 270 px (full path on hover, as
before) and a value text at most 120 px (full text on hover). Result: 816 px, no sideways scroll.

**C5** (`copy/snmpRead.ts`, `WriteValue.tsx`): the refusal keeps the device's words and adds "Changing values
usually needs the write community (often private), or a v3 user that may write. Connect again with it, then try
the change again." No new field.

**C6, B16** (`points/pointKind.ts`, `equipment/pointDisplay.ts`). The badge map only.

**C9, C12**: `PointTable.tsx` clears `sent` and `pending` with the other per-protocol state (as the master does
since 0089); `NotInList.tsx` uses the `master-list` table class, whose `.list-reason` is a block.

## Evidence

- Go: `go vet ./...` clean; `go test` gateway/..., toolsapi, app, snmp/... pass. New:
  `TestSmartProposesTypedCommands` (fails before: no step/set point), `TestProbeStop`; updated texts in
  `rows_test.go` (both), `typed_kinds_test.go`, `add_points_test.go` (fails before: paste text for a file).
- Native wire tests (Python rig against the real workers): new
  `test_dnp3_output_statuses_are_online_follow_status_and_report_events` (fails before: flags RESTART) and
  `test_find_registers_stops_on_request_and_commands_still_go_through` (fails before: unknown op); the two
  unreachable-device tests now expect "Nothing is listening" and cover Modbus too. 25 of 26 protocol tests pass;
  the failing one is pre-existing (see Limitations).
- Frontend: `npx tsc --noEmit` clean; 431 node tests pass, new `bughuntTools.test.ts` (C3, C4, C5, C10) and
  additions in `pointKind.test.ts` (C6) and `pointPanel.test.ts` (B16).
- Browser (dev build of this branch, SampleTyped.scd only, loopback, 1366×768):
  - C1: smart Propose IEC 104 gave 33 points (8 step commands paired with 8 tap positions at 45001…, 8 set points);
    the plain Propose for the same project gave 1 point. Modbus: set points, no steps.
  - C11: stopped gateway list, filter ATCC: table 816 px in 816 px (was 904 px), Raise…/Lower… and × whole.
  - C15: IEC 104 Connect to 127.0.0.1:12999 answered in 266 ms with the new text; DNP3 and Modbus 2.2 s each.
  - C12: *Not in your list* rows read "5" / "good" on two lines (`.list-reason` display block).
  - C3: Modbus master, *Write float, 2 registers (FC16)* shows *Word order* (High/Low word first).
  - C14: holding registers 0–9999, Stop after 28 requests: "Stopped by you after 30 requests." 267 ms later;
    990–1020 found, *Read these as* and *Add to the read list* fully visible (no inner scroll).
  - C4: converter tabs: Device points, What to read, Served to SCADA, …; the tab shows Find registers.
  - C2: master DNP3 against the gateway outstation: all g10/g40 points "good" after the integrity poll; an analog
    output 12.5 and a CROB close arrived as unsolicited events (AO 2 = 12.5, BO 9 = ON).
  - C6: master *Commands* log shows *Regulating step (C_RC)* with SE.

## Limitations and not fixed

- B16 uses the attribute name (`stVal`, `posVal`), because a panel row stores no functional constraint; an ST
  value under another name stays MV. Not checked in the browser (no point panel in the test project); unit test.
- C2 link by reference prefix only; a command whose status has another data object (an SPC pair) shows the last
  commanded value. Step commands (CROB raise/lower) keep the last command, as a tap position is no on/off state.
- C11 at ≤1440 px the path shows fewer characters before "…" (full path on hover); the IED name, buttons and ×
  stay whole. C9 and C5 were checked by code and unit test, not in the browser.
- C14: Stop waits for the request on the wire (at most the 3 s Modbus timeout); a probe on the gateway still
  reports 990–1020 because the gateway answers zeros inside mapped blocks (by design, verified-C).
- Pre-existing, not touched: `test_master_wire.py::test_dnp3_master_integrity_poll_events_and_crob` fails on
  this branch and on the main checkout's build ("CROB latch on index 4" no longer matches the traffic line).
  `TestTypedClassesInEveryControlModel` (tools/control) failed once under full-suite load and passed 3× alone.
- The dev server's `index.html` was cached by the browser across rebuilds (a reload with a new query string
  fixed it); worth a look under "no stale builds", not changed here.
- Not fixed by plan: C7, C8, C13, C16 (not bugs or not reproducible). No ADR: the only contract additions are
  the Go-only stop route, the native `probe_stop` op and the optional `stopped` answer field (backward
  compatible both ways); the owner may want them in ADR-0110 or the scada-master spec.
