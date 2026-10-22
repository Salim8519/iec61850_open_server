import assert from 'node:assert/strict';
import test from 'node:test';
import { createHistory, recordDiagram, redoDiagram, undoDiagram, type DiagramHistory } from '../../frontend/src/state/diagramHistory.ts';
import { edgeStep, hasRemoval, nodeStep } from '../../frontend/src/features/diagram/canvasChanges.ts';
import { createRemovalGesture } from '../../frontend/src/features/diagram/removalGesture.ts';
import type { Diagram, EquipmentNode } from '../../frontend/src/api/types.ts';

const item = (id: string, x: number): EquipmentNode => ({ id, type: 'equipment', position: { x, y: 0 },
  data: { kind: 'CB', label: id, rotation: 0 } });
const wire = (id: string, source: string, target: string) => ({ id, source, target });
// cb2 is wired to cb1 and cb3; w3 joins cb1 and cb3 and does not touch cb2.
const drawing: Diagram = { nodes: [item('cb1', 0), item('cb2', 100), item('cb3', 200)],
  edges: [wire('w1', 'cb1', 'cb2'), wire('w2', 'cb2', 'cb3'), wire('w3', 'cb1', 'cb3')] };
const remove = (id: string) => ({ type: 'remove' as const, id });

/** A tick the test ends by hand (the page ends it with a microtask). */
function ticks() {
  const waiting: (() => void)[] = [];
  return { schedule: (end: () => void) => { waiting.push(end); }, end: () => waiting.splice(0).forEach(end => end()) };
}

/** What Canvas does when Delete removes items: React Flow reports the wires, then the items, in one
 * tick, both from the diagram the page had rendered. */
function deleteIn(gesture: ReturnType<typeof createRemovalGesture>, past: DiagramHistory, shown: Diagram,
  gone: { wires?: string[]; items?: string[] }): DiagramHistory {
  let next = past;
  if (gone.wires?.length) {
    const removal = gesture.apply(shown, edgeStep(gone.wires.map(remove)));
    next = recordDiagram(next, removal.diagram, removal.group);
  }
  if (gone.items?.length) {
    const removal = gesture.apply(shown, nodeStep(gone.items.map(remove), null));
    next = recordDiagram(next, removal.diagram, removal.group);
  }
  return next;
}

test('deleting a wired item is one undo step that brings back the item and its wires', () => {
  const clock = ticks();
  const gesture = createRemovalGesture(clock.schedule);
  const after = deleteIn(gesture, createHistory(drawing), drawing, { wires: ['w1', 'w2'], items: ['cb2'] });
  assert.deepEqual(after.present.nodes.map(node => node.id), ['cb1', 'cb3']);
  assert.deepEqual(after.present.edges.map(edge => edge.id), ['w3']);
  assert.equal(after.past.length, 1);
  const undone = undoDiagram(after);
  assert.deepEqual(undone.present.nodes.map(node => node.id), ['cb1', 'cb2', 'cb3']);
  assert.deepEqual(undone.present.edges.map(edge => edge.id), ['w1', 'w2', 'w3']);
  assert.equal(undone.past.length, 0, 'one Undo is all it takes');
  assert.equal(redoDiagram(undone).present.nodes.length, 2);
});

test('the same deletion without the gesture was two steps (the reported bug)', () => {
  let history = createHistory(drawing);
  history = recordDiagram(history, edgeStep([remove('w1'), remove('w2')])(drawing));
  history = recordDiagram(history, nodeStep([remove('cb2')], null)(drawing));
  assert.equal(history.past.length, 2);
  const undone = undoDiagram(history);
  assert.equal(undone.present.nodes.length, 3);
  assert.equal(undone.present.edges.length, 1, 'the item was back but not its wires');
});

test('a wire deleted together with the item stays deleted', () => {
  const gesture = createRemovalGesture(ticks().schedule);
  const after = deleteIn(gesture, createHistory(drawing), drawing, { wires: ['w1', 'w2', 'w3'], items: ['cb2'] });
  assert.deepEqual(after.present.nodes.map(node => node.id), ['cb1', 'cb3']);
  assert.deepEqual(after.present.edges, [], 'w3 was selected too and must not come back');
  assert.equal(undoDiagram(after).present.edges.length, 3);
});

test('two deletions in different ticks stay two undo steps', () => {
  const clock = ticks();
  const gesture = createRemovalGesture(clock.schedule);
  let history = deleteIn(gesture, createHistory(drawing), drawing, { wires: ['w1', 'w2'], items: ['cb2'] });
  clock.end();
  history = deleteIn(gesture, history, history.present, { wires: ['w3'], items: ['cb3'] });
  assert.equal(history.past.length, 2);
  assert.deepEqual(history.present.nodes.map(node => node.id), ['cb1']);
  const first = undoDiagram(history);
  assert.deepEqual(first.present.nodes.map(node => node.id), ['cb1', 'cb3']);
  assert.deepEqual(first.present.edges.map(edge => edge.id), ['w3']);
  assert.equal(undoDiagram(first).present.edges.length, 3);
});

test('a deletion never merges into the edit before it', () => {
  const clock = ticks();
  const gesture = createRemovalGesture(clock.schedule);
  let history = recordDiagram(createHistory(drawing), { ...drawing, nodes: [item('cb1', 40), ...drawing.nodes.slice(1)] });
  clock.end();
  history = deleteIn(gesture, history, history.present, { wires: ['w1', 'w2'], items: ['cb2'] });
  assert.equal(history.past.length, 2);
  assert.equal(undoDiagram(history).present.nodes[0].position.x, 40, 'the move is still done after one Undo');
});

test('an item or a wire deleted alone is one step', () => {
  const lone = createRemovalGesture(ticks().schedule);
  assert.equal(deleteIn(lone, createHistory(drawing), drawing, { wires: ['w3'] }).past.length, 1);
  const bare: Diagram = { nodes: [item('a', 0), item('b', 100)], edges: [] };
  const after = deleteIn(createRemovalGesture(ticks().schedule), createHistory(bare), bare, { items: ['b'] });
  assert.equal(after.past.length, 1);
  assert.equal(undoDiagram(after).present.nodes.length, 2);
});

test('group names never repeat, so unrelated gestures cannot merge', () => {
  const clock = ticks();
  const first = createRemovalGesture(clock.schedule);
  const second = createRemovalGesture(clock.schedule); // another canvas, for example after a page change
  const names = [first.apply(drawing, edgeStep([remove('w1')])).group, second.apply(drawing, edgeStep([remove('w1')])).group];
  clock.end();
  names.push(first.apply(drawing, edgeStep([remove('w1')])).group, second.apply(drawing, edgeStep([remove('w1')])).group);
  assert.equal(new Set(names).size, 4);
});

test('only lists that delete something start a gesture', () => {
  assert.equal(hasRemoval([{ type: 'select' }, { type: 'dimensions' }]), false);
  assert.equal(hasRemoval([{ type: 'select' }, { type: 'remove' }]), true);
});
