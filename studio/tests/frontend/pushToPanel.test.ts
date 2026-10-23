// Change 0180: push buttons into point panels, and far-away symbols left out of Fit view.
import assert from 'node:assert/strict';
import test from 'node:test';
import type { Diagram, EquipmentNode } from '../../frontend/src/api/types.ts';
import { convertiblePushButtons, convertPushButtons, PANEL_GROUPS } from '../../frontend/src/equipment/pushToPanel.ts';
import { besideDrawing, fitOptionsFor, strayIds } from '../../frontend/src/equipment/strayNodes.ts';

const at = (id: string, x: number, y: number, data: Partial<EquipmentNode['data']> = {}): EquipmentNode =>
  ({ id, type: 'equipment', position: { x, y }, data: { label: id, kind: 'CB', rotation: 0, ...data } } as EquipmentNode);
const status = (id: string, ied: string, x: number, y: number) => at(id, x, y, { kind: 'PUSH_BUTTON',
  push_mode: 'latching_status', state_mapping: { mode: 'direct', source: { ied_id: ied, ref: `${ied}/GGIO1.${id}.stVal`, report_ref: null } } });
const command = (id: string, ied: string, x: number, y: number) => at(id, x, y, { kind: 'PUSH_BUTTON',
  push_mode: 'spc_momentary', control_binding: { ied_id: ied, ref: `${ied}/GGIO2.${id}` } });

test('push buttons become one point panel per IED, statuses first, at the buttons place', () => {
  const diagram: Diagram = { nodes: [at('CB1', 0, 0), command('C1', 'a', 300, 900), status('S1', 'a', 100, 800),
    status('S2', 'b', 2000, 800), at('X', 0, 0, { kind: 'PUSH_BUTTON', push_mode: 'self_resetting' }),
    status('W', 'a', 50, 50)], edges: [{ id: 'w', source: 'CB1', target: 'W' }] };
  assert.equal(convertiblePushButtons(diagram), 3);
  const result = convertPushButtons(diagram, { a: 'BAY A', b: 'BAY B' });
  assert.equal(result.converted, 3);
  assert.equal(result.kept, 2, 'wired and self-resetting buttons stay buttons');
  const panels = result.diagram.nodes.filter(node => node.data.kind === 'POINT_PANEL');
  assert.equal(panels.length, 2);
  const panelA = panels.find(node => node.data.ied_id === 'a')!;
  assert.equal(panelA.data.label, 'BAY A points');
  assert.deepEqual(panelA.position, { x: 100, y: 800 });
  assert.deepEqual(panelA.data.points!.map(point => [point.id, point.type, point.group, point.label]),
    [['p1', 'status', PANEL_GROUPS.status, 'S1'], ['p2', 'command', PANEL_GROUPS.command, 'C1']]);
  assert.equal(panelA.data.points![0].source!.ref, 'a/GGIO1.S1.stVal');
  assert.equal(panelA.data.points![1].control!.ref, 'a/GGIO2.C1');
  assert.deepEqual(result.diagram.edges, diagram.edges);
  assert.ok(result.diagram.nodes.some(node => node.id === 'W') && result.diagram.nodes.some(node => node.id === 'X'));
});

test('more than 300 rows of one IED split into a second panel', () => {
  const nodes = Array.from({ length: 305 }, (_, index) => status(`S${index}`, 'a', index, 0));
  const panels = convertPushButtons({ nodes, edges: [] }, { a: 'A' }).diagram.nodes;
  assert.deepEqual(panels.map(node => [node.data.label, node.data.points!.length]), [['A points', 300], ['A points 2', 5]]);
});

test('a symbol far below the drawing is a stray and Fit view leaves it out', () => {
  const nodes = [...Array.from({ length: 20 }, (_, index) => at(`n${index}`, index * 150, (index % 4) * 200)),
    at('INFEED-5', 200, 23460)];
  assert.deepEqual(strayIds(nodes), ['INFEED-5']);
  const options = fitOptionsFor(nodes, { maxZoom: 1, padding: 0.3 }) as { nodes?: { id: string }[] };
  assert.equal(options.nodes?.length, 20);
  assert.ok(!options.nodes?.some(node => node.id === 'INFEED-5'));
  assert.deepEqual(besideDrawing(nodes, ['INFEED-5']), { x: 0, y: 800 });
  assert.deepEqual(strayIds(nodes.slice(0, 20)), [], 'a normal drawing has no strays');
  assert.deepEqual(strayIds([at('a', 0, 0), at('b', 9000, 0), at('c', 0, 9000)]), [], 'scattered items are not strays');
});
