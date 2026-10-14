import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { Diagram, EquipmentKind } from '../../frontend/src/api/types.ts';
import { createHistory, recordDiagram, undoDiagram } from '../../frontend/src/state/diagramHistory.ts';
import { defaultSize, tidyDiagram } from '../../frontend/src/equipment/tidy/tidy.ts';
import { TIDY_LOCKED, tidyMessage, tidyTitle } from '../../frontend/src/features/diagram/tidyCopy.ts';

type Node = Diagram['nodes'][number];
const read = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8');
const model: Diagram = JSON.parse(read('./fixtures/model_substation_diagram.json'));

function random(seed: number) {
  let state = seed;
  return () => (state = (state * 9301 + 49297) % 233280) / 233280;
}

/** Items at messy places, wired as listed ("a-b"). */
function drawing(items: Record<string, EquipmentKind>, wires: string[], seed = 3, rotation = 0): Diagram {
  const next = random(seed);
  return {
    nodes: Object.entries(items).map(([id, kind]) => ({ id, type: 'equipment', data: { kind, label: id, rotation },
      position: { x: Math.round(next() * 900), y: Math.round(next() * 700) } })),
    edges: wires.map(wire => {
      const [source, target] = wire.split('-');
      return { id: wire, source, target, sourceHandle: 'east', targetHandle: 'west' };
    }),
  };
}

/** A busbar of three junctions with two feeders; feeder 2 has a tee to an earth switch. */
const singleBus = () => drawing(
  { J1: 'JUNCTION', J2: 'JUNCTION', J3: 'JUNCTION', DS1: 'DS', CB1: 'CB', CT1: 'CT', DS2: 'DS', CB2: 'CB',
    T2: 'JUNCTION', ES2: 'ES', CT2: 'CT', VT: 'VT', REL: 'RELAY', PB: 'PUSH_BUTTON' },
  ['J1-J2', 'J2-J3', 'J1-DS1', 'DS1-CB1', 'CB1-CT1', 'J2-DS2', 'DS2-CB2', 'CB2-T2', 'T2-ES2', 'T2-CT2', 'J3-VT'], 3, 90);

/** Two busbars (A, B), two bays joining both through a tee, and a bus coupler. */
function doubleBus(): Diagram {
  const items: Record<string, EquipmentKind> = { A1: 'JUNCTION', A2: 'JUNCTION', A3: 'JUNCTION',
    B1: 'JUNCTION', B2: 'JUNCTION', B3: 'JUNCTION', CA: 'DS', CC: 'CB', CBB: 'DS' };
  const wires = ['A1-A2', 'A2-A3', 'B1-B2', 'B2-B3', 'A2-CA', 'CA-CC', 'CC-CBB', 'CBB-B2'];
  for (const [bay, bus] of [['1', '1'], ['2', '3']]) {
    Object.assign(items, { [`DA${bay}`]: 'DS', [`DB${bay}`]: 'DS', [`T${bay}`]: 'JUNCTION', [`CB${bay}`]: 'CB', [`CT${bay}`]: 'CT' });
    wires.push(`A${bus}-DA${bay}`, `B${bus}-DB${bay}`, `DA${bay}-T${bay}`, `DB${bay}-T${bay}`, `T${bay}-CB${bay}`, `CB${bay}-CT${bay}`);
  }
  return drawing(items, wires, 11);
}

const byId = (diagram: Diagram) => new Map(diagram.nodes.map(node => [node.id, node]));
const centre = (node: Node) => ({ x: node.position.x + defaultSize(node).width / 2, y: node.position.y + defaultSize(node).height / 2 });

function overlaps(diagram: Diagram): string[] {
  const found: string[] = [];
  diagram.nodes.forEach((a, index) => diagram.nodes.slice(index + 1).forEach(b => {
    const sa = defaultSize(a), sb = defaultSize(b);
    if (a.position.x < b.position.x + sb.width && b.position.x < a.position.x + sa.width &&
        a.position.y < b.position.y + sb.height && b.position.y < a.position.y + sa.height) found.push(`${a.id}/${b.id}`);
  }));
  return found;
}

