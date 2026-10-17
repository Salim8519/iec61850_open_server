# ADR-0057: Gateway simulator, IED Explorer commands and autonomous tools

Accepted: 2026-10-17. Owner decisions in this session:
- build the gateway now, over IEC 60870-5-104, Modbus TCP and DNP3;
- "on by default, user can send real commands, just small confirmation before send";
- autonomous mode: the gateway starts data collection by itself, proposes a smart point
  list, remembers that it was running and starts again; Explorer watches key points by
  itself; every tool setting the user changes is remembered.

**Decision**

- **Workers.** Three new native programs, each licensed like the others
  (`license::require_valid_license()` before it serves) and sealed in the protected build
  (`protected_source.py`):
  - `studio_gateway`: protocol servers toward SCADA. It has no IEC 61850 client, so it can
    never operate an IED by itself. Commands go to the backend as `gw_command` events and
    are answered with `command_result`.
  - `studio_control`: the IEC 61850 control client. It keeps associations open and uses the
    device's own control model (direct or SBO, normal or enhanced security), originator,
    interlock and synchro check, waits for the termination, and reports AddCause.
  - `studio_live` (existing): gains the read-only `explore` and `identify` operations.
    LIVE-01 is unchanged: it still has no write or control operation.
- **Dependencies.**
  - lib60870-C 2.3.6 (MZ Automation, GPLv3/commercial): IEC 104 slave. Covered by the same
    commercial arrangement as libiec61850. The licence file ships in `licenses/`.
  - opendnp3 3.1.2 (Apache-2.0; upstream archived): DNP3 outstation, pinned by commit with
    its asio/ser4cpp/exe4cpp dependencies (`studio/tools/fetch_gateway_deps.py`, SHA-1
    checked). LICENSE and NOTICE ship in `licenses/`.
  - Modbus TCP: Studio's own small server (`gateway/modbus_pdu.cpp`). No new dependency.
  - Test-only masters (`studio_test_iec104_master`, `studio_test_dnp3_master`) are built
    from the same libraries and are never packaged. They give independent-client evidence.
- **Project format 13.** The project gains a `gateway` section: protocol, listen choice,
  settings per protocol, point list per protocol, `real_commands` (default on) and
  `auto_start`. It is checked for structure on restore and left out of the project
  snapshot, so engineering comparisons ignore it.
- **Commands to real IEDs (supersedes the "never command real IEDs" part of ADR-0031 for
  the Tools place only).**
  - IED Explorer operates a control only through `POST /tools/explorer/operate` with
    `confirm: true`. The page sends it only from the confirmation step.
  - The gateway forwards a SCADA command to a real IED while "Allow SCADA commands to real
    IEDs" is on. When it is off, the native worker answers negatively (104 negative
    ACT_CON, DNP3 NOT_AUTHORIZED, Modbus exception) and the backend refuses as well.
  - SLD and Test keep refusing commands to real IEDs (`command_refusal`). That is a separate
    owner decision, not made here.
  - Live acquisition is unchanged: reports on free report blocks only, polling fallback.
- **Autonomous tools.**
  - Gateway Start proposes the point list when it is empty and starts Run when it is
    stopped. `auto_start` is set after a successful start and cleared by a user Stop. After
    the project has been restored, the app starts a remembered gateway again. A failure is
    shown in the gateway status, never raised.
  - The smart point list (`domain/gateway_selection.py`) keeps:
    - positions, switchgear and supervision statuses, and protection trips;
    - measurements and harmonic totals;
    - logical-device health and every controllable object.

    A generic I/O (GGIO) value needs evidence that it is used: it is in a dataset, or the
    SCD describes it. Ranking keeps generic values last if the 5,000-point limit is
    reached.
  - Explorer restores the watch list it had for that address, or watches up to 40 key
    points, and starts polling.
  - Tool settings live in `tool-settings.json` in the data folder (`ToolMemory`), checked
    on read so a damaged file only resets the tool memory.

**Compatibility**
- Format 12 projects open with an empty gateway section; older apps refuse format 13 as
  usual for a newer format.
- Two new executables join the packaged and protected builds. Dev and packaged builds use
  the same programs (`native/build/Release` and the enforced build).
- No interoperability is claimed beyond the recorded evidence: independent lib60870 and
  opendnp3 masters, and a raw-socket Modbus client. Third-party SCADA masters and real
  relays remain open release gates.
