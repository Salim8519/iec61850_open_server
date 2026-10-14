import assert from 'node:assert/strict';
import test from 'node:test';
import { ENERGY_LOOK, ENERGY_ORDER, EnergizationStore, energizedEdges, NO_ENERGY, sectionSentence,
  type EdgeCache } from '../../frontend/src/equipment/energization.ts';

const CANVAS = '#e9edf2';
const luminance = (hex: string) => {
  const [r, g, b] = [1, 3, 5].map(i => parseInt(hex.slice(i, i + 2), 16) / 255)
    .map(c => c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};
const contrast = (a: string, b: string) => {
  const [x, y] = [luminance(a), luminance(b)].sort((m, n) => n - m);
  return (x + 0.05) / (y + 0.05);
};

test('every state has a style and a word', () => {
  assert.deepEqual([...ENERGY_ORDER].sort(), Object.keys(ENERGY_LOOK).sort());
  const patterns = new Set(ENERGY_ORDER.map(state => `${ENERGY_LOOK[state].width}/${ENERGY_LOOK[state].dash ?? 'solid'}`));
  assert.equal(patterns.size, ENERGY_ORDER.length, 'line pattern alone tells every state apart');
  for (const state of ENERGY_ORDER) {
    const look = ENERGY_LOOK[state];
    assert.ok(look.word && look.meaning, state);
    assert.ok(contrast(look.colour, CANVAS) >= 3, `${state} line is visible on the canvas (WCAG non-text 3:1)`);
  }
});

test('wires keep their object unless their state changes', () => {
  const edges = [{ id: 'a' }, { id: 'b' }];
  let cache: EdgeCache = new Map();
  const first = energizedEdges(edges, { a: 'live' }, cache);
  cache = first.cache;
  assert.equal(first.edges[1], edges[1], 'wire without a state is untouched');
  assert.equal(first.edges[0].className, 'wire-live');
  const again = energizedEdges(edges, { a: 'live', b: 'dead' }, cache);
  assert.equal(again.edges[0], first.edges[0], 'same state, same object');
  assert.equal(again.edges[1].className, 'wire-dead');
});

test('sheet sentence and store', () => {
  assert.equal(sectionSentence(['live']), 'This part is live.');
  assert.equal(sectionSentence(['live', 'dead']), 'One side is live, the other side is dead.');
  assert.equal(sectionSentence(undefined), '');
  const store = new EnergizationStore();
  let calls = 0;
  store.subscribe(() => { calls += 1; });
  store.set({ nodes: {}, edges: { a: 'live' } });
  assert.equal(store.get().edges.a, 'live');
  store.clear();
  assert.equal(store.get(), NO_ENERGY);
  assert.equal(calls, 2);
});
