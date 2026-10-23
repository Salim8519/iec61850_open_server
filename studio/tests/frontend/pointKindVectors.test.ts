// ADR-0113: the browser's kindOf61850 and the Go engine's signalindex pointKind answer every case of
// one shared table, so the picker chips (Go counts) and every list badge (browser) always agree.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { KINDS, kindOf61850, kindOfDataClass } from '../../frontend/src/points/pointKind.ts';

const table = JSON.parse(readFileSync(new URL('../fixtures/point_kinds/iec61850.json', import.meta.url), 'utf8')) as
  { cases: [string, string, string, string][] };

test('kindOf61850 answers every case of the shared IEC 61850 kind table', () => {
  assert.ok(table.cases.length > 700, 'every FC × basic type is in the table');
  for (const [fc, type, dataClass, kind] of table.cases) {
    assert.ok((KINDS as readonly string[]).includes(kind), `${kind} is a kind`);
    assert.equal(kindOf61850(fc, type, dataClass || undefined), kind, `${fc} ${type} ${dataClass}`);
  }
});

test('every standard data class has a kind when a list knows only the class', () => {
  const classes = ['SPS', 'DPS', 'INS', 'ENS', 'ACT', 'ACD', 'SEC', 'BCR', 'HST', 'VSS', 'MV', 'CMV', 'SAV', 'WYE',
    'DEL', 'SEQ', 'HMV', 'HWYE', 'HDEL', 'SPC', 'DPC', 'INC', 'ENC', 'BSC', 'ISC', 'APC', 'BAC', 'SPG', 'ING', 'ENG',
    'ORG', 'TSG', 'CUG', 'VSG', 'ASG', 'CURVE', 'CSG', 'DPL', 'LPL', 'CSD'];
  for (const dataClass of classes) assert.notEqual(kindOfDataClass(dataClass), 'UN', dataClass);
  assert.equal(kindOfDataClass('SEC'), 'CT');
  assert.equal(kindOfDataClass('ASG'), 'CF');
  assert.equal(kindOfDataClass('LPL'), 'TX');
  assert.equal(kindOfDataClass('constructor'), 'UN', 'never a property of the lookup object');
  assert.equal(kindOf61850('CO', 'BOOLEAN', 'constructor'), 'CM');
});
