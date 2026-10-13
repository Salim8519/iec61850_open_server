import assert from 'node:assert/strict';
import test from 'node:test';
// React Flow's own node adoption, used to prove projected nodes keep terminal bounds.
import { adoptUserNodes } from '../../frontend/node_modules/@xyflow/system/dist/esm/index.mjs';
import { stableFlowNodes } from '../../frontend/src/equipment/flowNodes.ts';
import { rememberMeasured, type MeasuredStore } from '../../frontend/src/equipment/measured.ts';
import type { EquipmentNode } from '../../frontend/src/api/types.ts';

const source: EquipmentNode[] = [{ id: 'cb', type: 'equipment', position: { x: 0, y: 0 },
  data: { kind: 'CB', label: 'CB-1', rotation: 0 } }];
const bounds = { source: [{ id: 'east', x: 100, y: 50, width: 9, height: 9, position: 'right' }], target: [] };

function adopted(nodes: EquipmentNode[], lookup: Map<string, any>) {
  adoptUserNodes(nodes, lookup, new Map(), { checkEquality: true });
  return lookup.get('cb');
}

function measuredLookup(nodes: EquipmentNode[]) {
  const lookup = new Map<string, any>();
  const internal = adopted(nodes, lookup);
  internal.measured = { width: 116, height: 112 };
  internal.internals.handleBounds = bounds;
  return lookup;
}

test('selecting a measured node keeps its terminal bounds so wires stay drawn', () => {
  const measured: MeasuredStore = new Map();
  rememberMeasured(measured, [{ type: 'dimensions', id: 'cb', dimensions: { width: 116, height: 112 } }]);
  let result = stableFlowNodes(source, new Set(), new Map(), measured);
  const lookup = measuredLookup(result.nodes);
  result = stableFlowNodes(source, new Set(['cb']), result.cache, measured);
  assert.equal(adopted(result.nodes, lookup).internals.handleBounds, bounds);
  const moved = [{ ...source[0], position: { x: 40, y: 0 } }];
  result = stableFlowNodes(moved, new Set(['cb']), result.cache, measured);
  assert.equal(adopted(result.nodes, lookup).internals.handleBounds, bounds);
});

test('without a remembered size React Flow drops the bounds (the failure being guarded)', () => {
  let result = stableFlowNodes(source, new Set(), new Map());
  const lookup = measuredLookup(result.nodes);
  result = stableFlowNodes(source, new Set(['cb']), result.cache);
  assert.equal(adopted(result.nodes, lookup).internals.handleBounds, undefined);
});

test('removed nodes are forgotten', () => {
  const measured: MeasuredStore = new Map([['cb', { width: 1, height: 1 }]]);
  rememberMeasured(measured, [{ type: 'remove', id: 'cb' }]);
  assert.equal(measured.size, 0);
});
