# Spec: SLD signal layers (GOOSE messages, Who is reading)

Decisions: [ADR-0065](../decisions/ADR-0065-sld-signal-layers.md),
[ADR-0066](../decisions/ADR-0066-goose-inputs-and-mms-access-events.md).
Change: [0094](../changes/2026/0094-goose-mms-signal-layers.md); smart lines:
[ADR-0103](../decisions/ADR-0103-goose-smart-lines.md), [0166](../changes/2026/0166-goose-smart-lines.md);
IED components: [ADR-0112](../decisions/ADR-0112-goose-ied-components.md),
[0183](../changes/2026/0183-goose-ied-components.md).
Owner request (2026-10-18): see GOOSE and IEC 61850 client activity on the single-line diagram
without distraction.

## Purpose

Two switches on the SLD show who sends GOOSE to whom and which programs read each simulated
IED, on the IED components the owner places and one dot per real change. Devices shows the same
partners as tables.

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
- IEDs are drawn only as **IED components** (ADR-0112): Go-only kind `IED`, one per IED, a 200 px
  card with the full IED name, simulated/real and running in words, "Sends 2 messages · gets 1" or
  "No GOOSE", no terminals. They are saved diagram nodes. The old "G" corner marks and the left
  name-tag column are gone. IEDs outside the project are not drawn: the toolbar shows "N senders
  outside project ▾" with *Import its CID*; project IEDs that use GOOSE but have no component are
  listed under "N IEDs not on the drawing ▾" with *Add*. Add › *IEDs and GOOSE* › *Add all GOOSE
  IEDs* places every missing one in one row above the drawing (Go decides where), one Undo step.
