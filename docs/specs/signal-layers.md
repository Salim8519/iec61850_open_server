# Spec: SLD signal layers (GOOSE messages, Who is reading)

Decisions: [ADR-0065](../decisions/ADR-0065-sld-signal-layers.md),
[ADR-0066](../decisions/ADR-0066-goose-inputs-and-mms-access-events.md).
Change: [0094](../changes/2026/0094-goose-mms-signal-layers.md).
Owner request (2026-10-18): see GOOSE and IEC 61850 client activity on the single-line diagram
without distraction.

## Purpose

Two switches on the SLD show who sends GOOSE to whom and which programs read each simulated
IED, as quiet IED tags and one dot per real change. Devices shows the same partners as tables.

## Behaviour contract

**GOOSE link map** (`scl/goose_inventory.py`, `goose_remote.py`, `goose_sources.py`,
`domain/goose_link_map.py`, `goose_link_live.py`, `GET /goose/links`)
- Built from every SCL file stored with the project; an IED found in several files is read once,
  from the file it was loaded from. Only links touching a project IED are listed.
- Identity is the control block reference (goCBRef); APPID and MAC are attributes only.
- Each GSEControl has a status: `sendable`, `disabled_in_file` (GE `Function="Disabled"`),
  `no_dataset`, `no_address`, `not_goose`, `not_sendable`. Links keep the block's status;
  a link whose publisher is in no file is `remote`.
- Remote inputs count only when they say they are GOOSE: `serviceType="GOOSE"`, a
  `srcCBName`, or a packed vendor `intAddr` whose MAC field is a MAC. Bare spare `ExtRef`
  (`intAddr` + `desc` only) are ignored.
- Diagnostics (level, code, message, fix hint): unresolved input (problem), block not sent
  (warning), remote publisher (info), APPID 0 (hint), duplicate APPID (warning), shared MAC
  (hint). None fails an import.
- Live join per goCBRef: stopped → `stopped`; running without a watching listener → `null`
  ("Not watched"); otherwise the listener's state, problem and stNum. A declared link is never
  presented as received without listener evidence.

**SLD** (`features/diagram/signals/`, `equipment/signalLayout.ts`, `signalPulses.ts`,
`signalLayerStore.ts`)
- Two switches, *GOOSE messages* and *Who is reading*, with On/Off words; remembered in the
  backend preferences `sld_goose_layer`, `sld_readers_layer`.
- Tags beside the topmost equipment bound to the IED; outside or unbound IEDs in a left column,
  outside ones dashed with "outside project". Tags are never saved. They scale 1–3× against zoom.
- Pulse: only when a listener frame raises stNum (never the first frame, never a
  retransmission), at most one per link per second, ~0.7 s dot then a ~2 s fading link and a
  label "PUB → SUB · change". Without a listener, a publisher value that is a dataset member of
  a sent block gives a hollow dashed pulse labelled "(expected)". Reduced motion: static.
- Persistent subscriber problems in words with a dashed border; "waiting" only once a listener
  reports. Clicking a tag or link opens the side sheet with the signal-flow details.
- *Who is reading*: "◉ n read" for external MMS clients; hover lists IP, nodes read and report
  blocks; an external control shows one "‹ip› operated ‹object›" line.

**Devices** (`features/devices/GooseHint.tsx`, `GoosePartners.tsx`, `ReadByClients.tsx`)
- List hint "↑ sends to n · ↓ gets from X · ⚠ n not heard"; Overview tables *GOOSE partners*
  (direction, partner chip, block · what, status) and *Read by clients* (address, reads,
  reports, commands).

**MMS access** (`native/worker/server/access_log.*`, `read_hook.*`, `simulation/mms_access.py`,
`domain/mms_access.py`, `GET /mms/access`)
- Per client (`ip:port`) per logical node read counts, control writes and enabled report blocks,
  at most one `mms_access` summary per second, live view only. Disconnected clients disappear.

## Out of scope

