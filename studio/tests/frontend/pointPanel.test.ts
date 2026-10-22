import assert from 'node:assert/strict';
import test from 'node:test';
import {
  addPoints, MAX_PANEL_POINTS, newCommandPoint, newSignalPoint, nextPointId, patchPoint, pointKeys, pointRef, withIds,
} from '../../frontend/src/equipment/pointPanel.ts';
import {
  displayToRaw, healthyText, isHealthy, kindOfPoint, pointSettingFromText, positionWord, rawToInput, showPoint, summarize,
} from '../../frontend/src/equipment/pointDisplay.ts';
import { chipCounts, filterPoints, groupPoints, listItems } from '../../frontend/src/equipment/pointFilter.ts';
import {
  commandChoices, controlOf, healthySentence, HEALTHY_BODY, pointAvailability, pointRoute, sendBody, setBody,
} from '../../frontend/src/equipment/pointRequests.ts';
import { defaultLabel, kindInfo, setupItems } from '../../frontend/src/equipment/kindInfo.ts';
import type { EquipmentData, Ied, PanelPoint } from '../../frontend/src/api/types.ts';

const plain = { group: '', unit: '', scale: 1, offset: 0, decimals: 2, pinned: false, healthy: null, source: null, control: null };
const signal = (ref: string) => ({ ied_id: 'ied1', report_ref: null, ref });
const point = (id: string, type: PanelPoint['type'], extra: Partial<PanelPoint> = {}): PanelPoint =>
  ({ id, label: `Point ${id}`, type, ...plain, ...(type === 'command' ? { control: { ied_id: 'ied1', ref: `LD0/CSWI${id}.Pos` } }
    : { source: signal(`LD0/GGIO1.${id}.stVal`) }), ...extra });

test('ids count up after the highest one in use, and new points get readable names (ADR-0102)', () => {
  assert.equal(nextPointId([]), 'p1');
  assert.equal(nextPointId([point('p2', 'status'), point('p7', 'status'), point('x', 'status')]), 'p8');
  const bound = signal('IED1LD0/GGIO1.Ind1.stVal');
  const status = newSignalPoint('status', bound, { ref: bound.ref, description: '' }, 'p1', 'IED1');
  assert.deepEqual(status, { id: 'p1', label: 'LD0/GGIO1.Ind1.stVal', type: 'status', ...plain, source: bound });
  assert.equal(newSignalPoint('value', bound, { ref: bound.ref, description: 'Phase A voltage' }, 'p2').label, 'Phase A voltage');
  const command = newCommandPoint('ied1', { ref: 'IED1LD0/CSWI1.Pos' }, 'p3', 'IED1');
  assert.deepEqual([command.type, command.label, command.control, command.source], ['command', 'LD0/CSWI1.Pos', { ied_id: 'ied1', ref: 'IED1LD0/CSWI1.Pos' }, null]);
  assert.equal(pointRef(command), 'IED1LD0/CSWI1.Pos');
  assert.deepEqual([...pointKeys([status, command])], ['status:IED1LD0/GGIO1.Ind1.stVal', 'command:IED1LD0/CSWI1.Pos']);
});

test('rows are added up to 300 and the rest is counted, never dropped silently', () => {
  const held = Array.from({ length: 298 }, (_, index) => point(`p${index + 1}`, 'status'));
  const fresh = withIds(held, [point('', 'status'), point('', 'double'), point('', 'value')]);
  assert.deepEqual(fresh.map(item => item.id), ['p299', 'p300', 'p301']);
  const result = addPoints(held, fresh);
  assert.equal(result.points.length, MAX_PANEL_POINTS);
  assert.equal(result.skipped, 1);
  assert.equal(addPoints(result.points, fresh).skipped, 3);
});

test('patching one row leaves the others as they are', () => {
  const rows = [point('p1', 'status'), point('p2', 'value')];
  const next = patchPoint(rows, 'p2', { pinned: true, unit: 'V' });
  assert.equal(next[0], rows[0]);
  assert.deepEqual([next[1].pinned, next[1].unit], [true, 'V']);
});

