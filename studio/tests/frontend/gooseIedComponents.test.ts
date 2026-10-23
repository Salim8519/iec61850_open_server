// ADR-0112 / change 0183: GOOSE IED components. Lines go origin -> sender IED component -> receiving
// IED components and never end on equipment; receivers without a component are named, not drawn.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { GooseLink, GooseLinkMap } from '../../frontend/src/api/signalLayers.ts';
import type { EquipmentNode, GooseLinkSetting } from '../../frontend/src/api/types.ts';
import { GOOSE_IEDS_COPY, IED_COPY } from '../../frontend/src/copy/iedComponent.ts';
import { messagesOfRef, leafIndex } from '../../frontend/src/equipment/gooseLeaves.ts';
import { linkActions, linkChoices, linkComplete, storedLink, takesGooseLink, valueWords }
  from '../../frontend/src/equipment/gooseLinkChoices.ts';
import { triggeredOrigins } from '../../frontend/src/equipment/gooseTriggers.ts';
import { watchIedLive } from '../../frontend/src/equipment/iedStatus.ts';
import { createKeyedStore } from '../../frontend/src/equipment/keyedStore.ts';
import { changeGroups } from '../../frontend/src/equipment/signalChanges.ts';
import { SignalLayerStore } from '../../frontend/src/equipment/signalLayerStore.ts';
import { componentBoxes } from '../../frontend/src/equipment/signalLayout.ts';
import { planChanges } from '../../frontend/src/equipment/signalPlan.ts';
import { nextSlot, withIedComponents } from '../../frontend/src/features/diagram/ied/iedNodes.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const POS = 'PUBCTRL/XCBR1.Pos.stVal';
const REF = 'PUBCTRL/LLN0$GO$GoCB01';
const node = (id: string, x: number, y: number, data: Record<string, unknown>) =>
  ({ id, type: 'equipment', position: { x, y }, data: { kind: 'CB', label: id, rotation: 0, ...data } }) as unknown as EquipmentNode;
const ied = (name: string, id: string) => ({ name, ied_id: id, role: 'simulated', in_project: true, remote: false, desc: '', manufacturer: '' });
const link = (id: string, subscriber: string, publisher = 'PUB'): GooseLink => ({ id, gocb_ref: REF, publisher, subscriber,
  block: 'GoCB01', status: 'declared', remote: false, match: 'named', inputs: [], live: null });
const map: GooseLinkMap = { running: true, diagnostics: [], summary: {}, ieds: [ied('PUB', 'p'), ied('SUB1', 's1'), ied('SUB2', 's2')],
  blocks: [{ publisher: 'PUB', ld_inst: 'CTRL', name: 'GoCB01', gocb_ref: REF, go_id: 'g', dataset: 'DS',
    dataset_ref: 'PUBCTRL/LLN0$DS', conf_rev: 1, app_id: 1, mac: '01-0C-CD-01-00-01', status: 'sendable', label: '',
    members: [], member_refs: ['PUBCTRL/XCBR1.Pos', 'PUBCTRL/CSWI1.Pos'] }],
  links: [link('L1', 'SUB1'), link('L2', 'SUB2')] };
const names = { p: 'PUB', s1: 'SUB1', s2: 'SUB2' };
const breaker = node('Q1', 100, 300, { state_mapping: { mode: 'direct', source: { ied_id: 'p', report_ref: null, ref: POS } } });
const card = (id: string, x: number, iedId: string) => node(id, x, 20, { kind: 'IED', ied_id: iedId });
const boxOf = (item: EquipmentNode) => ({ id: item.id, x: item.position.x, y: item.position.y,
  width: item.data.kind === 'IED' ? 200 : 60, height: item.data.kind === 'IED' ? 120 : 60 });

/** One expected change of POS at t=0, planned on a drawing of `drawn` nodes, with `triggers` acted on. */
function plan(drawn: EquipmentNode[], triggers: { key: number; nodeId: string; at: number; iedId?: string; refs?: string[] }[] = []) {
  const store = new SignalLayerStore();
  store.setLayers({ goose: true, readers: false });
  store.setMap(map);
  store.handle({ kind: 'signal', ied_id: 'p', ref: POS, value: 1 }, 0);
  const links = new Map(map.links.map(item => [item.id, item]));
  const groups = changeGroups(store.get().pulses, store.get().signals, links, triggeredOrigins(drawn, triggers));
  const boxes = drawn.map(boxOf);
  return planChanges(groups, boxes, componentBoxes(drawn, boxes, names))[0];
}
const ends = (change: ReturnType<typeof plan>) => change.lines.map(line => line.route.points.at(-1));

