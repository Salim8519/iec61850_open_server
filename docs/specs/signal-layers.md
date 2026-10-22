# Spec: SLD signal layers (GOOSE messages, Who is reading)

Decisions: [ADR-0065](../decisions/ADR-0065-sld-signal-layers.md),
[ADR-0066](../decisions/ADR-0066-goose-inputs-and-mms-access-events.md).
Change: [0094](../changes/2026/0094-goose-mms-signal-layers.md); smart lines:
[ADR-0103](../decisions/ADR-0103-goose-smart-lines.md), [0166](../changes/2026/0166-goose-smart-lines.md).
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
  (shown as "Not checked yet"; GOOSE runs inside ScadaBench in every mode since ADR-0073, so no
  lab network is needed; a sender outside the project shows "Outside project", change 0097); otherwise the listener's state, problem and stNum. A declared link is never
  presented as received without listener evidence.

**SLD** (`features/diagram/signals/`, `equipment/signalLayout.ts`, `signalPulses.ts`,
`signalLayerStore.ts`)
- Two switches, *GOOSE messages* and *Who is reading*, with On/Off words; remembered in the
  backend preferences `sld_goose_layer`, `sld_readers_layer`.
- The SLD stays almost clean (change 0097): on the canvas only equipment and small IED
  marks, only while a switch is on. An IED bound to equipment is one small "G" mark inside the
  top-right corner of the topmost equipment bound to it, so it never covers a neighbour or a wire
  (change 0105); the full IED name and details are on hover and in the sheet on click. Unbound
  project IEDs keep a full-name tag in a small left column outside the drawing; an IED that is not
  running in a partial run says "not running" (dotted). IEDs outside the project get **no tag**:
  the toolbar shows "N senders outside project ▾" next to the switches, a small list with each
  name and *Import its CID*. Tags are never saved. Name tags scale 1–3× against zoom, corner marks 1–1.5×.
