import assert from 'node:assert/strict';
import test from 'node:test';
import { createHistory, recordDiagram, sameEngineering, undoDiagram, redoDiagram } from '../../frontend/src/state/diagramHistory.ts';

const empty = { nodes: [], edges: [] };
const node = { id: 'cb', type: 'equipment' as const, position: { x: 20, y: 20 },
  data: { kind: 'CB' as const, label: 'CB1', rotation: 0 } };

test('undo and redo preserve equipment identifiers and discard stale redo on edit', () => {
  let state = createHistory(empty);
  state = recordDiagram(state, { nodes: [node], edges: [] });
  state = undoDiagram(state);
  assert.equal(state.present.nodes.length, 0);
  state = redoDiagram(state);
  assert.equal(state.present.nodes[0].id, 'cb');
  state = undoDiagram(state);
  state = recordDiagram(state, { nodes: [{ ...node, id: 'other' }], edges: [] });
  assert.equal(state.future.length, 0);
});

test('drag transactions collapse and selection-only changes do not create history', () => {
  let state = createHistory({ nodes: [node], edges: [] });
  state = recordDiagram(state, { nodes: [{ ...node, selected: true }], edges: [] });
  assert.equal(state.past.length, 0);
  state = recordDiagram(state, { nodes: [{ ...node, position: { x: 40, y: 20 } }], edges: [] }, 'drag');
  state = recordDiagram(state, { nodes: [{ ...node, position: { x: 60, y: 20 } }], edges: [] }, 'drag');
  assert.equal(state.past.length, 1);
  assert.equal(undoDiagram(state).present.nodes[0].position.x, 20);
});

test('history is bounded and stored snapshots do not share mutable input data', () => {
  let state = createHistory(empty);
  const input = structuredClone(node);
  state = recordDiagram(state, { nodes: [input], edges: [] });
  input.data.label = 'Changed outside history';
  assert.equal(state.present.nodes[0].data.label, 'CB1');
  for (let index = 0; index < 70; index++) {
    state = recordDiagram(state, { nodes: [{ ...node, position: { x: index, y: 0 } }], edges: [] });
  }
  assert.equal(state.past.length, 50);
});

test('measured sizes alone do not count as an engineering change', () => {
  const measured = { ...node, measured: { width: 104, height: 90 }, selected: true };
  assert.equal(sameEngineering({ nodes: [node], edges: [] }, { nodes: [measured], edges: [] }), true);
  const moved = { ...node, position: { x: 40, y: 20 } };
  assert.equal(sameEngineering({ nodes: [node], edges: [] }, { nodes: [moved], edges: [] }), false);
  const renamed = { ...node, data: { ...node.data, label: 'CB2' } };
  assert.equal(sameEngineering({ nodes: [node], edges: [] }, { nodes: [renamed], edges: [] }), false);
});
