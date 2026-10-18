# 0088: SCADA master and protocol converter (IEC 104, DNP3, Modbus TCP), correct SCADA words

Date: 2026-10-18. Decision: [ADR-0061](../../decisions/ADR-0061-scada-roles-master-converter.md).
Spec: [SCADA master and protocol converter](../../specs/scada-master.md) (MASTER-01..12).

Owner request: "in the real SCADA, there is master and slave or client and server or gateway …
we should simulate each". The gateway that gets data from another machine should find all
points by default, and the user can choose which to check. The owner also said "be careful with
the correct words".

## Change

- **Native: new worker `studio_master`** (`native/master/`). It is licensed on connect and in
  the `STUDIO_GATEWAY` CMake block.
  - `client_104.cpp`: the IEC 104 controlling station, using `iec104_decode` (monitoring ASDUs
    as records with words for the cause) and `iec104_command_objects` (C_SC, C_DC, C_SE, with
    or without time tag).
  - `client_dnp3.cpp` and `dnp3_values.cpp`: the opendnp3 master (integrity and event polls,
    unsolicited events, CROB and analog outputs).
  - `client_modbus.cpp`, `modbus_link.cpp` and `modbus_reads.cpp`: the Modbus TCP client
    (typed ranges, FC5/6/16 writes, bounded Find registers).
  - `value_events.cpp`: `master_values`, `master_link` and `master_interrogation` events.
    Traffic reuses `gateway/traffic.cpp`.
- **Packaging.** `studio_master.exe` is in both PyInstaller specs, `protected_source.SEALED` and
  the `compile_protected.py` targets.
- **Backend.**
  - `domain/master_rules.py`: settings, host, ranges, Find registers bounds and command routes.
  - `domain/master_points.py`: the discovered table, quality words, labels, CSV and command
    kind per point.
  - `domain/passthrough.py`: the converter mapping.
  - `gateway/master_manager.py`: connect, discovery, re-interrogation, reconnect, commands and
    auto-connect. The point table has its own lock, so values arriving during a connect never
    block the worker reader.
  - `gateway/modbus_probe.py`: Find registers in the background, with progress.
  - `gateway/converter.py`: remote values and command relay. `gateway/manager.py` and
    `commands.py` use it in the converter role.
  - `api/gateway_master.py` (`/tools/master/...` and `/tools/gateway/role`).
  - `GET /activity` has a `master` entry.
  - Project format 14 (`role` plus the `master` block), with an upgrade notice for format 13.
- **Frontend.**
  - The tool is renamed "SCADA gateway & master". `ScadaTool` and `RoleSwitch` provide "Act as"
    and the "Which side am I?" diagram.
  - `master/`: connect card, status bar, discovered points (tick boxes, names, filter, CSV,
    command buttons), the command form with the REAL-device warning, the command log, and
    Modbus "What to read" with Find registers.
  - `converter/`: device side, upstream side, served points and command routes.
  - Copy: `copy/master.ts` and `copy/converter.ts`, and ten glossary terms. The gateway copy
    now says controlled station / Modbus server, and "SCADA connection" instead of "SCADA
    master".

## Evidence

- **Wire, master against `studio_gateway`** (`test_master_wire.py`):
  - 104: GI finds every IOA; spontaneous change; select, execute and ACT_TERM; negative
    confirmation and unknown-IOA refusal.
  - DNP3: integrity poll, events, select-and-operate CROB, refusal "not authorized",
    unreachable outstation.
- **Independent Modbus server** (Python stand-in in `test_master_modbus_wire.py`): Find
  registers 90-129 finds exactly 100-109 in at most 40 requests; typed reads; a refused range
  is worded; coil write; the probe bound is enforced.
- **API end to end:**
  - `test_master_runtime.py` (104: discover, name, untick, CSV, confirmed command, remembered
    connection);
  - `test_master_modbus_runtime.py` (find, add ranges, live poll sees a change, coil write).
- **Converter:** the independent lib60870 104 master reads the DNP3 outstation's analog value
  (230.5, good) through ScadaBench, and its C_SC is confirmed and terminated and reaches the
  outstation (`test_converter_runtime.py`).
- **License:** the enforced `studio_master` refuses to connect without a license.
- **Totals:** 431 Python (unit, integration, protocol) passed with 7 skipped; 114 frontend
  passed; tsc clean; `verify.py` PASS.
- **Source app on port 18766** with a temporary data folder (the owner's app on 8765 was not
  touched), at 1440×900:
  - 104 master: 32 points discovered and 9 rows visible with no page scroll; the confirmed
    C_DC was select confirmed, execute confirmed and terminated; after a restart it reconnected
    by itself.
  - Modbus: Find registers, then added ranges, then 375 points polled.
  - Converter: 40 DNP3 points served as 48 IEC 104 points.
  - No console errors.

## Limitations

- No third-party SCADA, RTU or gateway has been recorded yet; the evidence is listed above.
- IEC 104 command addresses cannot be discovered. The converter relays 104 commands only
  through command routes.
- Modbus has no quality: a refused range is shown as "no answer". Find registers reads one
  table per run.
- Serial IEC 101, Modbus RTU and DNP3 serial, 104 redundancy groups and file transfer are not
  included.
- No EXE was built (AGENTS.md: only on request).