Listen-only lab mode (watching a real station without simulating), physical VLAN switches,
Sampled Values on the SLD, counting data-attribute writes (SP/CF/DC), R-GOOSE.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| LAYER-01 | A vendor-disabled GOOSE block is not published, says why at import and keeps its vendor attributes. | `studio/tests/unit/test_goose_link_map.py::test_vendor_disabled_block_is_not_sent_and_vendor_data_is_kept` |
| LAYER-02 | A link to a block that is not sent stays visible with its reason and a fix hint. | `studio/tests/unit/test_goose_link_map.py::test_link_to_disabled_block_is_kept_with_its_reason` |
| LAYER-03 | A packed vendor intAddr is read only in its strict shape and only as a hint. | `studio/tests/unit/test_goose_link_map.py::test_packed_vendor_intaddr_is_only_a_hint` |
| LAYER-04 | A publisher in no file is a remote publisher (info), never a problem; bare spare inputs are ignored. | `studio/tests/unit/test_goose_link_map.py::test_publisher_outside_every_file_is_remote_not_a_problem`, `studio/tests/unit/test_goose_link_map.py::test_bare_spare_inputs_are_ignored` |
| LAYER-05 | APPID 0, duplicate APPID and shared MAC are hints with fixes, never failures. | `studio/tests/unit/test_goose_link_map.py::test_appid_zero_duplicate_appid_and_shared_mac_are_hints_not_failures` |
| LAYER-06 | An IED copied into several files is read once, preferring its own file; the map is deterministic. | `studio/tests/unit/test_goose_link_map.py::test_stub_ied_in_several_files_is_read_once_preferring_its_own_file`, `studio/tests/unit/test_goose_link_map.py::test_link_map_is_deterministic` |
| LAYER-07 | A declared link is never shown as received without listener evidence. | `studio/tests/unit/test_goose_link_map.py::test_live_join_never_marks_a_declared_link_received` |
| LAYER-08 | `/goose/links` names publisher, block, subscriber, dataset members and per-IED summary. | `studio/tests/integration/test_signal_layers_api.py::test_goose_links_name_publisher_block_and_subscriber` |
| LAYER-09 | Both switches are remembered by the backend, not the browser. | `studio/tests/integration/test_signal_layers_api.py::test_layer_switches_are_remembered_across_restarts`, `studio/tests/frontend/signalLayers.test.ts::layer switches are remembered by the backend, not the browser` |
| LAYER-10 | Tags sit beside bound equipment, outside IEDs at the edge, and are never saved. | `studio/tests/frontend/signalLayers.test.ts::tags sit beside bound equipment, remote publishers at the edge, and are never saved` |
| LAYER-11 | Observed pulses come only from a real stNum increase. | `studio/tests/frontend/signalLayers.test.ts::observed pulses come only from a real stNum increase` |
| LAYER-12 | At most one pulse per link per second. | `studio/tests/frontend/signalLayers.test.ts::one pulse per link per second; extra changes are coalesced` |
| LAYER-13 | Without a listener, pulses are hollow and say "expected". | `studio/tests/frontend/signalLayers.test.ts::without a listener a publisher change is a hollow expected pulse` |
| LAYER-14 | Reduced motion gives a static highlight; animation uses transform and opacity only. | `studio/tests/frontend/signalLayers.test.ts::reduced motion turns pulses into a static highlight; animation uses transform and opacity only` |
| LAYER-15 | Problems persist as words with a dashed border, never colour alone. | `studio/tests/frontend/signalLayers.test.ts::problems persist as words with a dashed border, never colour alone` |
| LAYER-16 | MMS reads are counted per client per logical node and summarised for the live view only. | `studio/tests/unit/test_mms_access.py::test_reads_are_counted_per_logical_node_and_clients_by_address`, `studio/tests/unit/test_mms_access.py::test_summaries_go_to_the_live_view_only` |
| LAYER-17 | Observing reads never changes the MMS answer; the worker summarises at most once per second. | `studio/tests/unit/test_mms_access.py::test_worker_observes_reads_without_changing_the_answer`, `studio/tests/protocol/test_mms_access_wire.py::test_reads_are_counted_per_client_and_answers_are_unchanged` |
| LAYER-18 | Connection, report and control events carry the client address; ScadaBench's own command is not external. | `studio/tests/protocol/test_mms_access_wire.py::test_report_client_and_control_carry_the_peer`, `studio/tests/unit/test_mms_access.py::test_scadabench_own_command_is_not_an_external_control` |
| LAYER-19 | A client that disconnects leaves the list. | `studio/tests/unit/test_mms_access.py::test_a_client_that_disconnects_is_removed` |

## Acceptance evidence

Change [0094](../changes/2026/0094-goose-mms-signal-layers.md). Private vendor files are
checked only locally (`test_goose_link_map_vendor_files.py`, skipped elsewhere) and described
generically.