- Pulse: only when a listener frame raises stNum (never the first frame, never a
  retransmission), at most one per link per second. Smart lines (ADR-0103): publisher changes that
  land on the same SLD symbol within 400 ms (a breaker's open and closed contacts) are **one change**.
  It starts at that symbol (the equipment whose state, command, relay or measurement binding names
  the changed reference; else the publisher's IED mark), which gets a brief outline, and one plain
  chip above it: "Open · sent to 4 IEDs" (expected, dashed) or "Open · 4 of 4 IEDs got it" (seen).
  Switch words come from the backend's `equipment_state`, never computed in the browser. One
  orthogonal line with rounded corners goes to each receiving IED's mark: sideways into the nearest
  free gap, up to one shared lane above the drawing, along it and down the receiver's free gap,
  never through an equipment box. Each line draws itself behind a ~0.7 s dot, lines of one change
  start 80 ms apart (at most 400 ms), then fade by 2 s; at most 16 lines move at once. The receiver
  mark says "✓ got it" only after a real stNum increase. No per-link words on the canvas; a click
  on a line still opens that link, a click on the chip the publisher's links. A GOOSE from a sender outside the project moves one
  dot from a fixed screen distance beside the receiver tag into it for ~2 s, with a fading line,
  then nothing remains. Dot, line and label scale like the tags, so they stay readable when
  zoomed out, and the receiver tag shows "← PUB · change" for ~2 s (ADR-0070). Without a listener, a publisher value that is a dataset member of
  a sent block gives a hollow dashed pulse labelled "(expected)". Reduced motion: static.
- A subscriber problem is one small ⚠ on the receiver's tag (dashed border; words on hover);
  "waiting" only once a listener reports. Clicking a tag opens the side sheet with one line per
  link, e.g. "← C11_Y120 · GoCB01 · Receiving", the reason on hover. Clicking a link opens its details.
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
| LAYER-10 | A bound IED is a corner mark inside its equipment box (full name on hover); unbound project IEDs get a full-name tag left of the drawing; outside senders get no tag; tags are never saved. | `studio/tests/frontend/signalLayers.test.ts::a bound IED is a corner mark inside its equipment, outside senders get no tag, and tags are never saved (changes 0097, 0105)` |
| LAYER-11 | Observed pulses come only from a real stNum increase. | `studio/tests/frontend/signalLayers.test.ts::observed pulses come only from a real stNum increase` |
| LAYER-12 | At most one pulse per link per second. | `studio/tests/frontend/signalLayers.test.ts::one pulse per link per second; extra changes are coalesced` |
| LAYER-13 | Without a listener, pulses are hollow and say "expected". | `studio/tests/frontend/signalLayers.test.ts::without a listener a publisher change is a hollow expected pulse` |
| LAYER-14 | Reduced motion gives a static highlight; animation uses transform and opacity only. | `studio/tests/frontend/signalLayers.test.ts::reduced motion turns pulses into a static highlight; animation uses transform and opacity only` |
| LAYER-15 | Problems persist as a small ⚠ with words on hover and a dashed border, never colour alone. | `studio/tests/frontend/signalLayers.test.ts::problems persist as a small ⚠ with words on hover and a dashed border, never colour alone` |
| LAYER-20 | A GOOSE from a sender outside the project is one dot that starts beside its receiver for ~2 s. | `studio/tests/frontend/signalLayers.test.ts::a GOOSE from a sender outside the project pulses once beside its receiver for ~2 s` |
| LAYER-22 | An outside sender's dot starts 130-440 px from its receiver at any zoom; dot, line and words keep their screen size; the receiver tag names the sender for ~2 s. | `studio/tests/frontend/gooseVisibility.test.ts::an outside sender starts beside its receiver at every zoom, never off the drawing edge`, `studio/tests/frontend/gooseVisibility.test.ts::dot, line and words keep their screen size; the receiver tag names the sender for ~2 s` |
| LAYER-23 | A receiving sender outside the project shows its live state in GOOSE partners; a silent one says "Outside project". | `studio/tests/frontend/gooseVisibility.test.ts::a receiving sender outside the project shows its live state, a silent one says Outside project` |
| LAYER-24 | Publisher changes on the same symbol within 400 ms are one change, even when the per-link gate silences the second contact. | `studio/tests/frontend/gooseSmartLines.test.ts::both contacts of one breaker inside 400 ms are one change; later or elsewhere is another`, `studio/tests/frontend/gooseSmartLines.test.ts::the store keeps a gated partner contact so one change reads "CB Q1 open · sent to 2 IEDs"` |
| LAYER-25 | A change starts at the symbol whose own bindings name the changed reference (state over command over panel rows), else at the publisher's IED mark. | `studio/tests/frontend/gooseSmartLines.test.ts::a symbol is found from its own bindings; state beats command beats panel rows`, `studio/tests/frontend/gooseSmartLines.test.ts::the plan starts at the symbol, staggers lines a little and caps the moving ones` |
| LAYER-26 | Smart lines are orthogonal with rounded corners, share the climb and the lane above the drawing, and never pass through an equipment box. | `studio/tests/frontend/gooseSmartLines.test.ts::a smart line leaves sideways, climbs a free gap to the lane and drops onto the badge, never through a box`, `studio/tests/frontend/gooseSmartLines.test.ts::receivers of one change share the climb and the lane` |
| LAYER-27 | One chip per change: "sent to N IEDs" when expected (never "received"), "k of N IEDs got it" only after a real stNum increase; one line per receiving IED. | `studio/tests/frontend/gooseSmartLines.test.ts::the store keeps a gated partner contact so one change reads "CB Q1 open · sent to 2 IEDs"`, `studio/tests/frontend/gooseSmartLines.test.ts::observed changes count who really got a new stNum; one line per receiver` |
| LAYER-28 | No per-link words on the canvas; a line draws itself behind its dot (paint only) and shows at once with reduced motion. | `studio/tests/frontend/gooseSmartLines.test.ts::the canvas has no per-link words; lines draw behind the dot and respect reduced motion` |
| LAYER-21 | GOOSE status uses plain words ("Not checked yet", "Outside project"), never "Not watched" or "Needs lab network". | `studio/tests/frontend/signalLayers.test.ts::GOOSE status uses plain words instead of "Not watched"` |
| LAYER-16 | MMS reads are counted per client per logical node and summarised for the live view only. | `studio/tests/unit/test_mms_access.py::test_reads_are_counted_per_logical_node_and_clients_by_address`, `studio/tests/unit/test_mms_access.py::test_summaries_go_to_the_live_view_only` |
| LAYER-17 | Observing reads never changes the MMS answer; the worker summarises at most once per second. | `studio/tests/unit/test_mms_access.py::test_worker_observes_reads_without_changing_the_answer`, `studio/tests/protocol/test_mms_access_wire.py::test_reads_are_counted_per_client_and_answers_are_unchanged` |
| LAYER-18 | Connection, report and control events carry the client address; ScadaBench's own command is not external. | `studio/tests/protocol/test_mms_access_wire.py::test_report_client_and_control_carry_the_peer`, `studio/tests/unit/test_mms_access.py::test_scadabench_own_command_is_not_an_external_control` |
| LAYER-19 | A client that disconnects leaves the list. | `studio/tests/unit/test_mms_access.py::test_a_client_that_disconnects_is_removed` |

## Acceptance evidence

Change [0094](../changes/2026/0094-goose-mms-signal-layers.md); visibility between two machines: [0102](../changes/2026/0102-goose-visibility-and-timing.md). Private vendor files are
checked only locally (`test_goose_link_map_vendor_files.py`, skipped elsewhere) and described
generically.
