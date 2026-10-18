# Spec: Find IEDs

Decision: [ADR-0054](../decisions/ADR-0054-tools-place.md).
Change: [0081](../changes/2026/0081-ied-explorer-find-ieds.md).

## Purpose

Answer "what IEC 61850 devices are on this network?" in one press, with vendor, model and
firmware, so a beginner knows where to connect.

## Behaviour contract

- **Choose.** Each connected adapter's own network is offered, plus "This PC" (simulated IEDs
  on 127.50.0.x). The last choice, range and port are remembered.
- **Preview.** A /24 or smaller network, one address, or a range such as `a.b.c.20-40`. The
  page states exactly what will be checked before Find IEDs is pressed.
- **Probe.** Port 102 (or the chosen port) is opened with bounded parallelism and a short
  timeout, using plain sockets. Progress is shown and the scan can be stopped.
- **Identify.** Each open port is asked for its MMS identity (vendor, model, revision) and its
  logical devices through the read-only `identify` operation. A device that refuses is
  listed with the reason.
- **Results.** Answer times, identity and an "Open in Explorer" button.
- **Layout (ADR-0064).** The form is one toolbar row with the "N addresses" preview and a "?";
  a thin progress line sits under it and the results fill the window, filtered by search,
  Answered, IEC 61850 only, Not in project and vendor ([tool layout](tool-layout.md)).
- **Safety.** No writes and no other ports. Only the previewed range is scanned.

| ID | Rule | Enforced by |
| --- | --- | --- |
| FIND-01 | Ranges are limited to 256 addresses and explained before scanning. | `studio/tests/unit/test_explorer_find.py::test_scan_range_is_limited_and_explained` |
| FIND-02 | Each adapter network is proposed (at most a /24) plus this PC. | `studio/tests/frontend/tools.test.ts::find IEDs proposes each adapter network, at most a /24, plus this PC` |
| FIND-03 | Open ports are found and identified; a scan can be stopped and needs the live client. | `studio/tests/unit/test_explorer_find.py::test_scan_finds_open_ports_and_identifies_them`, `studio/tests/unit/test_explorer_find.py::test_scan_needs_the_live_client_and_can_be_stopped` |
| FIND-04 | Identify reports identity and devices, or a reason. | `studio/tests/protocol/test_explorer_wire.py::test_identify_reports_identity_and_devices`, `studio/tests/protocol/test_explorer_wire.py::test_identify_unreachable_port_gives_a_reason` |
| FIND-05 | A real scan finds and identifies a stand-in IED end to end. | `studio/tests/protocol/test_explorer_wire.py::test_find_ieds_scans_a_range_and_identifies_the_stand_in`, `studio/tests/integration/test_tools_api.py::test_explorer_and_find_ieds_over_http` |
