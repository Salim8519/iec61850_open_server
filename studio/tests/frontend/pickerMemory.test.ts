import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { initialView, rememberView } from '../../frontend/src/components/pickerMemory.ts';

const source = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('a reopened picker continues with the same IED, search, report and page', () => {
  rememberView('test-a', { iedId: 'ied-1', query: 'Beh', reportRef: 'LLN0.RP.urcb01', offset: 100 });
  assert.deepEqual(initialView('test-a'), { iedId: 'ied-1', query: 'Beh', reportRef: 'LLN0.RP.urcb01', offset: 100 });
  assert.deepEqual(initialView('test-a', 'ied-1'), { iedId: 'ied-1', query: 'Beh', reportRef: 'LLN0.RP.urcb01', offset: 100 });
});

test('changing a point on another IED opens that IED, not an old search', () => {
  rememberView('test-b', { iedId: 'ied-1', query: 'Beh', reportRef: '__all__', offset: 50 });
  assert.deepEqual(initialView('test-b', 'ied-2'), { iedId: 'ied-2', query: '', reportRef: '__all__', offset: 0 });
});

test('a first picker preselects the only IED and an unnamed picker remembers nothing', () => {
  assert.equal(initialView('never-used', undefined, 'only').iedId, 'only');
  rememberView(undefined, { iedId: 'x', query: 'q', reportRef: '__all__', offset: 0 });
  assert.equal(initialView(undefined).iedId, '');
});

test('every SLD and Test picker keeps its place and the chosen point stays visible', () => {
  assert.match(source('features/diagram/StateMappingEditor.tsx'), /memoryKey=\{`sld-state-\$\{key\}`\} chosen=\{binding\}/);
  assert.match(source('features/diagram/RelaySignalAdder.tsx'), /memoryKey=/);
  assert.match(source('features/test/TestPoints.tsx'), /memoryKey="test-points"/);
  // Switching connection type no longer clears the saved point; only "Not connected" does.
  assert.match(source('features/diagram/StateMappingEditor.tsx'), /if \(next === 'none'\)/);
});