test('a line never ends on equipment: without IED components nothing is drawn, the receivers are named', () => {
  const change = plan([breaker], [{ key: 1, nodeId: 'Q1', at: 0 }]);
  assert.equal(change.lines.length, 0);
  assert.equal(change.fly.length, 0);
  assert.deepEqual(change.missing, ['SUB1', 'SUB2']);
  assert.deepEqual(change.chip, { x: 100, y: 296 }, 'the chip still sits at the breaker that was operated');
  assert.deepEqual(change.missingAt, { x: 100, y: 364 });
  assert.equal(GOOSE_IEDS_COPY.alsoChip(change.missing), 'also SUB1, SUB2 · not on drawing');
  assert.equal(GOOSE_IEDS_COPY.alsoChip(['A', 'B', 'C', 'D']), 'also A, B +2 · not on drawing');
});

test('no trigger: the change starts at the sender IED component and fans out to the receivers', () => {
  const change = plan([breaker, card('IP', 100, 'p'), card('IS1', 400, 's1'), card('IS2', 700, 's2')]);
  assert.equal(change.box, null, 'a value change alone never outlines equipment');
  assert.equal(change.sender?.id, 'IP');
  assert.deepEqual(change.chip, { x: 100, y: 16 });
  assert.deepEqual(change.lines.map(line => line.route.points[0]), [{ x: 200, y: 20 }, { x: 200, y: 20 }]);
  assert.deepEqual(ends(change), [{ x: 500, y: 20 }, { x: 800, y: 20 }]);
  assert.deepEqual(change.missing, []);
});

test('sent from the sender component itself, the line simply starts there', () => {
  const drawn = [breaker, card('IP', 100, 'p'), card('IS1', 400, 's1')];
  const change = plan(drawn, [{ key: 1, nodeId: 'IP', at: 0, iedId: 'p', refs: [POS] }]);
  assert.equal(change.box, null);
  assert.equal(change.lines[0].route.points[0].y, 20, 'leaves the top of the sender component');
  assert.deepEqual(change.missing, ['SUB2']);
  assert.deepEqual(change.missingAt, { x: 100, y: 144 }, 'named under the sender component');
});

test('a sender without a component: the line goes from the origin straight to the receivers', () => {
  const change = plan([breaker, card('IS1', 400, 's1'), card('IS2', 700, 's2')], [{ key: 1, nodeId: 'Q1', at: 0 }]);
  assert.equal(change.box?.id, 'Q1');
  assert.equal(change.sender, null);
  assert.deepEqual(ends(change), [{ x: 500, y: 20 }, { x: 800, y: 20 }]);
  assert.ok(change.lines.every(line => line.route.clear), 'never through a box');
});

test('a sender with no place at all flies one dot into each receiving component', () => {
  const change = plan([card('IS1', 400, 's1')]);
  assert.equal(change.chip, null, 'no anchor, no chip');
  assert.equal(change.lines.length, 0);
  assert.deepEqual(change.fly.map(line => [line.subscriber, line.to]), [['SUB1', { x: 500, y: 20 }]]);
  assert.match(change.fly[0].label, /^← PUB · /);
  assert.deepEqual(change.missing, ['SUB2']);
});

test('expected pulses follow the exact leaves from Go: Pos.ctlModel is never part of Pos [ST]', () => {
  const store = new SignalLayerStore();
  store.setMap(map);
  store.handle({ kind: 'signal', ied_id: 'p', ref: 'PUBCTRL/CSWI1.Pos.ctlModel', value: 1 }, 0);
  assert.equal(store.get().pulses.length, 2, 'an older engine without the list keeps the dataset prefix rule');
  const exact = new SignalLayerStore();
  exact.setMap(map);
  exact.setMembers({ ieds: { p: { [REF]: [POS, 'PUBCTRL/CSWI1.Pos.stVal'] } } });
  exact.handle({ kind: 'signal', ied_id: 'p', ref: 'PUBCTRL/CSWI1.Pos.ctlModel', value: 1 }, 0);
  assert.equal(exact.get().pulses.length, 0);
  exact.handle({ kind: 'signal', ied_id: 'p', ref: POS, value: 2 }, 2000);
  assert.deepEqual(exact.get().pulses.map(pulse => pulse.linkId), ['L1', 'L2']);
  const index = leafIndex({ ieds: { p: { [REF]: [POS] } } });
  assert.deepEqual(messagesOfRef(index, 'p', POS), [REF]);
  assert.deepEqual(messagesOfRef(index, 'p', 'PUBCTRL/XCBR1.Pos'), [REF], 'a row naming the data object');
  assert.deepEqual(messagesOfRef(index, 'p', 'PUBCTRL/XCBR1.PosX'), []);
  assert.deepEqual(messagesOfRef(index, 's1', POS), []);
});

