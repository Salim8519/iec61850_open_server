// ADR-0095, helper A: point types in Devices, Test and the SLD pickers (display and request rules).
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { buildSignalQuery } from '../../frontend/src/components/signalQuery.ts';
import { controlClass, kindOfSignal } from '../../frontend/src/components/signalKind.ts';
import { compareKind, compareMatches, COMPARE_FILTER, LIVE_FILTER, POINT_KIND, SIGNAL_FILTER, signalQuery }
  from '../../frontend/src/features/devices/deviceFilters.ts';
import { swapPoints, TEST_FILTER, testIedChoices, testPointKind, testPointMatches }
  from '../../frontend/src/features/test/testFilters.ts';
import { initialValues } from '../../frontend/src/ui/filterState.ts';

const source = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const query = (changes: Record<string, string | boolean> = {}, spec = SIGNAL_FILTER) =>
  new URLSearchParams(signalQuery({ ...initialValues(spec), ...changes }, 0));

test('Data points send point_kind and helpers=hide only when chosen; type chips replace the FC chips', () => {
  const start = query();
  assert.deepEqual(Object.fromEntries(start), { q: '', offset: '0', kind_counts: '1' }, 'a fresh list only asks for the type counts');
  assert.equal(start.get('point_kind'), null);
  assert.equal(start.get('helpers'), null);
  assert.equal(query({ [POINT_KIND]: 'DP' }).get('point_kind'), 'DP');
  assert.equal(query({ helpers: true }).get('helpers'), 'hide');
  const both = query({ kind: 'measured', [POINT_KIND]: 'MV', helpers: true });
  assert.equal(both.get('fc'), null, 'a remembered old "Kind of data" choice no longer filters');
  assert.equal(both.get('point_kind'), 'MV');
  assert.equal(query({ [POINT_KIND]: 'banana' }).get('point_kind'), null, 'an unknown type is never sent');
  assert.ok(!SIGNAL_FILTER.groups?.some(group => group.key === 'kind'));
  assert.ok(SIGNAL_FILTER.groups?.some(group => group.key === POINT_KIND && group.kinds));
  assert.ok(SIGNAL_FILTER.toggles?.some(toggle => toggle.key === 'helpers'));
});

test('real-IED live values and the SLD picker use the same two request words', () => {
  assert.equal(query({ [POINT_KIND]: 'SC', helpers: true }, LIVE_FILTER).get('point_kind'), 'SC');
  assert.equal(query({ helpers: true }, LIVE_FILTER).get('helpers'), 'hide');
  assert.equal(new URLSearchParams(buildSignalQuery({ pointKind: 'CT' })).get('point_kind'), 'CT');
  assert.equal(new URLSearchParams(buildSignalQuery({ pointKind: 'all' })).get('point_kind'), null);
  assert.equal(new URLSearchParams(buildSignalQuery({})).get('point_kind'), null);
});

test('a signal row gets its kind from FC, basic type, data class or the IED control list', () => {
  const controls = [{ ref: 'LD/XCBR1.Pos', cdc: 'DPC' as const }, { ref: 'LD/CSWI1.Pos', cdc: 'SPC' as const }];
  const signal = (ref: string, fc: string, kind: string) => ({ ref, fc, kind });
  assert.equal(controlClass(controls, 'LD/XCBR1.Pos.Oper.ctlVal'), 'DPC');
  assert.equal(controlClass(controls, 'LD/XCBR10.Pos.Oper.ctlVal'), undefined, 'a longer name is another object');
  assert.equal(kindOfSignal(signal('LD/XCBR1.Pos.Oper.ctlVal', 'CO', 'BOOLEAN')), 'CM');
  assert.equal(kindOfSignal(signal('LD/XCBR1.Pos.Oper.ctlVal', 'CO', 'BOOLEAN'), undefined, controls), 'DC');
  assert.equal(kindOfSignal(signal('LD/CSWI1.Pos.Oper.ctlVal', 'CO', 'BOOLEAN'), undefined, controls), 'SC');
  assert.equal(kindOfSignal(signal('LD/XCBR1.Pos.stVal', 'ST', 'Dbpos'), undefined, controls), 'DP');
  assert.equal(kindOfSignal(signal('LD/MMTR1.SupWh.actVal', 'ST', 'INT64')), 'IS');
  assert.equal(kindOfSignal(signal('LD/MMTR1.SupWh.actVal', 'ST', 'INT64'), () => 'BCR'), 'CT');
  assert.equal(kindOfSignal(signal('LD/MMXU1.TotW.mag.f', 'MX', 'FLOAT32')), 'MV');
});

