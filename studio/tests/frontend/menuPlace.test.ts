// Menus open next to their button and always stay inside the window (no cut-off items).
import assert from 'node:assert/strict';
import test from 'node:test';
import { menuPlace } from '../../frontend/src/ui/menuPlace.ts';

const view = { width: 1366, height: 768 };
const button = (left: number, top: number) => ({ left, top, right: left + 200, bottom: top + 40 });

test('a left-aligned menu opens under its button', () => {
  const place = menuPlace(button(270, 330), { width: 320, height: 220 }, view, 'left', false);
  assert.deepEqual([place.left, place.top], [270, 376]);
});

test('a menu wider than the space to the right moves left to stay in the window', () => {
  const place = menuPlace(button(1200, 330), { width: 320, height: 220 }, view, 'left', false);
  assert.equal(place.left, 1366 - 320 - 8);
});

test('a right-aligned menu ends at its button and never past the left edge', () => {
  assert.equal(menuPlace(button(900, 20), { width: 240, height: 120 }, view, 'right', false).left, 1100 - 240);
  assert.equal(menuPlace(button(0, 20), { width: 240, height: 120 }, view, 'right', false).left, 8);
});

test('a menu flips up when there is no room below, and down when there is no room above', () => {
  assert.equal(menuPlace(button(270, 700), { width: 320, height: 220 }, view, 'left', false).top, 700 - 6 - 220);
  assert.equal(menuPlace(button(270, 20), { width: 320, height: 220 }, view, 'left', true).top, 66);
});

test('a menu taller than the window is limited to the window and scrolls', () => {
  const place = menuPlace(button(270, 300), { width: 320, height: 2000 }, view, 'left', false);
  assert.equal(place.top, 8);
  assert.equal(place.maxHeight, 768 - 16);
});