test('live values are said in words: ON/OFF, Open/Closed, scaled values, no value, stopped', () => {
  assert.deepEqual(showPoint(point('a', 'status'), true), { text: 'ON', state: 'on' });
  assert.deepEqual(showPoint(point('a', 'status'), false), { text: 'OFF', state: 'off' });
  assert.deepEqual(showPoint(point('d', 'double'), 1), { text: 'Open', state: 'position' });
  assert.deepEqual(showPoint(point('d', 'double'), 2), { text: 'Closed', state: 'position' });
  assert.equal(showPoint(point('d', 'double'), 0).text, 'Intermediate');
  assert.equal(showPoint(point('d', 'double'), 3).text, 'Bad');
  assert.equal(positionWord(9), '9');
  const volts = point('v', 'value', { unit: 'kV', scale: 0.001, decimals: 1 });
  assert.deepEqual(showPoint(volts, 11042.5), { text: '11.0 kV', state: 'value' });
  assert.deepEqual(showPoint(volts, undefined), { text: '—', state: 'none' });
  assert.deepEqual(showPoint(volts, 5, true), { text: 'Stopped', state: 'stopped' });
  assert.deepEqual(showPoint(point('a', 'status'), 7), { text: '7', state: 'text' });
});

test('healthy: equal value, float32 rounding allowed, no value is not healthy', () => {
  const status = point('s', 'status', { healthy: true });
  assert.equal(isHealthy(status, true), true);
  assert.equal(isHealthy(status, false), false);
  assert.equal(isHealthy(status, undefined), false);
  const frequency = point('f', 'value', { healthy: 50, unit: 'Hz' });
  assert.equal(isHealthy(frequency, 50.00000001), true);
  assert.equal(isHealthy(frequency, 49.9), false);
  assert.equal(isHealthy(point('x', 'status'), 'anything'), true, 'a point without a healthy value is never unhealthy');
  assert.equal(healthyText(point('d', 'double', { healthy: 2 })), 'Closed');
  assert.equal(healthyText(point('x', 'status')), '');
});

test('the card counts types, shows pinned values and counts what is not healthy (only while running)', () => {
  const rows = [point('a', 'status', { healthy: true }), point('b', 'status', { healthy: true }), point('c', 'command'),
    point('d', 'double', { healthy: 2 }), point('v', 'value', { pinned: true, unit: 'V', healthy: 230 }), point('w', 'value')];
  const values: Record<string, unknown> = { a: true, b: false, d: undefined, v: 229, w: 1 };
  const summary = summarize(rows, row => values[row.id], true);
  assert.deepEqual(summary.counts, { status: 2, double: 1, command: 1, value: 2 });
  assert.deepEqual(summary.pinned, [{ id: 'v', label: 'Point v', text: '229.00 V', state: 'value' }]);
  assert.deepEqual([summary.withHealthy, summary.notHealthy], [4, 3]);
  const idle = summarize(rows, row => values[row.id], false);
  assert.equal(idle.notHealthy, 0, 'nothing is called unhealthy when the IED does not run');
  assert.equal(idle.pinned[0].text, 'Stopped');
});

test('a typed value becomes the raw value: (typed − offset) ÷ scale, without float noise', () => {
  const volts = { scale: 0.001, offset: 0 };
  assert.deepEqual(displayToRaw(volts, '11.04'), { value: 11040 });
  assert.deepEqual(displayToRaw({ scale: 1, offset: -4 }, '8'), { value: 12 });
  assert.deepEqual(displayToRaw({ scale: 1, offset: 0 }, '49,9'), { value: 49.9 });
  assert.match(displayToRaw(volts, 'abc').error!, /Type a number/);
  assert.match(displayToRaw(volts, '').error!, /Type a number/);
  assert.equal(rawToInput(point('v', 'value', { scale: 0.001, decimals: 1 }), 11042.5), '11.0');
  assert.equal(rawToInput(point('v', 'value'), undefined), '');
});

test('a scale of 0 is refused because the typed value could not be turned back', () => {
  assert.match(pointSettingFromText('scale', '0').error!, /cannot be 0/);
  assert.deepEqual(pointSettingFromText('scale', '0.001'), { patch: { scale: 0.001 } });
  assert.deepEqual(pointSettingFromText('unit', ' kV '), { patch: { unit: 'kV' } });
  assert.match(pointSettingFromText('decimals', '9').error!, /0 to 6/);
});

test('point kinds follow ADR-0095: SP, DP, MV and SC or DC by the control', () => {
  const measured = point('v', 'value', { source: signal('LD0/MMXU1.TotW.mag.f') });
  assert.deepEqual([kindOfPoint(point('a', 'status')), kindOfPoint(point('d', 'double')), kindOfPoint(measured)], ['SP', 'DP', 'MV']);
  // 0178 (B16): an integer read from a status (ST) attribute is an integer status, not a measured value.
  assert.equal(kindOfPoint(point('n', 'value', { source: signal('LD0/XCBR1.OpCnt.stVal') })), 'IS');
  assert.equal(kindOfPoint(point('t', 'value', { source: signal('LD0/YLTC1.TapPos.valWTr.posVal') })), 'IS');
  assert.equal(kindOfPoint(point('1', 'command'), { cdc: 'SPC' }), 'SC');
  assert.equal(kindOfPoint(point('1', 'command'), { cdc: 'DPC' }), 'DC');
  assert.equal(kindOfPoint(point('1', 'command')), 'CM');
});

