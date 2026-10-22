// ADR-0103 / change 0166: one GOOSE change is one chip at its symbol and one smart line per receiving IED.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { GooseLink, GooseLinkMap } from '../../frontend/src/api/signalLayers.ts';
import type { EquipmentNode } from '../../frontend/src/api/types.ts';
import { changeChip, GOT_IT } from '../../frontend/src/copy/signalLayers.ts';
import { changeGroups, coalesceSignals } from '../../frontend/src/equipment/signalChanges.ts';
import { SignalLayerStore, type Pulse } from '../../frontend/src/equipment/signalLayerStore.ts';
import { equipmentByIed, placeTags, tagCentre, type Box } from '../../frontend/src/equipment/signalLayout.ts';
import { originIndex, refMatches, symbolName } from '../../frontend/src/equipment/signalOrigin.ts';
import { animatedLines, MAX_STAGGER_MS, planChanges } from '../../frontend/src/equipment/signalPlan.ts';
import { crossings, laneY, roundedPath, routeFan, routeLine } from '../../frontend/src/equipment/signalRoute.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const OPEN = 'PUBCTRL/GGIO1.Ind1.stVal';
const CLOSED = 'PUBCTRL/GGIO1.Ind2.stVal';
const REF = 'PUBCTRL/LLN0$GO$GoCB01';
const binding = (ref: string) => ({ ied_id: 'p', report_ref: null, ref });
const node = (id: string, x: number, y: number, data: Record<string, unknown>) =>
  ({ id, type: 'equipment', position: { x, y }, measured: { width: 60, height: 60 },
    data: { kind: 'CB', label: id, rotation: 0, ...data } }) as unknown as EquipmentNode;
const nodes = [
  node('Q1', 100, 300, { state_mapping: { mode: 'contacts_to_dbpos', open: binding(OPEN), closed: binding(CLOSED) } }),
  node('Q2', 100, 180, { kind: 'DS', control_binding: { ied_id: 'p', ref: 'PUBCTRL/CSWI1.Pos' } }),
  node('Q3', 300, 300, { state_mapping: { mode: 'direct', source: { ied_id: 's1', report_ref: null, ref: 'SUBCTRL/XCBR1.Pos.stVal' } } }),
  node('Q4', 500, 300, { kind: 'RELAY', relay_statuses: [{ id: 'r', label: 'Trip', source: { ied_id: 's2', report_ref: null, ref: 'SUBPROT/PTRC1.Tr.general' } }] }),
];
const ied = (name: string, id: string) => ({ name, ied_id: id, role: 'simulated', in_project: true, remote: false, desc: '', manufacturer: '' });
const link = (id: string, subscriber: string): GooseLink => ({ id, gocb_ref: REF, publisher: 'PUB_BAY0001', subscriber, block: 'GoCB01',
  status: 'declared', remote: false, match: 'named', inputs: [], live: null });
const map: GooseLinkMap = { running: true, diagnostics: [], summary: {},
  ieds: [ied('PUB_BAY0001', 'p'), ied('SUB_BAY0002', 's1'), ied('SUB_BAY0003', 's2')],
  blocks: [{ publisher: 'PUB_BAY0001', ld_inst: 'CTRL', name: 'GoCB01', gocb_ref: REF, go_id: 'g', dataset: 'DS',
    dataset_ref: 'PUBCTRL/LLN0$DS', conf_rev: 1, app_id: 1, mac: '01-0C-CD-01-00-01', status: 'sendable', label: '',
    members: [], member_refs: [OPEN, CLOSED] }],
  links: [link('L1', 'SUB_BAY0002'), link('L2', 'SUB_BAY0003')] };

test('a symbol is found from its own bindings; state beats command beats panel rows', () => {
  const originOf = originIndex(nodes);
  assert.equal(refMatches('LD/XCBR1.Pos', 'LD/XCBR1.Pos.stVal'), true);
  assert.equal(refMatches('LD/XCBR1.Pos', 'LD/XCBR1.PosX.stVal'), false, 'only whole name parts match');
  assert.deepEqual(originOf('p', OPEN), { nodeId: 'Q1', role: 'open', kind: 'CB', label: 'Q1' });
  assert.equal(originOf('p', CLOSED)?.role, 'closed');
  assert.equal(originOf('p', 'PUBCTRL/CSWI1.Pos.stVal')?.nodeId, 'Q2');
  assert.equal(originOf('s2', 'SUBPROT/PTRC1.Tr.general')?.row, 'Trip');
  assert.equal(originOf('p', 'PUBCTRL/GGIO9.Ind9.stVal'), null, 'unbound signal: fall back to the IED mark');
  assert.equal(symbolName({ kind: 'CB', label: 'Y130' }), 'CB Y130');
  assert.equal(symbolName({ kind: 'CB', label: 'CB-2' }), 'CB-2');
  assert.equal(symbolName({ kind: 'TRIP_RELAY', label: 'K1' }), 'Trip relay K1');
});

