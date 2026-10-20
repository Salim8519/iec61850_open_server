import assert from 'node:assert/strict';
import test from 'node:test';
import {
  applyToRows, formatMeasurement, MEASUREMENT_COPY, moveRow, newMeasurement, numericValue, qualityBad,
  settingFromText, sharedSettings,
} from '../../frontend/src/equipment/measurementDisplay.ts';
import { defaultLabel, kindInfo, setupItems } from '../../frontend/src/equipment/kindInfo.ts';
import { measurementBindings } from '../../frontend/src/equipment/stateBindings.ts';
import { equipmentByIed } from '../../frontend/src/equipment/signalLayout.ts';
import type { EquipmentData, EquipmentNode, Measurement, Signal } from '../../frontend/src/api/types.ts';

const plain = { scale: 1, offset: 0, unit: '', decimals: 2 };
const row = (id: string, extra: Partial<Measurement> = {}): Measurement => ({ id, label: id,
  source: { ied_id: 'ied1', report_ref: null, ref: `IED1LD0/MMXU1.${id}.mag.f` }, ...plain, ...extra });

test('shown value is raw × scale + offset, rounded, with the unit (ADR-0090)', () => {
  assert.deepEqual(formatMeasurement(11000, { scale: 0.001, offset: 0, unit: 'kV', decimals: 2 }),
    { text: '11.00 kV', state: 'value' });
  assert.equal(formatMeasurement(12, { scale: 1, offset: -4, unit: 'mA', decimals: 1 }).text, '8.0 mA');
  assert.equal(formatMeasurement(2.5, { scale: 1000, offset: 0, unit: '', decimals: 0 }).text, '2500');
  assert.equal(formatMeasurement('49.987', { ...plain, unit: 'Hz', decimals: 3 }).text, '49.987 Hz');
  assert.equal(formatMeasurement(-0.001, plain).text, '0.00', 'no "-0.00"');
  assert.equal(formatMeasurement(1.005, { ...plain, decimals: 9 }).text, '1.005000', 'decimals capped at 6');
});

test('no value, stopped IED and non-numbers are said in words, not colour', () => {
  assert.deepEqual(formatMeasurement(undefined, plain), { text: '—', state: 'none' });
  assert.deepEqual(formatMeasurement(null, plain), { text: '—', state: 'none' });
  assert.deepEqual(formatMeasurement(5, plain, true), { text: 'Stopped', state: 'stopped' });
  assert.deepEqual(formatMeasurement(true, plain), { text: 'true', state: 'text' });
  assert.equal(numericValue('abc'), null);
  assert.equal(numericValue(Infinity), null);
  assert.equal(qualityBad(0), false);
  assert.equal(qualityBad(undefined), false);
  assert.equal(qualityBad(64), true);
  assert.ok(MEASUREMENT_COPY.scaleHint.includes('11000 × 0.001 = 11.00 kV'));
});

test('bulk edit changes only the ticked rows and only the given fields', () => {
  const rows = [row('a', { unit: 'V' }), row('b', { offset: 3 }), row('c')];
  const next = applyToRows(rows, new Set(['a', 'c']), { scale: 0.001, unit: 'kV' });
  assert.deepEqual(next.map(item => [item.scale, item.unit, item.offset]), [[0.001, 'kV', 0], [1, '', 3], [0.001, 'kV', 0]]);
  assert.equal(next[1], rows[1], 'unticked rows are kept as they are');
  assert.deepEqual(sharedSettings(next, new Set(['a', 'c'])), { scale: 0.001, offset: 0, unit: 'kV', decimals: 2 });
  assert.deepEqual(sharedSettings(next, new Set(['a', 'b'])), { decimals: 2 }, 'differing fields are left out');
  assert.deepEqual(sharedSettings(next, new Set()), {});
});

test('typed settings are checked with plain sentences', () => {
  assert.deepEqual(settingFromText('scale', '0,001'), { patch: { scale: 0.001 } });
  assert.deepEqual(settingFromText('offset', '-4'), { patch: { offset: -4 } });
  assert.deepEqual(settingFromText('unit', ' kV '), { patch: { unit: 'kV' } });
  assert.deepEqual(settingFromText('decimals', '3'), { patch: { decimals: 3 } });
  assert.match(settingFromText('scale', 'abc').error!, /Scale must be a number/);
  assert.match(settingFromText('scale', '').error!, /Scale must be a number/);
  assert.match(settingFromText('offset', '5e9').error!, /between/);
  assert.match(settingFromText('decimals', '7').error!, /0 to 6/);
  assert.match(settingFromText('decimals', '1.5').error!, /whole number/);
  assert.match(settingFromText('unit', 'x'.repeat(17)).error!, /at most 16/);
});

test('rows reorder within bounds; a new row gets a readable name and default scaling', () => {
  assert.deepEqual(moveRow(['a', 'b', 'c'], 0, 1), ['b', 'a', 'c']);
  const same = ['a', 'b'];
  assert.equal(moveRow(same, 0, -1), same);
  const signal = { ref: 'IED1LD0/MMXU1.PhV.phsA.cVal.mag.f', description: '' } as Signal;
  const binding = { ied_id: 'ied1', report_ref: null, ref: signal.ref };
  assert.deepEqual(newMeasurement(binding, signal, 'IED1', 'r1'),
    { id: 'r1', label: 'LD0/MMXU1.PhV.phsA.cVal.mag.f', source: binding, ...plain });
  assert.equal(newMeasurement(binding, { ...signal, description: 'Phase A voltage' }, 'IED1', 'r2').label, 'Phase A voltage');
});

test('setup asks for at least one measurement; the kind has a friendly name', () => {
  const list = (measurements: Measurement[]): EquipmentData => ({ kind: 'MEASUREMENT_LIST', label: 'M', rotation: 0, measurements });
  assert.deepEqual(setupItems(list([])), [{ label: 'At least one measurement', done: false }]);
  assert.deepEqual(setupItems(list([row('a')])), [{ label: 'At least one measurement', done: true }]);
  assert.equal(kindInfo('MEASUREMENT_LIST').name, 'Measurement list');
  assert.equal(kindInfo('MEASUREMENT_LIST').category, 'instrument');
  assert.equal(defaultLabel('MEASUREMENT_LIST', 2), 'Measurements-2');
  assert.equal(defaultLabel('CB', 1), 'CB-1');
});

test('signal layers see every IED a measurement list reads from', () => {
  const data: EquipmentData = { kind: 'MEASUREMENT_LIST', label: 'M', rotation: 0,
    measurements: [row('a'), { ...row('b'), source: { ied_id: 'ied2', report_ref: null, ref: 'X/MMXU1.A.mag.f' } }] };
  assert.deepEqual(measurementBindings(data).map(item => item.ied_id), ['ied1', 'ied2']);
  const node = { id: 'm1', type: 'equipment', position: { x: 0, y: 0 }, data } as EquipmentNode;
  const bound = equipmentByIed([node]);
  assert.deepEqual([bound.get('ied1'), bound.get('ied2')], [['m1'], ['m1']]);
});
