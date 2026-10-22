// Lane G: own gateway points (made-up values the user controls, served next to the IED points).
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { GatewayPoint } from '../../frontend/src/api/gateway.ts';
import { OWN_COPY, OWN_KIND_LABELS } from '../../frontend/src/copy/gateway.ts';
import { blockedReason } from '../../frontend/src/features/tools/gateway/gatewayCommands.ts';
import { isOwn, liveEdit, ownName, ownTargets, parseOwnNumber, statusChoices, statusNumber, targetName }
  from '../../frontend/src/features/tools/gateway/ownPoints.ts';
import { GATEWAY_POINT_FILTER, gatewayPointMatches } from '../../frontend/src/features/tools/gateway/pointFilters.ts';
import { mergeValues } from '../../frontend/src/features/tools/gateway/pointRows.ts';
import { initialValues } from '../../frontend/src/ui/filterState.ts';

const SRC = new URL('../../frontend/src/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, SRC), 'utf8');

const ied: GatewayPoint = { id: 'bay:X:double', kind: 'double', address: 20001, ied: 'bay', ref: 'X', scale: 1, description: '',
  ied_name: 'Bay', online: true, value_text: 'ON (closed)' };
const lamp: GatewayPoint = { id: 'own:1', kind: 'single', address: 10001, ied: '', ref: '', scale: 1, description: 'Lamp',
  source: 'manual', value: 1, ied_name: 'Own point', online: true, value_text: 'ON' };
const breaker: GatewayPoint = { ...lamp, id: 'own:2', kind: 'double', address: 20002, description: '', value: 2 };
const order: GatewayPoint = { ...lamp, id: 'own:3', kind: 'command_double', address: 50001, description: 'Close it',
  value: undefined, target: 'own:2' };

test('own points are told apart from IED points and named in plain words', () => {
  assert.ok(isOwn(lamp) && !isOwn(ied));
  assert.equal(ownName(lamp), 'Lamp');
  assert.equal(ownName(breaker), 'own:2', 'without a name the id is shown');
  assert.equal(ownName({ ...lamp, id: undefined, description: ' ' }), OWN_COPY.newPoint);
  assert.equal(targetName([lamp, breaker, order], order.target), 'own:2');
  assert.equal(targetName([lamp], undefined), '');
  assert.deepEqual(ownTargets([ied, lamp, breaker, order]).map(choice => choice.value), ['own:1', 'own:2'],
    'only own single and double points can be driven by a command');
});

test('status values are words; the stored number stays the value', () => {
  assert.deepEqual(statusChoices('single').map(choice => choice.label), ['OFF', 'ON']);
  assert.deepEqual(statusChoices('double').map(choice => choice.value), ['1', '2', '0', '3']);
  assert.equal(statusNumber('double', true), 2);
  assert.equal(statusNumber('single', false), 0);
  assert.equal(statusNumber('double', undefined), 1, 'a new position starts open');
});

test('typed numbers are checked before they leave the page', () => {
  assert.deepEqual(parseOwnNumber('12,5', 'float'), { value: 12.5 });
  assert.deepEqual(parseOwnNumber('-3', 'integer'), { value: -3 });
  assert.deepEqual(parseOwnNumber('1.5', 'integer'), { error: OWN_COPY.badWhole });
  assert.deepEqual(parseOwnNumber('abc', 'float'), { error: OWN_COPY.badNumber });
  assert.deepEqual(parseOwnNumber('', 'float'), { error: OWN_COPY.badNumber });
});

test('a saved own point changes at once; a draft or an edited list waits for Save', () => {
  assert.equal(liveEdit(lamp, false), true);
  assert.equal(liveEdit(lamp, true), false);
  assert.equal(liveEdit({ ...lamp, draft: true }, false), false);
  assert.equal(blockedReason({ ...order, draft: true }, false), 'Wait until the point list is saved');
  assert.equal(blockedReason({ ...order, monitor_only: false }, false), '', 'an own command is never a real-IED refusal');
});

test('the "Own points" filter chip and the "Not served" chip know own drafts', () => {
  assert.ok(GATEWAY_POINT_FILTER.toggles?.some(toggle => toggle.key === 'own' && toggle.label === 'Own points'));
  const all = initialValues(GATEWAY_POINT_FILTER);
  assert.ok(gatewayPointMatches(lamp, { ...all, own: true }));
  assert.ok(!gatewayPointMatches(ied, { ...all, own: true }));
  assert.ok(gatewayPointMatches(lamp, { ...all, text: 'own point' }), 'search finds "Own point"');
  assert.ok(gatewayPointMatches({ ...lamp, draft: true }, { ...all, unserved: true }), 'an added own point is not served yet');
  assert.ok(!gatewayPointMatches(lamp, { ...all, unserved: true }));
});

test('live values merge into own rows by id, never into drafts', () => {
  const fresh = [{ ...lamp, value_text: 'OFF' }, { ...breaker, value_text: 'bad state' }];
  const merged = mergeValues([lamp, { ...breaker, draft: true }, ied], fresh);
  assert.equal(merged[0].value_text, 'OFF');
  assert.equal(merged[1].value_text, 'ON', 'a draft keeps what the page shows');
  assert.equal(merged[2].value_text, 'ON (closed)');
});

test('the point list offers Add point next to Propose from IEDs, which keeps own points', () => {
  assert.match(read('features/tools/gateway/PointToolbar.tsx'), /GATEWAY_COPY\.propose[\s\S]*OWN_COPY\.add/);
  assert.match(read('features/tools/gateway/PointTable.tsx'), /gatewayApi\.propose\(protocol, \(rows \?\? \[\]\)\.filter\(isOwn\)\)/);
  assert.match(read('features/tools/gateway/PointRow.tsx'), /<OwnValue row=\{row\}/);
  const cells = read('features/tools/gateway/OwnCells.tsx');
  assert.ok(!/OwnValue[\s\S]*disabled=\{locked\}/.test(cells), 'own values stay changeable while the gateway runs');
  assert.deepEqual(Object.keys(OWN_KIND_LABELS), ['single', 'double', 'float', 'integer', 'command_single', 'command_double']);
  assert.match(OWN_COPY.hint, /made-up value you control/);
  assert.match(OWN_COPY.hint, /SCADA sees it like any other point/);
});