test('both contacts of one breaker inside 400 ms are one change; later or elsewhere is another', () => {
  const sig = (key: number, ref: string, start: number, publisher = 'PUB_BAY0001') =>
    ({ key, iedId: 'p', publisher, ref, value: true, text: ref, start, linkIds: ['L1', 'L2'] });
  const groups = coalesceSignals([sig(1, OPEN, 0), sig(2, CLOSED, 150), sig(3, OPEN, 900),
    sig(4, 'PUBCTRL/CSWI1.Pos.stVal', 100), sig(5, OPEN, 50, 'OTHER_BAY0009')], originIndex(nodes));
  assert.deepEqual(groups.map(group => group.signals.map(signal => signal.key)), [[1, 2], [5], [4], [3]]);
});

test('the store keeps a gated partner contact so one change reads "CB Q1 open · sent to 2 IEDs"', () => {
  const store = new SignalLayerStore();
  store.setLayers({ goose: true, readers: false });
  store.setMap(map);
  store.handle({ kind: 'signal', ied_id: 'p', ref: OPEN, value: true }, 0);
  store.handle({ kind: 'signal', ied_id: 'p', ref: CLOSED, value: false }, 120);
  store.handle({ kind: 'equipment_state', node_id: 'Q1', value: 1 }, 130);
  const state = store.get();
  assert.equal(state.pulses.length, 2, 'the second contact is gated: still one pulse per link');
  assert.equal(state.signals.length, 2, 'but both contacts are kept for the wording');
  assert.ok(state.pulses.every(pulse => pulse.cause === state.signals[0].key));
  const links = new Map(map.links.map(item => [item.id, item]));
  const [group] = changeGroups(state.pulses, state.signals, links, originIndex(nodes));
  assert.equal(group.origin?.nodeId, 'Q1');
  assert.deepEqual(group.receivers, ['SUB_BAY0002', 'SUB_BAY0003']);
  const chip = changeChip(group, state.equipment);
  assert.equal(chip.text, 'Open · sent to 2 IEDs');
  assert.equal(chip.full, 'CB Q1 open');
  assert.doesNotMatch(chip.text + chip.title, /received|got it/, 'expected never claims arrival');
});

test('observed changes count who really got a new stNum; one line per receiver', () => {
  const store = new SignalLayerStore();
  store.setLayers({ goose: true, readers: false });
  store.setMap(map);
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'receiving', st_num: 1 }, 0);
  store.handle({ kind: 'signal', ied_id: 'p', ref: OPEN, value: false }, 2000);
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'receiving', st_num: 2 }, 2030);
  const state = store.get();
  const links = new Map(map.links.map(item => [item.id, item]));
  const twice: Pulse[] = [...state.pulses, { ...state.pulses[0], key: 999, kind: 'expected' }];
  const [group] = changeGroups(twice, state.signals, links, originIndex(nodes));
  assert.equal(group.kind, 'observed');
  assert.equal(group.lines.length, 2, 'a second pulse to the same IED adds no line');
  assert.match(changeChip(group).text, /· 2 of 2 IEDs got it$/);
  assert.equal(GOT_IT, '✓ got it');
  const loose = changeGroups([{ ...state.pulses[0], cause: undefined, change: 'stNum 2' }], [], links, originIndex(nodes));
  assert.equal(changeChip(loose[0]).text, 'New state · 1 of 1 IED got it', 'only a stNum: plain words');
});

const boxes: Box[] = [
  { id: 'Q1', x: 100, y: 300, width: 60, height: 60 }, { id: 'Q2', x: 90, y: 180, width: 80, height: 60 },
  { id: 'Q3', x: 300, y: 300, width: 60, height: 60 }, { id: 'Q5', x: 290, y: 180, width: 80, height: 60 },
  { id: 'Q6', x: 172, y: 260, width: 20, height: 120 },
];