test('Compare filters by type from the device type, else the SCD type, without changing the old rules', () => {
  const row = { ref: 'LD/MMXU1.TotW.mag.f', fc: 'MX', category: 'type_mismatch', scd_type: 'VisString255', device_type: 'FLOAT32' };
  assert.equal(compareKind(row), 'MV', 'the device type is what clients read');
  assert.equal(compareKind({ ...row, device_type: null }), 'TX', 'without a device type the SCD type decides');
  assert.equal(compareKind({ ...row, scd_type: null, device_type: null }), 'UN');
  const values = { ...initialValues(COMPARE_FILTER), show: 'all' };
  assert.ok(compareMatches(row, values), 'two arguments: no type filter');
  assert.ok(compareMatches(row, { ...values, [POINT_KIND]: 'MV' }, 'MV'));
  assert.ok(!compareMatches(row, { ...values, [POINT_KIND]: 'DP' }, 'MV'));
  assert.ok(!compareMatches({ ...row, category: 'match' }, { ...values, show: 'problems' }, 'MV'));
});

test('Test points filter by name, IED and type; moving swaps with the visible neighbour', () => {
  const signal = { ref: 'LD/GGIO1.Ind1.stVal', fc: 'ST', kind: 'BOOLEAN', value: false, writable: true, quality: 0,
    timestamp: 0, description: '', enums: {} };
  const point = { label: 'Trip alarm', ied_id: 'a', ref: signal.ref, signal };
  const kind = testPointKind(point, { controls: [] });
  assert.equal(kind, 'SP');
  const values = initialValues(TEST_FILTER);
  assert.ok(testPointMatches(point, kind, 'FEED1', values));
  assert.ok(testPointMatches(point, kind, 'FEED1', { ...values, text: 'trip feed1' }));
  assert.ok(!testPointMatches(point, kind, 'FEED1', { ...values, text: 'close' }));
  assert.ok(!testPointMatches(point, kind, 'FEED1', { ...values, ied: 'b' }));
  assert.ok(!testPointMatches(point, kind, 'FEED1', { ...values, point_kind: 'MV' }));
  assert.deepEqual(testIedChoices([{ ied_id: 'a' }, { ied_id: 'a' }], [{ id: 'a', name: 'FEED1' }, { id: 'b', name: 'FEED2' }]),
    [{ value: 'a', label: 'FEED1' }]);
  const ids = [{ id: '1' }, { id: '2' }, { id: '3' }];
  assert.deepEqual(swapPoints(ids, '1', '3').map(item => item.id), ['3', '2', '1']);
  assert.deepEqual(swapPoints(ids, '1', 'gone').map(item => item.id), ['1', '2', '3']);
});

test('every point list of the area shows the kind badge and the shared row edge', () => {
  for (const file of ['features/devices/SignalTable.tsx', 'features/devices/LiveValueTable.tsx',
    'features/devices/ComparisonView.tsx', 'features/test/TestPointRow.tsx']) {
    assert.match(source(file), /kindRow\(/, `${file} marks its rows with the kind edge`);
    assert.match(source(file), /TypeCell|KindBadge/, `${file} shows the badge`);
    assert.match(source(file) + source('features/test/TestPoints.tsx'), /compact-rows/, 'rows are compact');
  }
  for (const file of ['components/signalPicker/SignalPickRow.tsx', 'features/diagram/MeasurementRow.tsx', 'features/diagram/RelayStatusRow.tsx']) {
    assert.match(source(file), /<KindBadge /, `${file} shows the badge`);
    assert.match(source(file), /kindRow\(/, `${file} gets the edge`);
  }
  for (const file of ['features/diagram/RelayStatusRow.tsx', 'features/diagram/MeasurementRow.tsx', 'features/test/TestPointRow.tsx']) {
    assert.match(source(file), /<EditableText /, `${file} renames in place`);
  }
});
