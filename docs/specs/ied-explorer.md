# Spec: IED Explorer

Decisions: [ADR-0054](../decisions/ADR-0054-tools-place.md),
[ADR-0057](../decisions/ADR-0057-gateway-explorer-commands.md).
Change: [0081](../changes/2026/0081-ied-explorer-find-ieds.md). Models: the Siemens IEC
Browser polling view and the IEDScout data model browser.

## Purpose

Connect to any IEC 61850 server by address, without a project. Browse its data model, watch
the values that matter, and operate a control after one small confirmation.

## Behaviour contract

- **Connect.** Type an IP address (port 102 by default) or open a Find IEDs result.
  - One warm `studio_live` worker serves one Explorer association, independent of Run.
  - The last address, port, interval and watch list per address are remembered.
- **Autonomous watch.** After connecting, Explorer restores the watch list it had for that
  address. Otherwise it opens only the key logical nodes and watches up to 40 key points:
  - positions;
  - trips;
  - measurements;
  - health.

  Then it starts polling.
- **Tree.** Logical device → logical node → data object → attribute, with FC badges.
  - Each level loads only when opened.
  - Values, quality and times are shown as words and an icon, never by colour alone.
- **Watch list.** Polls at 0.5–10 s, offers Read now, and exports to CSV.
  - Report blocks are listed only; Explorer never enables or reserves them.
- **Commands (owner decision 2026-10-17).**
  - A data object with `Oper.ctlVal` shows its choices: Close/Open for a double-point
    position, On/Off for a single point, a number for step and analogue controls.
  - Each command is sent only after a confirmation that names the object and the device.
  - `studio_control` operates with the device's own control model and reports the device's
    answer and AddCause. Explorer then reads `stVal` again and shows it.
- **Layout (ADR-0064).** A toolbar row (address, port, Connect, status, ?); the model tree in the
  left column; the node detail with the watch list pinned below it, each scrolling on its own.
  The tree filters by logical-node class and the watch list by change and quality.
- **Connect always starts clean.** One connect runs at a time; Connect and Disconnect reset the
  live worker's association whatever the previous state, so a failed or repeated connect never
  leaves it answering "Already connected" (change 0095).
- **Read paths stay read-only.** `studio_live` has no write or control operation (LIVE-01).
  Commands use the separate `studio_control` worker.

| ID | Rule | Enforced by |
| --- | --- | --- |
| EXPLORE-01 | The model loads one level at a time and needs an association. | `studio/tests/protocol/test_explorer_wire.py::test_explore_one_level_at_a_time`, `studio/tests/protocol/test_explorer_wire.py::test_explore_needs_a_connection` |
| EXPLORE-02 | Values, quality and times are described in plain words; attributes are grouped by data object with blocks listed. | `studio/tests/unit/test_explorer_find.py::test_values_are_described_in_plain_words`, `studio/tests/unit/test_explorer_find.py::test_node_view_groups_attributes_and_blocks` |
| EXPLORE-03 | Bad addresses and ports get actionable messages. | `studio/tests/unit/test_explorer_find.py::test_endpoint_checks_are_actionable`, `studio/tests/integration/test_tools_api.py::test_bad_input_gets_actionable_errors` |
| EXPLORE-04 | The watch list polls, reads and exports CSV through the API. | `studio/tests/protocol/test_explorer_wire.py::test_explorer_service_browses_and_polls_a_watch_list`, `studio/tests/integration/test_tools_api.py::test_explorer_and_find_ieds_over_http` |
| EXPLORE-05 | Explorer watches key points by itself and remembers settings per address. | `studio/tests/unit/test_tool_autonomy.py::test_explorer_picks_key_nodes_and_points`, `studio/tests/unit/test_tool_autonomy.py::test_tool_memory_survives_damage_and_keeps_every_tool` |
| EXPLORE-06 | A command is sent only with confirmation, only from the confirmation step, with choices that match the object. | `studio/tests/frontend/tools.test.ts::explorer sends a command only through the confirmation step and never writes`, `studio/tests/frontend/tools.test.ts::command choices follow the MMS types Explorer shows`, `studio/tests/integration/test_tools_api.py::test_bad_input_gets_actionable_errors` |
| EXPLORE-07 | Commands use the device's control model and report the device's own refusal. | `studio/tests/protocol/test_control_client.py::test_breaker_closes_and_opens_with_its_own_control_model`, `studio/tests/protocol/test_control_client.py::test_blocked_control_reports_the_device_reason`, `studio/tests/protocol/test_control_client.py::test_unknown_object_is_explained`, `studio/tests/protocol/test_control_client.py::test_unreachable_device_is_explained` |
| EXPLORE-08 | The control program refuses to operate without a valid license. | `studio/tests/unit/test_native_enforcement.py::test_control_client_refuses_to_operate_without_license` |
| EXPLORE-09 | A failed, repeated or overlapping connect never leaves the live worker associated; the next Connect works. | `studio/tests/unit/test_explorer_connect.py::test_connect_after_a_failed_connect_resets_the_worker`, `studio/tests/unit/test_explorer_connect.py::test_overlapping_connects_to_one_device_both_succeed`, `studio/tests/protocol/test_explorer_wire.py::test_explorer_reconnects_after_overlapping_and_failed_connects` |
