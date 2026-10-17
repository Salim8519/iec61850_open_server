# 0081: IED Explorer and Find IEDs, with autonomous watching and confirmed commands

Date: 2026-10-17. Decisions: [ADR-0054](../../decisions/ADR-0054-tools-place.md),
[ADR-0057](../../decisions/ADR-0057-gateway-explorer-commands.md).
Specs: [IED Explorer](../../specs/ied-explorer.md), [Find IEDs](../../specs/find-ieds.md).

Owner requests:
- "tools well crafted, nice UI, clear, and fast";
- autonomous mode, with every setting remembered;
- "on by default, user can send real commands, just small confirmation before send".

## Change

- **Native.**
  - `studio_live` gains the read-only `explore` operation (devices, nodes, one node) and the
    `identify` operation, through `common/browse` and `live/explore` + `live/identify`.
  - The new `studio_control` (`native/control/`) operates controls with the device's own
    control model and reports AddCause.
- **Backend.**
  - `tools/explorer.py` keeps one warm worker, so connect plus tree take about 1 s.
  - `tools/explorer_auto.py` with `domain/explorer_watch.py`: watches key points by itself
    and restores the watch list per address.
  - `tools/ied_scan.py` uses socket probes with bounded parallelism.
  - `tools/control_client.py` and `tools/tool_memory.py` (`tool-settings.json`).
  - Routes in `api/tools.py`, including `POST /tools/explorer/operate` (needs `confirm: true`)
    and `GET /tools/memory`.
- **Frontend.**
  - `features/tools/explorer/*`: tree, node detail, watch list, and `CommandPanel` with an
    inline confirmation.
  - `features/tools/find/*`: the last network choice is remembered.
- **Packaging.** `studio_control.exe` is added to both specs, `SEALED` and the protected
  compile targets.

## Evidence

- **Protocol tests:**
  - `test_explorer_wire.py`: 6 pass;
  - `test_control_client.py`: 5 pass. A breaker closes and opens with its own model; a
    blocked control returns the device's refusal; a real-role IED is commanded through the
    gateway path.
- **Integration:** `test_tools_api.py` covers connect, tree, read, watch, CSV and a confirmed
  operate. The page reads "on (closed)" afterwards, and the remembered watch list is saved.
- **Dev app on the owner's project (6 IEDs, 90k signals):**
  - window ready 0.5 s;
  - Explorer connect 0.9 s, then 40 key points watched and polling by itself;
  - the breaker confirmation shows "Close" and "Open" and names the object and device. Cancel
    sends nothing.
- **Enforced build:** `test_control_client_refuses_to_operate_without_license` passes.
- **Totals:** 358 unit + integration, 50 protocol (7 skipped need hardware) and 108 frontend
  tests pass; `verify.py` PASS.

## Limitations

- Commands were verified against Studio's own simulated IEDs (libiec61850 server) only, not
  yet against a third-party relay.
- Find IEDs identifies over MMS. Devices that refuse an association show "not identified" with
  the reason.
- The owner's visual acceptance is pending.
