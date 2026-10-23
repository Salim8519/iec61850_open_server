import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import test from 'node:test';
import type { GooseLinkMap } from '../../frontend/src/api/signalLayers.ts';
import type { EquipmentNode } from '../../frontend/src/api/types.ts';
import { SignalLayerStore } from '../../frontend/src/equipment/signalLayerStore.ts';
import { componentBoxes, edgePoint, linkCurve, tagScale } from '../../frontend/src/equipment/signalLayout.ts';
import { iedLiveWords } from '../../frontend/src/equipment/iedStatus.ts';
import { EDGE_PULSE_MS, observedLinks, PulseGate, problemWords, reducedMotion } from '../../frontend/src/equipment/signalPulses.ts';
import { linkLine, linkStatus } from '../../frontend/src/copy/signalLayers.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const REF = 'PUBLD0/LLN0$GO$GoCB01';
const map: GooseLinkMap = {
  running: true, diagnostics: [], summary: {},
  ieds: [{ name: 'PUB_BAY0001', ied_id: 'p', role: 'simulated', in_project: true, remote: false, desc: '', manufacturer: '' },
    { name: 'SUB_BAY0002', ied_id: 's', role: 'simulated', in_project: true, remote: false, desc: '', manufacturer: '' }],
  blocks: [{ publisher: 'PUB_BAY0001', ld_inst: 'LD0', name: 'GoCB01', gocb_ref: REF, go_id: 'g', dataset: 'DS',
    dataset_ref: 'PUBLD0/LLN0$DS', conf_rev: 1, app_id: 1, mac: '01-0C-CD-01-00-01', status: 'sendable', label: '',
    members: [], member_refs: ['PUBLD0/XCBR1.Pos.stVal'] }],
  links: [{ id: 'L1', gocb_ref: REF, publisher: 'PUB_BAY0001', subscriber: 'SUB_BAY0002', block: 'GoCB01',
    status: 'declared', remote: false, match: 'named', inputs: [], live: null }],
};

test('layer switches are remembered by the backend, not the browser', () => {
  const hook = read('features/diagram/signals/useSignalLayers.ts');
  assert.match(hook, /sld_goose_layer/);
  assert.match(hook, /sld_readers_layer/);
  assert.match(hook, /setPreferences/);
  assert.doesNotMatch(hook, /localStorage/);
  const prefs = readFileSync(new URL('../../backend/studio_app/projects/preferences.py', import.meta.url), 'utf8');
  assert.match(prefs, /"sld_goose_layer": int, "sld_readers_layer": int/);
});

test('one pulse per link per second; extra changes are coalesced', () => {
  const gate = new PulseGate();
  assert.equal(gate.allow('L1', 0), true);
  assert.equal(gate.allow('L1', 400), false);
  assert.equal(gate.allow('L2', 400), true);
  assert.equal(gate.allow('L1', 1000), true);
  const store = new SignalLayerStore();
  store.setMap(map);
  for (const [st, at] of [[1, 0], [2, 100], [3, 200], [4, 300]] as const) {
    store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'receiving', st_num: st }, at);
  }
  assert.equal(store.get().pulses.length, 1, 'first frame never pulses; three fast changes give one pulse');
});

test('observed pulses come only from a real stNum increase', () => {
  const store = new SignalLayerStore();
  store.setMap(map);
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'receiving', st_num: 5 }, 0);
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'receiving', st_num: 5 }, 2000);
  assert.equal(store.get().pulses.length, 0, 'retransmission with the same stNum is quiet');
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'receiving', st_num: 6 }, 4000);
  const [pulse] = store.get().pulses;
  assert.equal(pulse.kind, 'observed');
  assert.match(pulse.label, /^PUB_BAY0001 → SUB_BAY0002 · /, "full IED names");
});

test('without a listener a publisher change is a hollow expected pulse', () => {
  const store = new SignalLayerStore();
  store.setMap(map);
  store.handle({ kind: 'signal', ied_id: 'p', ref: 'PUBLD0/XCBR1.Pos.stVal', value: 1 }, 0);
  const [pulse] = store.get().pulses;
  assert.equal(pulse.kind, 'expected');
  assert.equal(pulse.label, 'PUB_BAY0001 → SUB_BAY0002 · XCBR1.Pos Open (expected)');
  store.handle({ kind: 'signal', ied_id: 'p', ref: 'PUBLD0/MMXU1.TotW.mag.f', value: 3 }, 5000);
  assert.equal(store.get().pulses.length, 1, 'a value outside the dataset sends no GOOSE');
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'receiving', st_num: 1 }, 6000);
  store.handle({ kind: 'signal', ied_id: 'p', ref: 'PUBLD0/XCBR1.Pos.stVal', value: 2 }, 8000);
  assert.ok(store.get().pulses.every(item => item.kind !== 'expected' || item.start < 8000),
    'once a listener reports, no more expected pulses');
  const css = read('features/diagram/signals/signal-layers.css');
  assert.match(css, /\.pulse-dot\.expected[^}]*fill: (none|var\(--surface\))/);
});

