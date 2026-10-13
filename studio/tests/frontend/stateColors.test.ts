import assert from 'node:assert/strict';
import test from 'node:test';
import { colorScheme, defaultScheme, paintFor, stateFill, textOn, validHex } from '../../frontend/src/equipment/stateColors.ts';
import type { EquipmentData } from '../../frontend/src/api/types.ts';

const data = (kind: EquipmentData['kind'], extra: Partial<EquipmentData> = {}): EquipmentData =>
  ({ kind, label: kind, rotation: 0, ...extra });

test('switchgear defaults: closed green, open white', () => {
  const scheme = colorScheme(data('CB'))!;
  assert.equal(scheme.active, '#16a34a');
  assert.equal(scheme.inactive, '#ffffff');
  assert.equal(scheme.activeLabel, 'Closed');
  assert.equal(scheme.inactiveLabel, 'Open');
});

test('object colours override defaults; invalid ones fall back', () => {
  assert.equal(colorScheme(data('DS', { active_color: '#ff0000' }))!.active, '#ff0000');
  assert.equal(colorScheme(data('DS', { active_color: 'red' }))!.active, '#16a34a');
  assert.equal(validHex('#12345'), null);
});

test('only two-state objects get a scheme', () => {
  for (const kind of ['CT', 'VT', 'SOURCE', 'RELAY', 'JUNCTION'] as const) assert.equal(defaultScheme(data(kind)), null);
  assert.equal(defaultScheme(data('PUSH_BUTTON', { push_mode: 'spc_momentary' })), null);
  assert.equal(defaultScheme(data('TRIP_RELAY'))!.active, '#dc2626');
});

test('fill follows the confirmed switch position only', () => {
  assert.equal(stateFill(data('CB'), 'Closed', 2, true), 'active');
  assert.equal(stateFill(data('CB'), 'Open', 1, true), 'inactive');
  assert.equal(stateFill(data('CB'), 'Intermediate', 0, true), 'intermediate');
  assert.equal(stateFill(data('CB'), 'Invalid', 3, true), 'invalid');
  assert.equal(stateFill(data('CB'), 'Awaiting value', undefined, true), null);
  assert.equal(stateFill(data('CB'), 'Closed', 2, false), null);
});

test('trip relay fill respects state polarity', () => {
  assert.equal(stateFill(data('TRIP_RELAY'), '', true, true), 'active');
  assert.equal(stateFill(data('TRIP_RELAY', { trip_state_polarity: 'operated_false' }), '', true, true), 'inactive');
  assert.equal(stateFill(data('TRIP_RELAY'), '', 5, true), 'invalid');
  assert.equal(stateFill(data('TRIP_RELAY'), '', undefined, true), null);
});

test('push button: pressed/latched is the active colour', () => {
  assert.equal(stateFill(data('PUSH_BUTTON', { push_mode: 'latching_status' }), '', true, true), 'active');
  assert.equal(stateFill(data('PUSH_BUTTON', { push_mode: 'latching_status' }), '', false, true), 'inactive');
});

test('text colour picks the higher contrast', () => {
  assert.equal(textOn('#ffffff'), '#112b46');
  assert.equal(textOn('#dc2626'), '#ffffff');
  assert.equal(textOn('#000000'), '#ffffff');
});

test('painted parts keep a visible outline, also when the colour is white', () => {
  assert.deepEqual(paintFor('#ffffff'), { fill: '#ffffff', outline: '#475569', ink: '#112b46' });
  const green = paintFor('#16a34a');
  assert.equal(green.fill, '#16a34a');
  assert.equal(green.outline, '#0e6a30');
});