function fuzz(seed: number): Diagram {
  const next = random(seed);
  const kinds: EquipmentKind[] = ['CB', 'DS', 'ES', 'CT', 'VT', 'SOURCE', 'RELAY', 'TRIP_RELAY', 'PUSH_BUTTON', 'JUNCTION', 'JUNCTION'];
  const nodes: Node[] = Array.from({ length: 2 + Math.floor(next() * 50) }, (_, index) => ({
    id: `n${index}`, type: 'equipment', position: { x: Math.round(next() * 3000), y: Math.round(next() * 2000) },
    data: { kind: kinds[Math.floor(next() * kinds.length)], label: `n${index}`, rotation: [0, 90, 180, 270][Math.floor(next() * 4)] } }));
  const wireable = nodes.filter(node => !['RELAY', 'PUSH_BUTTON'].includes(node.data.kind));
  const edges = wireable.slice(1).flatMap((node, index) => next() < 0.85 ?
    [{ id: `e${index}`, source: wireable[Math.floor(next() * (index + 1))].id, target: node.id }] : []);
  edges.push({ id: 'loose-end', source: nodes[0].id, target: 'not-on-diagram' });
  return { nodes, edges };
}

test('tidy never adds, removes or relinks anything', () => {
  for (const before of [singleBus(), doubleBus(), model, ...Array.from({ length: 40 }, (_, i) => fuzz(i + 1))]) {
    const after = tidyDiagram(before).diagram;
    assert.deepEqual(after.nodes.map(node => node.id), before.nodes.map(node => node.id));
    after.nodes.forEach((node, index) => {
      const { rotation: _turned, ...data } = node.data;
      const { rotation: _was, ...original } = before.nodes[index].data;
      assert.deepEqual(data, original);
    });
    assert.deepEqual(after.edges.map(edge => edge.id), before.edges.map(edge => edge.id));
    after.edges.forEach((edge, index) => assert.deepEqual([edge.source, edge.target].sort(),
      [before.edges[index].source, before.edges[index].target].sort()));
  }
});

test('no items overlap after tidy', () => {
  assert.deepEqual(overlaps(tidyDiagram(model).diagram), []);
  for (let seed = 1; seed <= 150; seed += 1) assert.deepEqual(overlaps(tidyDiagram(fuzz(seed)).diagram), [], `seed ${seed}`);
});

test('busbars are a straight top row and bays hang below', () => {
  const after = byId(tidyDiagram(singleBus()).diagram);
  const bar = ['J1', 'J2', 'J3'].map(id => centre(after.get(id)!));
  assert.ok(bar.every(point => point.y === bar[0].y), 'one busbar row');
  for (const id of ['DS1', 'CB1', 'CT1', 'DS2', 'CB2', 'VT']) assert.ok(centre(after.get(id)!).y > bar[0].y, `${id} below`);
  const { edges } = tidyDiagram(singleBus()).diagram;
  for (const edge of edges) {
    const a = centre(after.get(edge.source)!), b = centre(after.get(edge.target)!);
    if (a.y === b.y) assert.deepEqual([edge.sourceHandle, edge.targetHandle, a.x < b.x], ['east', 'west', true], edge.id);
    else assert.deepEqual([edge.sourceHandle, edge.targetHandle, a.y < b.y], ['south', 'north', true], edge.id);
  }
});

test('a bay is one straight column with its earth switch beside it', () => {
  const after = byId(tidyDiagram(singleBus()).diagram);
  const column = (ids: string[]) => new Set(ids.map(id => centre(after.get(id)!).x));
  assert.equal(column(['DS1', 'CB1', 'CT1']).size, 1);
  assert.equal(column(['J2', 'DS2', 'CB2', 'T2', 'CT2']).size, 1, 'run continues straight through the tee');
  assert.notEqual(centre(after.get('ES2')!).x, centre(after.get('CT2')!).x, 'earth switch steps aside');
});

