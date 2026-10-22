// ADR-0105: the gateway's "Add points" dialog and the smart Propose preview. The engine plans and checks;
// these tests pin the display helpers and the page contract (what the dialog offers and when Add is allowed).
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { Candidate, Plan, TypeEntry } from '../../frontend/src/api/gatewayAdd.ts';
import type { Ied } from '../../frontend/src/api/types.ts';
import { ADD_COPY, iedBadge, SMART_COPY } from '../../frontend/src/copy/gatewayAdd.ts';
import { canAdd, filterIeds, hasRealDevice, picksOf, problemLines, splitTypes, tickShown, toggle, typedAddress }
  from '../../frontend/src/features/tools/gateway/add/addPlan.ts';

const SRC = new URL('../../frontend/src/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, SRC), 'utf8');

const types: TypeEntry[] = [
  { kind: 'single', name: 'Binary input', type_id: 'g1v2 / g2v2', most_used: true },
  { kind: 'command_single', name: 'Binary output command (CROB)', type_id: 'g12v1', most_used: true },
  { kind: 'integer', name: 'Analog input, 32-bit', type_id: 'g30v1 / g32v3', most_used: false },
];
const candidate = (ref: string, taken = false): Candidate =>
  ({ ref, name: ref, description: '', value_kind: 'BOOLEAN', fc: 'ST', taken, suggested: false });
const plan = (ok: boolean, role = 'simulated'): Plan => ({ ok, rows: [{ number: 1, problem: ok ? '' : 'Row 1: choose a type',
  point: { kind: 'single', address: 1, ied: 'f1', ref: 'X', scale: 1, description: '', ied_role: role } }],
  points: ok ? [{ kind: 'single', address: 1, ied: 'f1', ref: 'X', scale: 1, description: '', ied_role: role }] : [] });

test('the type list is the engine catalog: most used first, the rest under More types', () => {
  const { most, more } = splitTypes(types);
  assert.deepEqual(most.map(entry => entry.kind), ['single', 'command_single']);
  assert.deepEqual(more.map(entry => entry.kind), ['integer']);
  const list = read('features/tools/gateway/add/TypeList.tsx');
  assert.match(list, /ADD_COPY\.mostUsed[\s\S]*ADD_COPY\.moreTypes/);
  assert.match(list, /<small>\{entry\.type_id\}<\/small>/, 'the protocol type ids show as small secondary text');
  assert.match(list, /badge = kindOfGatewayPoint/, 'the gateway kinds are the default badge (the master passes its own, ADR-0110)');
  assert.match(list, /<KindBadge kind=\{badge\(entry\.kind\)\}/);
  assert.doesNotMatch(list, /M_SP|g12v1|FC5/, 'no type is written in the frontend: the catalog is one Go file');
});

test('IED search, ticks and picks keep the order and the typed names and addresses', () => {
  const ieds = [{ id: 'a', name: 'FEED1' }, { id: 'b', name: 'BUS1' }, { id: 'c', name: 'FEED2' }] as Ied[];
  assert.deepEqual(filterIeds(ieds, 'feed').map(ied => ied.id), ['a', 'c']);
  assert.deepEqual(toggle(toggle(['x'], 'y'), 'x'), ['y']);
  assert.deepEqual(tickShown(['b'], [candidate('a'), candidate('b'), candidate('c', true)]), ['b', 'a'], 'listed points stay unticked');
  assert.deepEqual(picksOf(['a', 'b'], { b: { name: 'CB', address: 50003 } }), [{ ref: 'a' }, { ref: 'b', name: 'CB', address: 50003 }]);
  assert.equal(typedAddress('1 042'), 1042);
  assert.equal(typedAddress('  '), undefined, 'empty means the next free address');
});

test('Add N points stays off until every row is right, and a real device needs I understand', () => {
  assert.equal(canAdd(plan(false), false, false), false);
  assert.deepEqual(problemLines(plan(false)), ['Row 1: choose a type']);
  assert.equal(canAdd(plan(true), false, false), true);
  assert.equal(canAdd(plan(true, 'real'), true, false), false);
  assert.equal(canAdd(plan(true, 'real'), true, true), true);
  assert.equal(hasRealDevice(plan(true, 'real').points), true);
  assert.equal(ADD_COPY.realLine, 'This is a real device. SCADA commands will operate it.');
  assert.equal(ADD_COPY.add(3), 'Add 3 points');
  assert.equal(iedBadge('real').label, 'Real device');
  assert.equal(iedBadge('simulated').label, 'Simulated');
  assert.notEqual(iedBadge('real').tone, iedBadge('simulated').tone);
});

test('the dialog offers From an IED, Simulated (made up) with a range, and Paste from Excel with the template', () => {
  const dialog = read('features/tools/gateway/add/AddPointsDialog.tsx');
  assert.deepEqual(Object.values(ADD_COPY.sources), ['From an IED', 'Simulated (made up)', 'Paste from Excel']);
  assert.match(dialog, /<OwnRange /);
  const from = read('features/tools/gateway/add/FromIed.tsx');
  assert.match(from, /useState\(true\)/, 'Also add its status point is ticked by default');
  assert.match(from, /command && withStatus/);
  assert.match(from, /<RealWarning /);
  const paste = read('features/tools/gateway/add/PasteRows.tsx');
  assert.match(paste, /addApi\.templateUrl\(protocol\)/);
  assert.match(paste, /addApi\.uploaded\(protocol/);
  assert.match(paste, /disabled=\{!canAdd\(plan, real, understood\)\}/);
  const rows = read('features/tools/gateway/add/PlannedRows.tsx');
  assert.match(rows, /role="alert">⚠ \{row\.problem\}/, 'a bad row says why in words, not colour alone');
});

test('Add points and Propose hand over the list before, so Undo puts it back', () => {
  const table = read('features/tools/gateway/PointTable.tsx');
  assert.match(table, /edit\(\[\.\.\.rows, \.\.\.points\], rows\)/);
  assert.match(table, /edit\(value\.points, rows\)/);
  assert.match(table, /<SmartPropose /);
  assert.deepEqual(Object.values(SMART_COPY.modes), ['Add only new points', 'Replace the IED points']);
  const smart = read('features/tools/gateway/add/SmartPropose.tsx');
  assert.match(smart, /useState<Mode>\(rows\.length \? 'add_new' : 'replace'\)/, 'a list with points keeps its edits by default');
});
