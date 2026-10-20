// Select mode in point tables: tick rows, tick all shown (every page), delete only rows still there.
import assert from 'node:assert/strict';
import test from 'node:test';
import { plural, stillThere, toggleShown, togglePick } from '../../frontend/src/ui/pickRows.ts';

test('ticking a row twice clears it again', () => {
  const once = togglePick(new Set(), 'a');
  assert.deepEqual([...once], ['a']);
  assert.deepEqual([...togglePick(once, 'a')], []);
});

test('the header box ticks every shown row, keeps rows ticked outside the filter, and clears when all are ticked', () => {
  const ticked = toggleShown(new Set(['x']), ['a', 'b', 'c']);
  assert.deepEqual([...ticked].sort(), ['a', 'b', 'c', 'x']);
  assert.deepEqual([...toggleShown(ticked, ['a', 'b', 'c'])], ['x']);
  assert.deepEqual([...toggleShown(new Set(), [])], []);
});

test('rows removed elsewhere drop out of what gets deleted, in table order', () => {
  assert.deepEqual(stillThere(new Set(['c', 'a', 'gone']), ['a', 'b', 'c']), ['a', 'c']);
});

test('delete buttons count in plain words', () => {
  assert.equal(plural(1, 'point'), '1 point');
  assert.equal(plural(4647, 'point'), '4,647 points');
});