test('user actions are kept as triggers for 1.5 s; they are never pulses', () => {
  const store = new SignalLayerStore();
  store.setLayers({ goose: true, readers: false });
  store.setMap(map);
  store.handle({ kind: 'goose_send_requested', node_id: 'Q1', ied_id: 'p', refs: [POS], action: 'open' }, 0);
  store.handle({ kind: 'equipment_test', node_id: 'Q1' }, 10);
  assert.equal(store.get().triggers.length, 2);
  assert.equal(store.get().pulses.length, 0);
  store.prune(5000);
  assert.equal(store.get().triggers.length, 0);
});

test('IED card words: full name, simulated or real, running, and GOOSE counts in words', () => {
  assert.equal(IED_COPY.goose(2, 1), 'Sends 2 messages · gets 1');
  assert.equal(IED_COPY.goose(1, 0), 'Sends 1 message');
  assert.equal(IED_COPY.goose(0, 3), 'Gets 3 messages');
  assert.equal(IED_COPY.goose(0, 0), 'No GOOSE');
  assert.deepEqual(IED_COPY.state('simulated', true, true), { text: 'Simulated · running', tone: 'running' });
  assert.equal(IED_COPY.state('simulated', false, false).text, 'Simulated · stopped');
  assert.equal(IED_COPY.state('real', false, false).text, 'Real device · watch only');
  const cardFile = read('equipment/IedCard.tsx');
  assert.doesNotMatch(cardFile, /Handle|ellipsis/, 'no terminals, nothing cut');
  assert.match(read('equipment/EquipmentNode.tsx'), /<IedCard data=\{data\} \/>/);
  assert.doesNotMatch(read('equipment/ied-card.css'), /ellipsis|nowrap/);
});

test('GOOSE traffic re-renders only the card whose words changed', () => {
  const store = new SignalLayerStore();
  store.setLayers({ goose: true, readers: false });
  store.setMap(map);
  const keyed = createKeyedStore<string>();
  const stop = watchIedLive(store, keyed);
  const calls = { s1: 0, s2: 0 };
  keyed.subscribe('s1', () => calls.s1++);
  keyed.subscribe('s2', () => calls.s2++);
  store.handle({ kind: 'signal', ied_id: 'p', ref: POS, value: 1 }, 0);
  assert.deepEqual(calls, { s1: 0, s2: 0 }, 'a pulse changes no card words');
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'timeout', st_num: 4 }, 100);
  assert.deepEqual(calls, { s1: 1, s2: 1 }, 'both receivers lost the message: each card once');
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'timeout', st_num: 4 }, 200);
  assert.deepEqual(calls, { s1: 1, s2: 1 }, 'the same words notify nobody');
  store.setLayers({ goose: true, readers: true });
  store.setClients({ s1: [{ peer: '10.0.0.5:5000', ip: '10.0.0.5', own: false, this_pc: false, reading: true, reads: [],
    more_nodes: 0, total_reads: 3, writes: 0, rcbs: [], connected_s: 4 }] });
  assert.deepEqual(calls, { s1: 2, s2: 1 }, 'a reader shows on its own card only');
  assert.equal(JSON.parse(keyed.get('s1')!).readers, '◉ 1 read');
  stop();
});

