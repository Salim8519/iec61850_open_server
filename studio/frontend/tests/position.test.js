import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import ts from 'typescript';

const source = await readFile(new URL('../src/equipment/position.ts', import.meta.url), 'utf8');
const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { switchPosition, tripRelayState } = await import(
  `data:text/javascript;base64,${Buffer.from(compiled).toString('base64')}`);

test('normalized switch state has IEC double-point interpretation', () => {
  const data = { state_mapping: { mode: 'direct' } };
  assert.deepEqual([0, 1, 2, 3, 99].map(value => switchPosition(data, value)),
    ['Intermediate', 'Open', 'Closed', 'Invalid', 'Invalid']);
  assert.equal(switchPosition(data, undefined), 'Awaiting value');
});
test('unmapped switch state is explicit', () => {
  assert.equal(switchPosition({}, 2), 'Mapping required');
});
test('trip relay remains latched until a reset state is observed', () => {
  assert.equal(tripRelayState(false, true), 'State not mapped');
  assert.equal(tripRelayState(true, undefined), 'Awaiting state');
  assert.equal(tripRelayState(true, false), 'Operated');
  assert.equal(tripRelayState(true, true), 'Reset');
  assert.equal(tripRelayState(true, true, true), 'Operated');
});