test('double busbar stacks both bars and centres each bay tee', () => {
  const after = byId(tidyDiagram(doubleBus()).diagram);
  const y = (id: string) => centre(after.get(id)!).y;
  const x = (id: string) => centre(after.get(id)!).x;
  assert.ok(['A2', 'A3'].every(id => y(id) === y('A1')) && ['B2', 'B3'].every(id => y(id) === y('B1')));
  assert.ok(y('A1') !== y('B1') && Math.max(y('A1'), y('B1')) < y('DA1'), 'two bar rows on top');
  for (const bay of ['1', '2']) {
    assert.equal(x(`T${bay}`), (x(`DA${bay}`) + x(`DB${bay}`)) / 2, `tee ${bay} centred`);
    assert.equal(x(`CB${bay}`), x(`T${bay}`));
    assert.equal(y(`DA${bay}`), y(`DB${bay}`));
  }
  // In the SCD-built station, the two 50 kV bars are stacked; bay tees are not.
  const station = byId(tidyDiagram(model).diagram);
  const bars = model.nodes.filter(node => node.data.kind === 'JUNCTION' && /^BB[12]\/L$/.test(node.data.label));
  const rows = new Set(bars.map(node => centre(station.get(node.id)!).y));
  assert.equal(rows.size, 2);
  assert.ok(Math.max(...rows) - Math.min(...rows) <= 100, 'stacked directly under each other');
});

test('unwired items are parked to the right, panels first', () => {
  const after = tidyDiagram(singleBus()).diagram;
  const wired = after.nodes.filter(node => !['REL', 'PB'].includes(node.id));
  const right = Math.max(...wired.map(node => node.position.x + defaultSize(node).width));
  const rel = after.nodes.find(node => node.id === 'REL')!, pb = after.nodes.find(node => node.id === 'PB')!;
  assert.ok(rel.position.x > right && pb.position.x > right);
  assert.ok(rel.position.y < pb.position.y, 'relay panel before push button');
});

test('tidy selection moves only the selected items', () => {
  const before = singleBus();
  const scope = new Set(['DS1', 'CB1', 'CT1']);
  const after = tidyDiagram(before, { scope }).diagram;
  before.nodes.forEach((node, index) => {
    if (!scope.has(node.id)) assert.equal(after.nodes[index], node, `${node.id} untouched`);
  });
  const corner = (diagram: Diagram) => ({ x: Math.min(...diagram.nodes.filter(n => scope.has(n.id)).map(n => n.position.x)),
    y: Math.min(...diagram.nodes.filter(n => scope.has(n.id)).map(n => n.position.y)) });
  assert.ok(Math.abs(corner(after).x - corner(before).x) <= 10 && Math.abs(corner(after).y - corner(before).y) <= 10, 'stays where it was');
  assert.equal(after.edges.find(edge => edge.id === 'J1-DS1'), before.edges.find(edge => edge.id === 'J1-DS1'), 'outside wire unchanged');
  assert.equal(new Set(['DS1', 'CB1', 'CT1'].map(id => centre(byId(after).get(id)!).x)).size, 1);
});

test('tidy is deterministic and a second tidy changes nothing', () => {
  for (const make of [singleBus, doubleBus, () => model]) {
    const first = tidyDiagram(make());
    assert.deepEqual(tidyDiagram(make()).diagram, first.diagram);
    const again = tidyDiagram(first.diagram);
    assert.equal(again.moved, 0);
    assert.equal(again.diagram, first.diagram, 'same object: nothing to save');
  }
});

test('symbols turn to follow their wires', () => {
  const after = byId(tidyDiagram(singleBus()).diagram);
  for (const id of ['DS1', 'CB1', 'CT1', 'VT']) assert.equal(after.get(id)!.data.rotation, 0, `${id} upright on vertical wires`);
  for (const id of ['REL', 'PB']) assert.equal(after.get(id)!.data.rotation, 90, `${id} has no wires and keeps its turn`);
});

test('tidy is one undo step and locked while running', () => {
  const before = singleBus();
  const after = tidyDiagram(before).diagram;
  const history = recordDiagram(createHistory(before), after);
  assert.equal(history.past.length, 1);
  assert.deepEqual(undoDiagram(history).present, before);
  const page = read('../../frontend/src/features/diagram/DiagramPage.tsx');
  const button = read('../../frontend/src/features/diagram/TidyButton.tsx');
  assert.match(page, /<TidyButton [^>]*locked=\{locked\}[^>]*apply=\{\(next: Diagram\) => edit\(next\)\}/);
  assert.match(button, /disabled=\{locked \|\|/);
  assert.doesNotMatch(button, /api\(|'PUT'/);
  assert.equal(tidyTitle(true, false), TIDY_LOCKED);
  assert.match(tidyTitle(false, false), /Undo puts it back/);
  assert.match(tidyMessage(true, 3, false), /^Tidied 3 items\. Press Undo/);
  assert.match(tidyMessage(false, 0, true), /already tidy/);
});
