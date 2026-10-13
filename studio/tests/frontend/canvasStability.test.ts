import assert from 'node:assert/strict';
import test from 'node:test';
import { stableFlowNodes } from '../../frontend/src/equipment/flowNodes.ts';
import { EquipmentValueStore } from '../../frontend/src/equipment/liveValues.ts';
import type { EquipmentNode } from '../../frontend/src/api/types.ts';

const sourceNodes: EquipmentNode[] = ['trip-a', 'trip-b', 'trip-c'].map((id, index) => ({
  id, type: 'equipment', position: { x: index * 120, y: 0 },
  data: { kind: 'TRIP_RELAY', label: id, rotation: 0,
    trip_command_polarity: 'operate_false', trip_state_polarity: 'operated_true' },
}));

test('unrelated live values notify only the equipment owning that key', () => {
  const values = new EquipmentValueStore();
  let firstRenders = 0;
  let secondRenders = 0;
  values.subscribe('trip-a', () => { firstRenders += 1; });
  values.subscribe('trip-b', () => { secondRenders += 1; });
  for (let index = 0; index < 1000; index += 1) values.set('trip-b', index % 2 === 0);
  assert.equal(firstRenders, 0);
  assert.equal(secondRenders, 1000);
  values.set('trip-b', false);
  assert.equal(secondRenders, 1000, 'an unchanged value must not rerender its node');
});

test('equipment snapshot replaces stale values and preserves authoritative no-op state', () => {
  const values = new EquipmentValueStore();
  let firstRenders = 0;
  values.subscribe('trip-a', () => { firstRenders += 1; });
  values.set('trip-a', false);
  values.set('stale', true);
  values.replace([{ node_id: 'trip-a', value: false }, { node_id: 'trip-b', value: true }]);
  assert.equal(firstRenders, 1, 'unchanged hydrated state must not notify again');
  assert.equal(values.get('stale'), undefined);
  assert.equal(values.get('trip-b'), true);
});

test('one thousand live changes and selection toggles retain visible stable nodes', () => {
  const values = new EquipmentValueStore();
  let cache = new Map();
  let result = stableFlowNodes(sourceNodes, new Set(['trip-a']), cache);
  cache = result.cache;
  const untouched = result.nodes[2];
  for (let index = 0; index < 1000; index += 1) {
    values.set(index % 2 ? 'trip-a' : 'trip-b', index % 4 < 2);
    const selected = index % 2 ? 'trip-a' : 'trip-b';
    result = stableFlowNodes(sourceNodes, new Set([selected]), cache);
    cache = result.cache;
    assert.deepEqual(result.nodes.filter(node => node.selected).map(node => node.id), [selected]);
    assert.ok(result.nodes.every(node =>
      (node.initialWidth || 0) > 0 && (node.initialHeight || 0) > 0));
    assert.equal(result.nodes[2], untouched, 'unaffected node objects remain stable');
  }
});
