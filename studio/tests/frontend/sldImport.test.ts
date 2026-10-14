import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { createHistory, recordDiagram, undoDiagram } from '../../frontend/src/state/diagramHistory.ts';
import { NO_LAYOUT_HINT, previewIsCurrent, sldAvailability, sldSummary } from '../../frontend/src/features/diagram/sldPreview.ts';

const card = (id: string, x = 0) => ({ id, type: 'equipment' as const, position: { x, y: 0 },
  data: { kind: 'CB' as const, label: id, rotation: 0 } });
const component = readFileSync(new URL('../../frontend/src/features/diagram/SldImport.tsx', import.meta.url), 'utf8');
const page = readFileSync(new URL('../../frontend/src/features/diagram/DiagramPage.tsx', import.meta.url), 'utf8');

test('apply is a single undo step', () => {
  const mine = { nodes: [card('a'), card('b', 140)], edges: [] };
  let history = createHistory(mine);
  const imported = { nodes: [...mine.nodes, ...Array.from({ length: 30 }, (_, i) => card(`scl-${i}`, 400 + i * 140))], edges: [] };
  history = recordDiagram(history, imported);
  assert.equal(history.past.length, 1);
  assert.deepEqual(undoDiagram(history).present.nodes.map(node => node.id), ['a', 'b']);
  // The dialog applies through the normal diagram edit (no drag group), never a direct save.
  assert.match(page, /apply=\{\(next: Diagram\) => edit\(next\)\}/);
  assert.doesNotMatch(component, /'\/project\/diagram', 'PUT'/);
  const preview = { revision: 4 } as Parameters<typeof previewIsCurrent>[0];
  assert.ok(previewIsCurrent(preview, 4) && !previewIsCurrent(preview, 5));
});

test('disabled without substation section', () => {
  const plain = { id: 'x', filename: 'relay.cid', ieds: ['P1'], substations: [], equipment: 0 };
  const station = { ...plain, id: 'y', filename: 'station.scd', substations: ['S1'], equipment: 12 };
  assert.deepEqual(sldAvailability([plain]), { enabled: false, hint: NO_LAYOUT_HINT, usable: [] });
  assert.equal(sldAvailability([]).enabled, false);
  assert.ok(sldAvailability([]).hint.length > 20);
  assert.deepEqual(sldAvailability([plain, station]).usable, [station]);
  const summary = sldSummary({ diagram: { nodes: [], edges: [] }, revision: 1, layout: 'automatic', removed: 3,
    drawn: [{ path: 'a', kind: 'CB', label: 'a', bound: true }], skipped: [{ path: 's', type: 'IFL', reason: 'r' }],
    unbound: [], diagnostics: [], already_present: [] });
  assert.ok(summary[0].startsWith('1 items will be drawn, 1 of them'));
  assert.ok(summary.some(line => line.includes('removes the 3 items')));
});