- Pulse: only when a listener frame raises stNum (never the first frame, never a
  retransmission), at most one per link per second. Publisher changes that land on the same origin
  within 400 ms (a breaker's open and closed contacts) are **one change** (ADR-0103).
- **Origin** (ADR-0112): only a component the user acted on within 1.5 s: a GOOSE send from an
  equipment link or an IED component (exact refs), a point panel row (exact ref), or an
  operate/test/button/tap command on equipment whose own bindings name the changed value. A change
  with no such trigger (generator, script, real IED, outside client) has no origin.
- **Line**: origin → the sender's IED component → every receiving IED component, orthogonal with
  rounded corners, sharing the climb and one lane above the drawing, never through a box (sizes are
  React Flow's measured sizes). No origin: it leaves the top of the sender's component. No sender
  component: from the origin straight to the receivers. A line never ends on equipment. Receivers
  without a component get no line; one dashed chip under the sender (or origin) says
  "also TR5 · not on drawing". A sender with no place at all (outside the project, or no component
  and no trigger) flies one dot from a fixed screen distance into each receiving component for ~2 s.
- **Words**: one chip at the origin, or at the sender's component without one: "Open · sent to 4
  IEDs" (expected, dashed) or "Open · 4 of 4 IEDs got it" (seen); a GOOSE send says what it sent,
  a panel row its name and value ("Line disconnector Open"); switch words come from the backend's
  `equipment_state`. Above a receiving component "✓ got it" only after a real stNum increase, or
  "← PUB · change" for a flying dot, for ~2 s. Each line draws itself behind a ~0.7 s dot, lines of
  one change start 80 ms apart (at most 400 ms), fade by 2 s; at most 16 move at once. A click on a
  line opens that link, on a chip or component flash the publisher's links. Without a listener, a
  publisher value that is one of Go's exact message leaves (`GET /goose/members`, FC-aware) gives a
  hollow dashed pulse labelled "(expected)". Reduced motion: static. Traffic never re-renders
  equipment nodes; a card re-renders only when its own words change (keyed store).
- A subscriber problem is words on the receiving IED component ("⚠ GOOSE lost from C09", dashed
  border); "waiting" (dotted) only once a listener reports. Clicking a component opens its sheet:
  which IED, the messages it sends (values in dataset order with their FC and send buttons while
  running; real devices watch only) and the messages it receives.
- **Sending** (ADR-0112): an IED component's sheet, an equipment *GOOSE* section (`goose_link`:
  IED → message → value, or open and closed as two BOOLEANs on a switch) and point panel rows
  tagged "GOOSE" write through Go in one sample batch, so the message's stNum rises exactly once;
  `goose_send_requested` / `goose_send_applied` / `goose_send_failed` carry one correlation id.
- *Who is reading*: "◉ n read" for external MMS clients on the IED component; hover lists IP,
  nodes read and report blocks; an external control shows one "‹ip› operated ‹object›" line above
  the component for ~2 s.

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
| LAYER-10 | IEDs are drawn only as IED components (one per IED); no corner marks, no name-tag column; outside senders are not drawn; the overlay never changes the diagram (ADR-0112). | `studio/tests/frontend/signalLayers.test.ts::IEDs are drawn only as IED components: no corner marks, no name tags, nothing saved (ADR-0112)` |
| LAYER-11 | Observed pulses come only from a real stNum increase. | `studio/tests/frontend/signalLayers.test.ts::observed pulses come only from a real stNum increase` |
| LAYER-12 | At most one pulse per link per second. | `studio/tests/frontend/signalLayers.test.ts::one pulse per link per second; extra changes are coalesced` |
| LAYER-13 | Without a listener, pulses are hollow and say "expected". | `studio/tests/frontend/signalLayers.test.ts::without a listener a publisher change is a hollow expected pulse` |
| LAYER-14 | Reduced motion gives a static highlight; animation uses transform and opacity only. | `studio/tests/frontend/signalLayers.test.ts::reduced motion turns pulses into a static highlight; animation uses transform and opacity only` |
| LAYER-15 | Problems persist as words on the receiving IED component with a dashed border, never colour alone. | `studio/tests/frontend/signalLayers.test.ts::problems persist on the receiving IED component in words with a dashed border, never colour alone` |
| LAYER-20 | A GOOSE from a sender outside the project is one dot that starts beside its receiver for ~2 s. | `studio/tests/frontend/signalLayers.test.ts::a GOOSE from a sender outside the project pulses once beside its receiver for ~2 s` |
| LAYER-22 | An outside sender's dot starts 130-440 px from its receiver at any zoom; dot, line and words keep their screen size; the receiving IED component names the sender for ~2 s. | `studio/tests/frontend/gooseVisibility.test.ts::an outside sender starts beside its receiver at every zoom, never off the drawing edge`, `studio/tests/frontend/gooseVisibility.test.ts::dot, line and words keep their screen size; the receiving IED component names the sender for ~2 s` |
| LAYER-23 | A receiving sender outside the project shows its live state in GOOSE partners; a silent one says "Outside project". | `studio/tests/frontend/gooseVisibility.test.ts::a receiving sender outside the project shows its live state, a silent one says Outside project` |
| LAYER-24 | Publisher changes on the same symbol within 400 ms are one change, even when the per-link gate silences the second contact. | `studio/tests/frontend/gooseSmartLines.test.ts::both contacts of one breaker inside 400 ms are one change; later or elsewhere is another`, `studio/tests/frontend/gooseSmartLines.test.ts::the store keeps a gated partner contact so one change reads "CB Q1 open · sent to 2 IEDs"` |
| LAYER-25 | A change starts only at a component the user acted on within 1.5 s (exact refs of a GOOSE send or panel row first, else an acted symbol whose own bindings name the value), never at a guessed symbol (ADR-0112). | `studio/tests/frontend/gooseSmartLines.test.ts::a symbol is an origin only when it was acted on; its own bindings say which value`, `studio/tests/frontend/gooseSmartLines.test.ts::a GOOSE send or a panel row names its exact value and wins over a bound symbol`, `studio/tests/frontend/gooseIedComponents.test.ts::user actions are kept as triggers for 1.5 s; they are never pulses` |
| LAYER-26 | Smart lines are orthogonal with rounded corners, share the climb and the lane above the drawing, and never pass through an equipment box. | `studio/tests/frontend/gooseSmartLines.test.ts::a smart line leaves sideways, climbs a free gap to the lane and drops onto the badge, never through a box`, `studio/tests/frontend/gooseSmartLines.test.ts::receivers of one change share the climb and the lane` |
| LAYER-27 | One chip per change: "sent to N IEDs" when expected (never "received"), "k of N IEDs got it" only after a real stNum increase; one line per receiving IED. | `studio/tests/frontend/gooseSmartLines.test.ts::the store keeps a gated partner contact so one change reads "CB Q1 open · sent to 2 IEDs"`, `studio/tests/frontend/gooseSmartLines.test.ts::observed changes count who really got a new stNum; one line per receiver` |
| LAYER-28 | No per-link words on the canvas; a line draws itself behind its dot (paint only) and shows at once with reduced motion. | `studio/tests/frontend/gooseSmartLines.test.ts::the canvas has no per-link words; lines draw behind the dot and respect reduced motion` |
| LAYER-29 | A line never ends on equipment: receivers without an IED component get no line and are named in one dashed "also … · not on drawing" chip. | `studio/tests/frontend/gooseIedComponents.test.ts::a line never ends on equipment: without IED components nothing is drawn, the receivers are named` |
| LAYER-30 | A line runs origin → sender IED component → every receiving component; without an origin, or sent from the sender component itself, it leaves the top of the sender component. | `studio/tests/frontend/gooseSmartLines.test.ts::the plan runs from the symbol through the sender IED component to each receiver component`, `studio/tests/frontend/gooseIedComponents.test.ts::no trigger: the change starts at the sender IED component and fans out to the receivers`, `studio/tests/frontend/gooseIedComponents.test.ts::sent from the sender component itself, the line simply starts there` |
| LAYER-31 | A sender without a component: the line goes from the origin straight to the receiving components; a sender with no place at all flies one dot into each. | `studio/tests/frontend/gooseIedComponents.test.ts::a sender without a component: the line goes from the origin straight to the receivers`, `studio/tests/frontend/gooseIedComponents.test.ts::a sender with no place at all flies one dot into each receiving component` |
| LAYER-32 | Expected pulses follow the exact, functional-constraint-aware leaves Go lists (Pos.ctlModel [CF] is never in Pos [ST]); an older engine keeps the dataset prefix rule. | `studio/tests/frontend/gooseIedComponents.test.ts::expected pulses follow the exact leaves from Go: Pos.ctlModel is never part of Pos [ST]`, `studio/engine/internal/goosemsg/members_test.go::TestMembersExpandWithTheirFunctionalConstraint` |
| LAYER-33 | An IED component card shows the full name, simulated/real, running and its GOOSE in words; traffic re-renders only a card whose words changed. | `studio/tests/frontend/gooseIedComponents.test.ts::IED card words: full name, simulated or real, running, and GOOSE counts in words`, `studio/tests/frontend/gooseIedComponents.test.ts::GOOSE traffic re-renders only the card whose words changed` |
| LAYER-34 | Kind IED and goose_link are Go-only, omittable and byte-identical when absent; a second component for one IED and wrong links are refused in plain words; removing an IED removes its component and links. | `studio/engine/internal/project/ied_component_test.go::TestIEDComponentBytesAndRefusals`, `studio/engine/internal/project/ied_component_test.go::TestGooseLinkBytesAndRefusals`, `studio/engine/internal/project/ied_component_test.go::TestIEDRemovalTakesItsComponentAndLinks` |
| LAYER-35 | One GOOSE send is one sample batch (open and closed together), logged as requested and applied with one correlation id; refusals write and log nothing; the receiver sees stNum rise exactly once. | `studio/engine/internal/runtime/goose_send_test.go::TestGooseSendWritesOneBatchPerAction`, `studio/engine/internal/runtime/goose_send_test.go::TestGooseSendRefusesWithPlainWords`, `studio/tests/protocol/test_goose_send_batch.py::test_two_values_in_one_batch_raise_st_num_once` |
| LAYER-36 | Real devices are watch only; skipped, GOOSE-off and not-running IEDs refuse a send with the reason in words. | `studio/engine/internal/goosemsg/messages_test.go::TestRefusalWords`, `studio/engine/internal/runtime/goose_send_test.go::TestGooseSendRefusesAnIEDThatIsNotRunning` |
| LAYER-37 | "Add all GOOSE IEDs" places every missing GOOSE IED in one row above the drawing (or continuing the component row) in one edit; one IED goes to the next slot. | `studio/engine/internal/goosemsg/proposal_test.go::TestProposeOneRowAboveTheDrawing`, `studio/tests/frontend/gooseIedComponents.test.ts::"Add all GOOSE IEDs" adds the placements from Go in one edit; one IED goes to the next slot` |
| LAYER-38 | The equipment GOOSE section offers only values Go accepts and the right buttons; point panel rows in a message carry a "GOOSE" tag in words. | `studio/tests/frontend/gooseIedComponents.test.ts::the GOOSE section offers what Go accepts and sends with the right buttons`, `studio/tests/frontend/gooseIedComponents.test.ts::point panel rows in a GOOSE message carry a GOOSE tag in words`, `studio/engine/internal/goosemsg/link_test.go::TestLinkCheckRefusals` |
| LAYER-21 | GOOSE status uses plain words ("Not checked yet", "Outside project"), never "Not watched" or "Needs lab network". | `studio/tests/frontend/signalLayers.test.ts::GOOSE status uses plain words instead of "Not watched"` |
| LAYER-16 | MMS reads are counted per client per logical node and summarised for the live view only. | `studio/tests/unit/test_mms_access.py::test_reads_are_counted_per_logical_node_and_clients_by_address`, `studio/tests/unit/test_mms_access.py::test_summaries_go_to_the_live_view_only` |
| LAYER-17 | Observing reads never changes the MMS answer; the worker summarises at most once per second. | `studio/tests/unit/test_mms_access.py::test_worker_observes_reads_without_changing_the_answer`, `studio/tests/protocol/test_mms_access_wire.py::test_reads_are_counted_per_client_and_answers_are_unchanged` |
| LAYER-18 | Connection, report and control events carry the client address; ScadaBench's own command is not external. | `studio/tests/protocol/test_mms_access_wire.py::test_report_client_and_control_carry_the_peer`, `studio/tests/unit/test_mms_access.py::test_scadabench_own_command_is_not_an_external_control` |
| LAYER-19 | A client that disconnects leaves the list. | `studio/tests/unit/test_mms_access.py::test_a_client_that_disconnects_is_removed` |

## Acceptance evidence

Change [0094](../changes/2026/0094-goose-mms-signal-layers.md); visibility between two machines: [0102](../changes/2026/0102-goose-visibility-and-timing.md). IED components and GOOSE sends: [0183](../changes/2026/0183-goose-ied-components.md). Private vendor files are
checked only locally (`test_goose_link_map_vendor_files.py`, skipped elsewhere) and described
generically.