test('a smart line leaves sideways, climbs a free gap to the lane and drops onto the badge, never through a box', () => {
  const lane = laneY(boxes);
  assert.equal(lane, 180 - 36);
  const badge = { point: { x: 351, y: 313 }, box: boxes[2] };
  const route = routeLine({ point: { x: 130, y: 330 }, box: boxes[0], edge: true }, badge, boxes);
  assert.equal(route.clear, true);
  assert.deepEqual(crossings(route.points, boxes, ['Q1', 'Q3']), []);
  assert.deepEqual(route.points[0], { x: 100, y: 330 }, 'starts on the side away from the post beside it');
  assert.ok(route.points.some(point => point.y === lane), 'runs along the shared lane');
  assert.ok(route.points.every((point, index) => !index || point.x === route.points[index - 1].x
    || point.y === route.points[index - 1].y), 'orthogonal');
  assert.deepEqual(route.points.at(-1), badge.point);
  const corners = route.points.length - 2;
  assert.equal(route.path.match(/ Q /g)?.length, corners, 'every corner is rounded');
  assert.match(roundedPath([{ x: 0, y: 0 }, { x: 0, y: -40 }, { x: 40, y: -40 }]), /^M 0 0 L 0 -32 Q 0 -40 8 -40 L 40 -40$/);
  assert.match(roundedPath([{ x: 0, y: 0 }, { x: 0, y: -6 }, { x: 40, y: -6 }]), /Q 0 -6 3 -6/, 'radius never exceeds half a leg');
  assert.deepEqual(route.at(1), badge.point);
});

test('receivers of one change share the climb and the lane', () => {
  const ends = [{ point: { x: 351, y: 313 }, box: boxes[2] }, { point: { x: -40, y: 200 }, box: null }];
  const [a, b] = routeFan({ point: { x: 130, y: 330 }, box: boxes[0], edge: true }, ends, boxes);
  assert.deepEqual(a.points.slice(0, 3), b.points.slice(0, 3), 'one shared trunk up to the lane');
  assert.equal(a.points[2].y, laneY(boxes, 36, ends.map(end => end.point)));
  for (const route of [a, b]) assert.equal(route.clear, true);
});

test('the plan starts at the symbol, staggers lines a little and caps the moving ones', () => {
  const store = new SignalLayerStore();
  store.setLayers({ goose: true, readers: false });
  store.setMap(map);
  store.handle({ kind: 'signal', ied_id: 'p', ref: OPEN, value: true }, 0);
  const links = new Map(map.links.map(item => [item.id, item]));
  const groups = changeGroups(store.get().pulses, store.get().signals, links, originIndex(nodes));
  const drawn = nodes.map(item => ({ id: item.id, x: item.position.x, y: item.position.y, width: 60, height: 60 }));
  const placed = placeTags(drawn, equipmentByIed(nodes), map.ieds.map(item => ({ name: item.name, iedId: item.ied_id,
    outside: false, remote: false })));
  const centre = new Map(placed.map(tag => [tag.name, tagCentre(tag, 1, 1)]));
  const [plan] = planChanges(groups, drawn, placed, centre);
  assert.equal(plan.box?.id, 'Q1', 'origin = the breaker, not the IED corner mark');
  assert.deepEqual(plan.chip, { x: 100, y: 296 });
  assert.deepEqual(plan.lines.map(line => line.start), [0, 80]);
  assert.ok(plan.lines.every(line => line.start <= MAX_STAGGER_MS && line.route.clear));
  assert.equal(animatedLines([plan], 1).size, 1);
});

test('the canvas has no per-link words; lines draw behind the dot and respect reduced motion', () => {
  const overlay = read('features/diagram/signals/SignalOverlay.tsx');
  assert.doesNotMatch(overlay, /signal-link-label|pulse\.label\}<\/text>/);
  assert.match(overlay, /edgePoint/, 'senders outside the project keep their dot beside the receiver');
  const line = read('features/diagram/signals/RouteLine.tsx');
  assert.match(line, /reducedMotion\(\)/);
  assert.match(line, /strokeDashoffset/);
  assert.doesNotMatch(line, /style\.(left|top|cx|cy|width|height)\b/);
  const css = read('features/diagram/signals/signal-layers.css');
  assert.match(css, /\.signal-route\.expected \.signal-route-line \{[^}]*stroke-dasharray/);
  assert.match(css, /\.signal-chip\.expected \{[^}]*dashed/);
  assert.match(css, /prefers-reduced-motion[\s\S]*\.signal-route,[^}]*animation: none/);
  assert.doesNotMatch(css, /\.signal-chip[^{]*\{[^}]*(ellipsis|max-width)/, 'chip words are never cut');
});
