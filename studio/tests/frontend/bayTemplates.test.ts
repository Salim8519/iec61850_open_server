import assert from 'node:assert/strict';
import test from 'node:test';
import type { Diagram } from '../../frontend/src/api/types.ts';
import { addBay, BAY_ORDER, BAYS, nextBayNumber } from '../../frontend/src/equipment/bayTemplates.ts';
import { createHistory, recordDiagram, undoDiagram } from '../../frontend/src/state/diagramHistory.ts';
import { defaultSize, tidyDiagram } from '../../frontend/src/equipment/tidy/tidy.ts';

const empty: Diagram = { nodes: [], edges: [] };
const counter = () => { let n = 0; return () => `id${++n}`; };
const bus = (id: string, x: number): Diagram['nodes'][number] =>
  ({ id, type: 'equipment', position: { x, y: 0 }, data: { kind: 'JUNCTION', label: id, rotation: 0 } });
const kinds = (diagram: Diagram) => diagram.nodes.map(node => node.data.kind).sort().join(',');

function overlaps(diagram: Diagram) {
  return diagram.nodes.some((a, i) => diagram.nodes.slice(i + 1).some(b => {
    const sa = defaultSize(a), sb = defaultSize(b);
    return a.position.x < b.position.x + sb.width && b.position.x < a.position.x + sa.width &&
      a.position.y < b.position.y + sb.height && b.position.y < a.position.y + sa.height;
  }));
}

test('each bay holds the agreed equipment and no links', () => {
  assert.equal(kinds(addBay(empty, 'line_feeder').diagram), 'CB,CT,DS,DS,ES,INFEED,JUNCTION,VT');
  assert.equal(kinds(addBay(empty, 'transformer_bay').diagram), 'CB,CT,DS,ES,INFEED,JUNCTION');
  assert.equal(kinds(addBay(empty, 'bus_coupler').diagram), 'CB,DS,DS,JUNCTION,JUNCTION');
  for (const kind of BAY_ORDER) {
    const { diagram } = addBay(empty, kind);
    assert.ok(diagram.nodes.every(node => !node.data.state_mapping && !node.data.control_binding), `${kind} has no links`);
    assert.ok(!overlaps(diagram), `${kind} symbols do not overlap`);
    const feed = diagram.nodes.find(node => node.data.kind === 'INFEED');
    if (feed) assert.ok(feed.data.infeed_type && feed.data.infeed_state, 'infeed is fully described');
    assert.ok(diagram.nodes.filter(node => node.data.kind !== 'INFEED').every(node => node.data.infeed_type === undefined));
  }
  assert.equal(addBay(empty, 'line_feeder').diagram.nodes.find(n => n.data.kind === 'INFEED')!.data.infeed_state, 'dead',
    'an outgoing line feeds nothing until the user says so');
});

test('bays get fresh ids and the next bay number', () => {
  const first = addBay(empty, 'line_feeder', {}, counter());
  const second = addBay(first.diagram, 'line_feeder', { at: { x: 600, y: 0 } }, () => crypto.randomUUID());
  const labels = second.diagram.nodes.map(node => node.data.label);
  assert.equal(new Set(labels).size, labels.length, 'labels are unique');
  assert.ok(labels.includes('F1-Q0') && labels.includes('F2-Q0'));
  assert.equal(new Set(second.diagram.nodes.map(node => node.id)).size, second.diagram.nodes.length);
  assert.equal(nextBayNumber(second.diagram, 'F'), 3);
  assert.equal(nextBayNumber(second.diagram, 'T'), 1);
});

test('a bay connects to the selected busbar points', () => {
  const bars: Diagram = { nodes: [bus('a', 0), bus('b', 400)], edges: [] };
  const feeder = addBay(bars, 'line_feeder', { attach: ['a'] });
  const link = feeder.diagram.edges.find(edge => edge.source === 'a')!;
  const top = feeder.diagram.nodes.find(node => node.id === link.target)!;
  assert.deepEqual([link.sourceHandle, link.targetHandle, top.data.label], ['south', 'north', 'F1-Q1']);
  assert.equal(top.position.x + 54, 20, 'hangs straight under the busbar point');
  const coupler = addBay(bars, 'bus_coupler', { attach: ['a', 'b'] });
  const targets = coupler.diagram.edges.filter(edge => ['a', 'b'].includes(edge.source))
    .map(edge => coupler.diagram.nodes.find(node => node.id === edge.target)!.data.label);
  assert.deepEqual(targets, ['C1-Q1', 'C1-Q2']);
  assert.equal(addBay(bars, 'line_feeder', { attach: ['missing'] }).diagram.edges.length, BAYS.line_feeder.wires.length);
});

test('adding a bay is one undo step and tidies cleanly', () => {
  const bars: Diagram = { nodes: [bus('a', 0)], edges: [] };
  const after = addBay(bars, 'transformer_bay', { attach: ['a'] }).diagram;
  const history = recordDiagram(createHistory(bars), after);
  assert.equal(history.past.length, 1);
  assert.deepEqual(undoDiagram(history).present, bars);
  const tidy = tidyDiagram(after).diagram;
  assert.ok(!overlaps(tidy));
  assert.equal(tidyDiagram(tidy).moved, 0);
});
