import assert from 'node:assert/strict';
import test from 'node:test';
import { kindInfo, setupItems, stateTone } from '../../frontend/src/equipment/kindInfo.ts';
import type { EquipmentData } from '../../frontend/src/api/types.ts';

const binding = { ied_id: 'i1', report_ref: null, ref: 'IED1LD/XCBR1.Pos.stVal' };
const base = (kind: EquipmentData['kind'], extra: Partial<EquipmentData> = {}): EquipmentData =>
  ({ kind, label: kind, rotation: 0, ...extra });

test('every kind has a friendly name and a category', () => {
  for (const kind of ['CB', 'DS', 'ES', 'CT', 'VT', 'SOURCE', 'RELAY', 'TRIP_RELAY', 'PUSH_BUTTON',
    'JUNCTION', 'TRANSFORMER', 'LOAD', 'BUSBAR'] as const) {
    assert.ok(kindInfo(kind).name.length > 0);
    assert.ok(kindInfo(kind).category.length > 0);
  }
  assert.equal(kindInfo('CB').name, 'Circuit breaker');
});

test('switchgear needs a position and a command', () => {
  assert.deepEqual(setupItems(base('CB')), [
    { label: 'Position', done: false }, { label: 'Command', done: false }]);
  const ready = base('CB', { state_mapping: { mode: 'direct', source: binding },
    control_binding: { ied_id: 'i1', ref: 'IED1LD/XCBR1.Pos', compatibility: 'standard' } });
  assert.ok(setupItems(ready).every(item => item.done));
});

test('push button requirements follow its behavior', () => {
  assert.deepEqual(setupItems(base('PUSH_BUTTON', { push_mode: 'latching_status' })).map(i => i.label), ['Button state']);
  assert.deepEqual(setupItems(base('PUSH_BUTTON', { push_mode: 'spc_momentary' })).map(i => i.label), ['Command']);
});

test('relay needs a signal; junction needs nothing', () => {
  assert.equal(setupItems(base('RELAY'))[0].done, false);
  assert.equal(setupItems(base('RELAY', { relay_statuses: [{ id: 's', label: 'x', source: binding }] }))[0].done, true);
  assert.deepEqual(setupItems(base('JUNCTION')), []);
});

test('canvas state tone separates unmapped, awaiting and live', () => {
  assert.equal(stateTone(false, 5), 'unmapped');
  assert.equal(stateTone(true, undefined), 'awaiting');
  assert.equal(stateTone(true, null), 'awaiting');
  assert.equal(stateTone(true, false), 'live');
  assert.equal(stateTone(true, 0), 'live');
});