test('"Add all GOOSE IEDs" adds the placements from Go in one edit; one IED goes to the next slot', () => {
  const diagram = { nodes: [card('IP', 0, 'p')], edges: [] };
  const places = [{ ied_id: 'p', name: 'PUB', position: { x: 0, y: -180 } },
    { ied_id: 's1', name: 'SUB1', position: { x: 240, y: -180 } }, { ied_id: 's2', name: 'SUB2', position: { x: 480, y: -180 } }];
  let id = 0;
  const result = withIedComponents(diagram, places, () => `n${++id}`);
  assert.deepEqual(result.ids, ['n1', 'n2'], 'an IED already drawn is skipped');
  assert.deepEqual(result.diagram.nodes.slice(1).map(item => [item.data.kind, item.data.ied_id, item.data.label, item.position.x]),
    [['IED', 's1', 'SUB1', 240], ['IED', 's2', 'SUB2', 480]]);
  assert.deepEqual(nextSlot(places, 's2'), [{ ied_id: 's2', name: 'SUB2', position: { x: 0, y: -180 } }]);
  assert.deepEqual(nextSlot(places, 'zz'), []);
  assert.equal(withIedComponents(diagram, [], () => 'x').diagram, diagram, 'nothing to add: same drawing');
  const page = read('features/diagram/ied/useGooseIeds.ts');
  assert.match(page, /await prepare\(\)/, 'the drawing is saved before Go proposes');
  assert.match(read('features/diagram/AddMenu.tsx'), /GOOSE_IEDS_COPY\.addAll/);
});

test('the GOOSE section offers what Go accepts and sends with the right buttons', () => {
  const leaf = (ref: string, value_type: 'onoff' | 'position' | 'number' | null, sendable = true) =>
    ({ ref, kind: 'x', fc: 'ST', value_type, writable: true, sendable, description: '', value: null });
  const message = { gocb_ref: REF, name: 'GoCB01', dataset: 'DS', go_id: 'g', app_id: 1, conf_rev: 1, subscribers: [], complete: true,
    members: [{ position: 1, text: 'XCBR1.Pos [ST]', fc: 'ST', refs: [leaf('A/XCBR1.Pos.stVal', 'position'), leaf('A/XCBR1.Pos.q', null)] },
      { position: 2, text: 'GGIO1 [ST]', fc: 'ST', refs: [leaf('A/GGIO1.Ind1.stVal', 'onoff'), leaf('A/GGIO1.Ind2.stVal', 'onoff'),
        leaf('A/MMXU1.A.mag.f', 'number'), leaf('A/GGIO1.Ind3.stVal', 'onoff', false)] }] };
  assert.deepEqual(linkChoices(message, 'CB', 'one').map(item => item.ref), ['A/XCBR1.Pos.stVal', 'A/GGIO1.Ind1.stVal', 'A/GGIO1.Ind2.stVal']);
  assert.deepEqual(linkChoices(message, 'CB', 'contacts').map(item => item.ref), ['A/GGIO1.Ind1.stVal', 'A/GGIO1.Ind2.stVal']);
  assert.equal(linkChoices(message, 'CT', 'one').length, 4, 'other equipment may send a number');
  for (const kind of ['JUNCTION', 'INFEED', 'POINT_PANEL', 'IED'] as const) assert.equal(takesGooseLink(kind), false);
  assert.equal(takesGooseLink('TRIP_RELAY'), true);
  const one: GooseLinkSetting = { ied_id: 'p', gocb_ref: REF, mode: 'one', one: 'A/GGIO1.Ind1.stVal', open: null, closed: null };
  assert.deepEqual(linkActions('CB', one, leaf('x', 'onoff')), ['open', 'closed'], 'a switch with one BOOLEAN sends Open/Closed');
  assert.deepEqual(linkActions('PUSH_BUTTON', one, leaf('x', 'onoff')), ['on', 'off']);
  assert.equal(linkActions('CT', one, leaf('x', 'number')), 'number');
  assert.deepEqual(linkActions('DS', { ...one, mode: 'contacts', one: null, open: 'a', closed: 'b' }), ['open', 'closed']);
  assert.equal(linkComplete({ mode: 'contacts', ied_id: 'p', gocb_ref: REF, open: 'a', closed: 'a' }), false, 'two different values');
  assert.equal(linkComplete({ mode: 'one', ied_id: 'p', gocb_ref: REF }), false);
  assert.deepEqual(storedLink({ ...one, open: 'stale' }), one, 'the unused fields are saved as null');
  assert.equal(valueWords({ value_type: 'position', value: 2 }), 'Closed');
  assert.equal(valueWords({ value_type: 'onoff', value: false }), 'Off');
});

test('point panel rows in a GOOSE message carry a GOOSE tag in words', () => {
  const row = read('features/diagram/points/PointRow.tsx');
  assert.match(row, /messagesOfRef\(useContext\(GooseLeaves\)/);
  assert.match(row, /control\?\.status/, 'a command row uses its status value');
  assert.match(read('features/diagram/points/point-dialog.css'), /\.pp-goose \{[^}]*border: 1px solid/);
});