test('chips count by type; search matches name, group and reference; groups keep first-seen order', () => {
  const rows = [point('a', 'status', { label: 'CB Y130 ready', group: 'CB Y130 close conditions' }), point('b', 'double', { group: 'Bay' }),
    point('c', 'command', { label: 'Close CB', group: 'CB Y130 close conditions' }), point('d', 'status', { label: 'Alarm' })];
  assert.deepEqual(chipCounts(rows), { all: 4, status: 2, double: 1, command: 1, value: 0 });
  assert.deepEqual(filterPoints(rows, 'status', '').map(item => item.id), ['a', 'd']);
  assert.deepEqual(filterPoints(rows, 'all', 'y130 close').map(item => item.id), ['a', 'c']);
  assert.deepEqual(filterPoints(rows, 'all', 'cswic').map(item => item.id), ['c'], 'the reference is searched too');
  assert.deepEqual(filterPoints(rows, 'all', 'nothing here'), []);
  assert.deepEqual(groupPoints(rows).map(item => [item.group, item.points.length]), [['CB Y130 close conditions', 2], ['Bay', 1], ['', 1]]);
  const items = listItems(rows);
  assert.deepEqual(items.map(item => item.kind), ['group', 'point', 'point', 'group', 'point', 'group', 'point']);
  assert.equal(new Set(items.map(item => item.key)).size, items.length, 'keys are unique for the scroller');
});

test('requests: one route, three actions, command choices by control, and the reason a point cannot change', () => {
  assert.equal(pointRoute('node/1'), '/equipment/node%2F1/test');
  const status = point('p1', 'status');
  assert.deepEqual(setBody(status, true), { action: 'set', row: 'p1', value: true });
  assert.deepEqual(sendBody(point('p2', 'command'), false), { action: 'send', row: 'p2', value: false });
  assert.deepEqual(HEALTHY_BODY, { action: 'healthy' });
  assert.deepEqual(commandChoices({ cdc: 'SPC' }).map(item => [item.label, item.value]), [['Send ON', true], ['Send OFF', false]]);
  assert.deepEqual(commandChoices({ cdc: 'DPC' }).map(item => [item.label, item.value]), [['Open', false], ['Close', true]]);
  assert.deepEqual(commandChoices(undefined), []);
  const ied = (role: Ied['role'], id = 'ied1') => ({ id, role, controls: [{ ref: 'LD0/CSWIp2.Pos', cdc: 'DPC' }] }) as unknown as Ied;
  assert.deepEqual(pointAvailability(undefined, new Set()), { ok: false, reason: 'missing' });
  assert.deepEqual(pointAvailability(ied('real'), new Set(['ied1'])), { ok: false, reason: 'real' });
  assert.deepEqual(pointAvailability(ied('off'), new Set(['ied1'])), { ok: false, reason: 'off' });
  assert.deepEqual(pointAvailability(ied('simulated'), new Set()), { ok: false, reason: 'stopped' });
  assert.deepEqual(pointAvailability(ied('simulated'), new Set(['ied1'])), { ok: true, reason: '' });
  assert.equal(controlOf(point('p2', 'command'), ied('simulated'), [])?.cdc, 'DPC');
  assert.equal(controlOf(point('p9', 'command'), ied('simulated'), []), undefined);
  assert.equal(healthySentence({ requested: 3, applied: 3, failed: 0, results: [] }), '3 points set to healthy.');
  assert.match(healthySentence({ requested: 3, applied: 2, failed: 1, results: [] }), /2 set to healthy, 1 could not be set/);
});

test('the kind has a friendly name, a default label and asks for at least one point', () => {
  const data = (points: PanelPoint[]): EquipmentData => ({ kind: 'POINT_PANEL', label: 'P', rotation: 0, ied_id: 'ied1', points });
  assert.equal(kindInfo('POINT_PANEL').name, 'Point panel');
  assert.equal(defaultLabel('POINT_PANEL', 2), 'Points-2');
  assert.deepEqual(setupItems(data([])), [{ label: 'At least one point', done: false }]);
  assert.deepEqual(setupItems(data([point('a', 'status')])), [{ label: 'At least one point', done: true }]);
});
