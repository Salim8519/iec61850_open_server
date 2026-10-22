// ADR-0110: the SCADA master's own point list. The engine compares, plans and checks; these tests pin the
// display helpers and the page contract (address-only, words with symbols, commands through the confirmation).
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { ComparedRow, Comparison, ListPlan, ListPoint } from '../../frontend/src/api/masterList.ts';
import { STATE_WORDS } from '../../frontend/src/copy/masterList.ts';
import { appendPoints, ALL, changePoint, checkText, draftOf, EXTRA, extraMatches, kindOfListType, listRowMatches, planProblems,
  planReady, removePoints, stateCounts, stateText, typedNumber } from '../../frontend/src/features/tools/gateway/master/list/listRows.ts';

const SRC = new URL('../../frontend/src/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, SRC), 'utf8');

const point = (id: string, type: string, address: number, name = ''): ListPoint =>
  ({ id, name, type, address, scale: 1, offset: 0, unit: '' });
const row = (state: ComparedRow['state'], extra: Partial<ComparedRow> = {}): ComparedRow =>
  ({ id: 'm1', point: point('m1', 'double', 20001, 'Feeder 1 breaker'), state, reason: '', command: null, ...extra });
const double = { kind: 'double', name: 'Double point', type_id: 'M_DP 3/31', most_used: true, group: 'double', command: '' };

test('each result is words with a symbol, never colour alone', () => {
  for (const words of Object.values(STATE_WORDS)) {
    assert.ok(words.symbol && words.label && words.hint);
  }
  assert.equal(stateText(row('missing')), '✗ Missing');
  assert.equal(stateText(row('wrong_type')), '⚠ Wrong type');
  assert.equal(checkText(row('received', { check: { ok: false, text: 'IED says Closed, master got Open: check the double point mapping.', source: '' } })),
    '✗ IED says Closed, master got Open: check the double point mapping.');
  assert.equal(checkText(row('received', { check: { ok: true, text: 'Both say Closed.', source: '' } })), '✓ Both say Closed.');
  assert.equal(checkText(row('received')), '', 'points not copied from the gateway have no IED check');
  const css = read('features/tools/gateway/master/list/master-list.css');
  assert.match(css, /words plus a\s+symbol/);
});

test('the filter chips count every result and the points not in the list', () => {
  const comparison = { rows: [], not_in_list: [{ key: 'float:1' }, { key: 'float:2' }],
    counts: { received: 3, missing: 1, wrong_type: 1, waiting: 0, command: 2, not_in_list: 2, total: 7 } } as unknown as Comparison;
  assert.deepEqual(stateCounts(comparison), { state: { all: 7, extra: 2, received: 3, missing: 1, wrong_type: 1, waiting: 0, command: 2 } });
  const shown = row('missing');
  assert.ok(listRowMatches(shown, { state: ALL, kind: 'all', text: 'feeder' }, double));
  assert.ok(!listRowMatches(shown, { state: 'received', kind: 'all', text: '' }, double));
  assert.ok(!listRowMatches(shown, { state: ALL, kind: 'SP', text: '' }, double));
  assert.ok(extraMatches({ key: 'float:30001', group: 'float', group_name: 'Measured (float)', address: 30001, type: 'M_ME_NC_1',
    label: '', value_text: '1.5', quality_text: 'good', good: true, list_type: 'float' }, { state: EXTRA, kind: 'MV', text: '30001' }));
  assert.equal(kindOfListType(double), 'DP');
  assert.equal(kindOfListType({ group: '', command: 'double' }), 'DC');
});

test('list edits keep the engine ids; numbers read as people type them', () => {
  const list = [point('m1', 'double', 20001), point('m2', 'single', 10001)];
  assert.deepEqual(appendPoints(list, [point('m3', 'float', 30001)]).map(item => item.id), ['m1', 'm2', 'm3']);
  assert.deepEqual(removePoints(list, new Set(['m1'])).map(item => item.id), ['m2']);
  assert.equal(changePoint(list, 'm2', { name: 'Alarm' })[1].name, 'Alarm');
  assert.equal(typedNumber('0,1'), 0.1);
  assert.equal(typedNumber(''), undefined);
  const bad = { rows: [{ number: 2, point: point('', 'x', 1), problem: 'Row 2: choose a type' }], points: [], ok: false } as ListPlan;
  assert.ok(!planReady(bad));
  assert.deepEqual(planProblems(bad), ['Row 2: choose a type']);
});

test('Send uses the saved command and goes through the same confirmation and REAL warning', () => {
  const command = { kind: 'double' as const, address: 50001, value: 2, select: true };
  assert.deepEqual(draftOf(row('command', { command })), command);
  assert.equal(draftOf(row('received')), null);
  const list = read('features/tools/gateway/master/list/MasterList.tsx');
  assert.match(list, /<MasterCommandForm[^>]*review/, 'Send opens the existing command form at its confirmation');
  const form = read('features/tools/gateway/master/MasterCommandForm.tsx');
  assert.match(form, /useState\(review\)/);
  assert.match(form, /MASTER_COPY\.realWarning/);
  assert.match(form, /<MasterCyclic/, 'repeat and address +1 stay available');
});

test('the master list is address-only: no IED anywhere in its dialog or table', () => {
  for (const file of ['ListAddDialog.tsx', 'ListOnePoint.tsx', 'ListPaste.tsx', 'ListTable.tsx', 'NotInList.tsx', 'ListPlanRows.tsx']) {
    const text = read(`features/tools/gateway/master/list/${file}`);
    assert.doesNotMatch(text, /ied_name|ieds:|IedChooser|FromIed|ied\??:/, `${file} must not link points to IEDs`);
  }
  const dialog = read('features/tools/gateway/master/list/ListAddDialog.tsx');
  assert.match(dialog, /add-points\.css/, 'the dialog reuses the gateway Add points pieces');
  assert.match(read('features/tools/gateway/master/list/ListOnePoint.tsx'), /<TypeList[\s\S]*badge=/);
  assert.doesNotMatch(read('features/tools/gateway/master/list/ListTable.tsx'), /M_DP|C_DC|g12v1|FC3/, 'type names come from the engine');
});