test('reduced motion turns pulses into a static highlight; animation uses transform and opacity only', () => {
  assert.equal(reducedMotion(() => ({ matches: true })), true);
  assert.equal(reducedMotion(() => ({ matches: false })), false);
  const dot = read('features/diagram/signals/PulseDot.tsx');
  assert.match(dot, /reducedMotion\(\)/);
  assert.match(dot, /transform/);
  assert.doesNotMatch(dot, /style\.(left|top|cx|cy)\b/);
  const css = read('features/diagram/signals/signal-layers.css');
  for (const block of css.match(/@keyframes[^{]+\{([^{}]*\{[^}]*\})*[^}]*\}/g) || []) {
    for (const property of block.match(/[a-z-]+(?=\s*:)/g) || []) assert.ok(['transform', 'opacity'].includes(property), property);
  }
  assert.match(css, /prefers-reduced-motion/);
});

test('problems persist on the receiving IED component in words with a dashed border, never colour alone', () => {
  assert.deepEqual(problemWords('C09', { state: 'timeout' }), { text: '⚠ GOOSE lost from C09', tone: 'problem' });
  assert.match(problemWords('C09', { state: 'problem', problem: 'conf_rev_mismatch' })!.text, /revision differs/);
  assert.equal(problemWords('C09', { state: 'receiving' }), null);
  const store = new SignalLayerStore();
  store.setLayers({ goose: true, readers: false });
  store.setMap(map);
  store.handle({ kind: 'goose_status', gocb_ref: REF, state: 'timeout', st_num: 3 }, 0);
  const words = JSON.parse(iedLiveWords(store.get()).s);
  assert.deepEqual(words.problems, [{ text: '⚠ GOOSE lost from PUB_BAY0001', tone: 'problem' }]);
  assert.equal(iedLiveWords(store.get()).p, undefined, 'the sender itself has nothing to say');
  const css = read('equipment/ied-card.css');
  assert.match(css, /\.ied-card-problem \{[^}]*dashed/);
  assert.match(css, /\.ied-card-state\.stopped \{[^}]*dotted/);
  const card = read('equipment/IedCard.tsx');
  assert.match(card, /\{record\.name\}/, 'the card shows the full IED name');
  assert.match(card, /problem\.text/);
});

test('IEDs are drawn only as IED components: no corner marks, no name tags, nothing saved (ADR-0112)', () => {
  const node = (id: string, data: Record<string, unknown>) => ({ id, type: 'equipment', position: { x: 0, y: 0 },
    data: { kind: 'CB', label: id, rotation: 0, ...data } }) as unknown as EquipmentNode;
  const nodes = [node('q1', { control_binding: { ied_id: 's', ref: 'x' } }), node('i1', { kind: 'IED', ied_id: 's' }),
    node('i2', { kind: 'IED', ied_id: 's' }), node('i3', { kind: 'IED', ied_id: null })];
  const boxes = nodes.map((item, index) => ({ id: item.id, x: index * 300, y: 0, width: 200, height: 120 }));
  const components = componentBoxes(nodes, boxes, { s: 'SUB_BAY0002' });
  assert.deepEqual([...components.byId.keys()], ['s'], 'equipment bound to an IED is not its component');
  assert.equal(components.byName.get('SUB_BAY0002')?.id, 'i1', 'one component per IED: the first one counts');
  assert.equal(tagScale(0.1), 3);
  assert.deepEqual(edgePoint({ x: 190, y: 213 }), { x: -30, y: 173 });
  assert.match(linkCurve({ x: 0, y: 0 }, { x: 100, y: 0 }).path, /^M 0 0 Q 50 -25 100 0$/);
  assert.equal(existsSync(new URL('../../frontend/src/features/diagram/signals/IedTag.tsx', import.meta.url)), false);
  assert.doesNotMatch(read('features/diagram/signals/signal-layers.css'), /ied-mark|ied-tag/);
  const overlay = read('features/diagram/signals/SignalOverlay.tsx');
  assert.doesNotMatch(overlay, /updateDiagram|change\(|placeTags|IedTag/);
});

test('a GOOSE from a sender outside the project pulses once beside its receiver for ~2 s', () => {
  const remote = { ...map, links: [{ ...map.links[0], id: 'R1', publisher: 'FAR', status: 'remote' as const, remote: true }] };
  assert.equal(observedLinks(remote, REF, 1, 2).length, 1);
  assert.equal(observedLinks(remote, REF, 2, 2).length, 0, 'no pulse without a new stNum');
  assert.equal(EDGE_PULSE_MS, 2000);
  assert.match(read('features/diagram/signals/FlyLines.tsx'), /edgePoint/);
  assert.match(read('features/diagram/signals/OutsideSenders.tsx'), /OUTSIDE_COPY\.button/, 'listed in the toolbar, not drawn');
});

test('GOOSE status uses plain words instead of "Not watched"', () => {
  const link = { ...map.links[0], live: null };
  assert.equal(linkStatus(link, true).label, 'Not checked yet');  // GOOSE needs no lab network (ADR-0073)
  assert.doesNotMatch(linkStatus(link, true).hint, /lab network|Npcap/);
  assert.equal(linkStatus({ ...link, status: 'remote' }, true).label, 'Outside project');
  assert.equal(linkLine(link, 'SUB_BAY0002'), '← PUB_BAY0001 · GoCB01');
  assert.equal(linkLine(link, 'PUB_BAY0001'), '→ SUB_BAY0002 · GoCB01');
  for (const file of ['copy/signalLayers.ts', 'features/devices/GoosePartners.tsx', 'features/diagram/signals/SignalSheet.tsx']) {
    assert.doesNotMatch(read(file), /Not watched/);
  }
});
