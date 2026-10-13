import assert from 'node:assert/strict';
import test from 'node:test';
import { connectedSides } from '../../frontend/src/equipment/junctionSides.ts';

test('junction stubs follow the wired terminals of this node only', () => {
  const sides = connectedSides('j', [
    { source: 'j', target: 'cb', sourceHandle: 'east', targetHandle: 'west' },
    { source: 'ds', target: 'j', sourceHandle: 'south', targetHandle: 'north' },
    { source: 'j', target: 'j2', sourceHandle: 'east', targetHandle: 'south' },
  ]);
  assert.deepEqual([...sides].sort(), ['east', 'north']);
  assert.equal(connectedSides('j', []).size, 0);
  assert.equal(connectedSides('j', [{ source: 'j', target: 'x', sourceHandle: null }]).size, 0);
});
