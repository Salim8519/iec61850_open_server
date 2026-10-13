import assert from 'node:assert/strict';
import test from 'node:test';
import { applyNodeSizes, isLiveResize, isResizeEnd } from '../../frontend/src/equipment/resize.ts';
import type { EquipmentNode } from '../../frontend/src/api/types.ts';

const busbar: EquipmentNode = { id: 'bus', type: 'equipment', position: { x: 0, y: 0 },
  data: { kind: 'BUSBAR', label: 'B1', rotation: 0 } };
const breaker: EquipmentNode = { id: 'cb', type: 'equipment', position: { x: 0, y: 200 },
  data: { kind: 'CB', label: 'CB1', rotation: 0 } };
const size = (id: string, width: number | undefined, resizing?: boolean) =>
  ({ id, type: 'dimensions', resizing, dimensions: width === undefined ? undefined : { width, height: 112 } }) as const;

test('a resize event persists a bounded rounded width on the busbar only', () => {
  const sized = applyNodeSizes([busbar, breaker], [size('bus', 1234.6, true)]);
  assert.equal(sized[0].data.width, 1235);
  assert.equal(sized[1].data.width, undefined);
  assert.notEqual(sized[0], busbar, 'a resized node must be a new object');
});

test('resize widths are clamped to the persisted bounds', () => {
  const small = { ...busbar, id: 'bus-a' };
  const large = { ...busbar, id: 'bus-b' };
  const [first, second] = applyNodeSizes([small, large], [size('bus-a', 20, true), size('bus-b', 9999, true)]);
  assert.equal(first.data.width, 80);
  assert.equal(second.data.width, 2400);
});

test('measure events without a resizing flag never change node data', () => {
  const sized = applyNodeSizes([busbar], [size('bus', 500)]);
  assert.equal(sized[0], busbar, 'plain measurement must not clone equipment');
});

test('unchanged widths keep the same node object identity', () => {
  const started = applyNodeSizes([busbar], [size('bus', 300, true)]);
  const again = applyNodeSizes(started, [size('bus', 300, true)]);
  assert.equal(again[0], started[0], 'an unchanged width must not rerender the node');
});

test('live and final resize events are recognised for undo grouping', () => {
  assert.equal(isLiveResize(size('bus', 300, true)), true);
  assert.equal(isLiveResize(size('bus', 300, false)), false);
  assert.equal(isResizeEnd(size('bus', 300, false)), true);
  assert.equal(isResizeEnd(size('bus', 300, true)), false);
});
