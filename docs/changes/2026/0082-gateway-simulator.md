# 0082: Gateway simulator (IEC 104, Modbus TCP, DNP3) with smart points and autonomous start

Date: 2026-10-17. Decision: [ADR-0057](../../decisions/ADR-0057-gateway-explorer-commands.md).
Spec: [Gateway simulator](../../specs/gateway.md).

Owner requests:
- a gateway simulator with a FreyrSCADA-like UI;
- points proposed from the SCD and editable, with CSV;
- autonomous mode (data collection starts by itself, smart list, remember and auto-start);
- real-device commands allowed with a confirmation.

## Change

- **Native `studio_gateway`.**
  - lib60870-C 2.3.6 for the IEC 104 slave; Studio's own Modbus TCP server; the opendnp3
    3.1.2 outstation.
  - Point table, command bridge (15 s answer timeout) and a rate-limited traffic log.
  - Licensed and sealed.
  - CMake option `STUDIO_GATEWAY` (default ON). `cmake/gateway_support.cmake` and
    `fetch_gateway_deps.py` pin the dependencies.
- **Test masters** `studio_test_iec104_master` and `studio_test_dnp3_master` give independent
  evidence. They are never packaged.
- **Backend.**
  - Pure domain modules: `domain/gateway_points.py`, `gateway_selection.py` (smart list),
    `gateway_rules.py`, `gateway_csv.py` and `gateway_values.py`.
  - `gateway/manager.py`: start, autonomous preparation, value sync, auto-start.
  - `gateway/commands.py`: request, applied and observed logged through `studio_control`.
  - `api/gateway.py`.
  - Project format 13 adds the `gateway` section.
  - `app_factory` starts a remembered gateway after the project has been restored.
- **Frontend `features/tools/gateway/*`:**
  - settings, with timers under Advanced and an "Allow SCADA commands to real IEDs" switch;
  - a point table (100 rows per page, live values);
  - a traffic log with filter and pause;
  - a command log.
- **Speed (owner project).**
  - Proposing took 15 s and now takes 0.6 s: per-block address cursors instead of
    rescanning, and a per-IED signal index in validation.
  - The smart list fell from 8,922 capped points (thousands of spare GGIO inputs and
    individual harmonics) to 3,240 useful ones.

## Evidence

- **`test_gateway_wire.py` (5 pass):**
  - lib60870 master: general interrogation, spontaneous changes, SC/DC select/execute with
    ACT_CON/ACT_TERM, and refusals (COT 44-47);
  - Modbus FC1-5/15 with exceptions;
  - opendnp3 master: integrity scan and CROB, with NOT_AUTHORIZED for monitor-only points.
- **`test_gateway_runtime.py`:** the autonomous start proposes points and starts Run. A 104
  master then selects and executes a double command on a simulated breaker. The position
  returns spontaneously, the command log shows "operated" and "is now", and a user Stop
  clears `auto_start`.
- **`test_control_client.py::test_gateway_commands_a_real_role_ied_and_respects_the_switch`:**
  - with the switch on, a real-role IED is operated;
  - with it off, the command is refused and nothing is sent.
- **Dev app on the owner's project (6 IEDs):**
  - the gateway starts in 2.5 s with Run already on, or about 12 s when it also starts Run;
  - 3,240 points;
  - general interrogation from the independent master returns 2,652 monitoring objects, all
    with good quality.
- **Enforced build:** the gateway refuses to listen without a license. Tamper detection
  covers `studio_gateway.exe`.
- **Totals:** 358 unit + integration, 50 protocol and 108 frontend tests pass; `verify.py` PASS.

## Release build fix

- The first 0.1.5 build attempt stopped with `studio_gateway.vcxproj` missing. The old
  `native/build/protected` cache still held `STUDIO_GATEWAY=OFF` from before the default
  changed, and CMake keeps cached option values.
- `compile_protected.py` now always passes `-DSTUDIO_GATEWAY=ON`, and the dry-run test
  checks it together with `studio_control`.

## Release 0.1.5 and website publish (2026-10-17, owner: "deploy the work")

- `ScadaBench-Setup-0.1.5.exe`: 23.7 MB, SHA-256 `3dd0da43…75e979`, leak check clean, unsigned.
  - Kaspersky locked the new setup file twice ("being used by another process"); the third
    Inno Setup attempt succeeded.
- **Built app checks** (own port 18765, temporary data folder):
  - `--self-test` ok; health reports 0.1.5; the owner's lifetime license is accepted; the UI
    is ready in 0.28 s; the 4 tool cards are shown.
  - With no license, the packaged `studio_ied`, `studio_live`, `studio_probe`,
    `studio_gateway` and `studio_control` all refuse to work.
  - Gateway end to end against the independent 104 master: autonomous start in 0.2 s; a
    general interrogation returned 11 objects, all good; select/execute was confirmed
    (ACT_CON ×2, ACT_TERM); the position came back spontaneously and the log shows
    "operated" and "is now 2".
- **Website:**
  - version 0.1.5 and a 0.1.5 changelog entry;
  - claims corrected for commands, IEC 104 support and the gateway (FeatureGrid, features,
    home "Safe by design", FAQ, contact);
  - the FAQ line "never connect it to a live operational network" was left unchanged for the
    owner to decide;
  - the owner ran `deploy.cmd`.
- **Live checks:** `/`, `/features/`, `/faq/`, `/changelog/`, `/download/` (Version 0.1.5),
  `/pricing/` and `/contact/` all return 200. The downloaded `/download/ScadaBench-Setup.exe`
  SHA-256 equals the build.
- The installed copy on the owner's PC was not changed by this release.

## Limitations

- No third-party SCADA master is recorded yet (FreyrSCADA, a real SCADA). Interoperability is
  claimed only against the independent lib60870 and opendnp3 masters.
- Real-IED commands through the gateway were tested against a stand-in IED with the real
  role, not a vendor relay.
- Starting Run for a large project takes about 10 s, dominated by IED start-up, not the
  gateway.
- Not yet in an installer: the packaged build waits for the owner's release request.
