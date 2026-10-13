import assert from 'node:assert/strict';
import test from 'node:test';
import { copySelection, nudgeNodes, pasteClip, removeNodes, uniqueLabel } from '../../frontend/src/equipment/clipboard.ts';
import type { Diagram } from '../../frontend/src/api/types.ts';

const diagram: Diagram = {
  nodes: ['a', 'b', 'c'].map((id, index) => ({ id, type: 'equipment', position: { x: index * 100, y: 0 },
    data: { kind: 'CB', label: `CB-${index + 1}`, rotation: 0,
      state_mapping: { ied_id: 'ied', reference: `LD/XCBR${index}.Pos.stVal` } } })) as Diagram['nodes'],
  edges: [
    { id: 'ab', source: 'a', target: 'b', sourceHandle: 'east', targetHandle: 'west' },
    { id: 'bc', source: 'b', target: 'c', sourceHandle: 'east', targetHandle: 'west' },
  ],
};

test('copy keeps only wires whose both ends are selected', () => {
  const clip = copySelection(diagram, new Set(['a', 'b']))!;
  assert.deepEqual(clip.nodes.map(node => node.id), ['a', 'b']);
  assert.deepEqual(clip.edges.map(edge => edge.id), ['ab']);
  assert.equal(copySelection(diagram, new Set()), null);
});

test('paste assigns fresh identifiers, remaps wires, offsets and keeps originals intact', () => {
  let counter = 0;
  const clip = copySelection(diagram, new Set(['a', 'b']))!;
  const { diagram: next, ids } = pasteClip(diagram, clip, 40, () => `n${++counter}`);
  assert.deepEqual(ids, ['n1', 'n2']);
  assert.equal(next.nodes.length, 5);
  assert.deepEqual(next.nodes.slice(0, 3), diagram.nodes, 'existing nodes unchanged');
  const pasted = next.nodes.slice(3);
  assert.deepEqual(pasted.map(node => node.position), [{ x: 40, y: 40 }, { x: 140, y: 40 }]);
  const wire = next.edges[2];
  assert.equal(wire.source, 'n1');
  assert.equal(wire.target, 'n2');
  assert.equal(wire.sourceHandle, 'east');
  assert.notEqual(wire.id, 'ab');
  pasted[0].data.label = 'changed';
  assert.equal(clip.nodes[0].data.label, 'CB-1', 'clip is not aliased by pasted nodes');
});

test('pasted labels are unique within the diagram', () => {
  const clip = copySelection(diagram, new Set(['a']))!;
  const once = pasteClip(diagram, clip, 40).diagram;
  const twice = pasteClip(once, clip, 80).diagram;
  assert.deepEqual(twice.nodes.slice(3).map(node => node.data.label), ['CB-4', 'CB-5']);
  assert.equal(uniqueLabel('Feeder', new Set(['Feeder', 'Feeder copy'])), 'Feeder copy 2');
});

test('remove drops attached wires and nudge moves only selected nodes', () => {
  const removed = removeNodes(diagram, new Set(['b']));
  assert.deepEqual(removed.nodes.map(node => node.id), ['a', 'c']);
  assert.deepEqual(removed.edges, []);
  const moved = nudgeNodes(diagram, new Set(['c']), 20, -20);
  assert.deepEqual(moved.nodes[2].position, { x: 220, y: -20 });
  assert.equal(moved.nodes[0], diagram.nodes[0]);
});
