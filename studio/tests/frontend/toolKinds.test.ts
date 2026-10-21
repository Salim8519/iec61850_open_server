// ADR-0095 in the tools: SNMP value lists, IED Explorer and the slim tool header use the shared point kinds.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { Attribute, WatchItem } from '../../frontend/src/api/tools.ts';
import type { ValueRow } from '../../frontend/src/api/snmp.ts';
import { attributeKind, basicTypeOf, objectKinds, watchKind } from '../../frontend/src/features/tools/explorer/attributeKind.ts';
import { attributeShown, NODE_FILTER, WATCH_FILTER, watchMatches } from '../../frontend/src/features/tools/explorer/explorerFilters.ts';
import { kindCounts, kindOfValue, readShown, READ_FILTER, simShown, SIM_FILTER } from '../../frontend/src/features/tools/snmp/valueFilter.ts';
import { initialValues } from '../../frontend/src/ui/filterState.ts';

const SRC = new URL('../../frontend/src/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, SRC), 'utf8');

const value = (type: string, extra: Partial<ValueRow> = {}): ValueRow => ({ oid: `1.3.6.1.4.1.9.${type.length}`, type: type as ValueRow['type'],
  value: '1', name: `X-MIB::${type}`, meaning: '', writable: false, ...extra });
const ROWS = [value('integer', { writable: true }), value('gauge32'), value('counter32', { key: true }), value('counter64'),
  value('octets', { polled: true }), value('timeticks'), value('null')];

test('an SNMP value gets its kind from its SMI type, and a value without a type stays Unknown', () => {
  assert.deepEqual(ROWS.map(kindOfValue), ['IS', 'MV', 'CT', 'CT', 'TX', 'CT', 'UN']);
  assert.equal(kindOfValue({ type: '' as ValueRow['type'] }), 'UN');
  assert.deepEqual(kindCounts(ROWS), { kind: { IS: 1, MV: 1, CT: 3, TX: 1, UN: 1 } });
});

test('the SNMP value lists filter by search, type chip and their own toggles', () => {
  for (const spec of [SIM_FILTER, READ_FILTER]) assert.equal(spec.groups?.[0].key, 'kind', 'both lists have the type chips');
  const all = initialValues(SIM_FILTER);
  assert.equal(ROWS.filter(row => simShown(row, all)).length, ROWS.length);
  assert.equal(ROWS.filter(row => simShown(row, { ...all, kind: 'CT' })).length, 3);
  assert.deepEqual(ROWS.filter(row => simShown(row, { ...all, writable: true })).map(row => row.type), ['integer']);
  assert.deepEqual(ROWS.filter(row => simShown(row, { ...all, kind: 'MV', text: 'gauge' })).map(row => row.type), ['gauge32']);
  assert.equal(ROWS.filter(row => simShown(row, { ...all, kind: 'MV', text: 'octets' })).length, 0);
  const read = initialValues(READ_FILTER);
  assert.deepEqual(ROWS.filter(row => readShown(row, { ...read, key: true })).map(row => row.type), ['counter32']);
  assert.deepEqual(ROWS.filter(row => readShown(row, { ...read, polled: true, kind: 'TX' })).map(row => row.type), ['octets']);
});

const leaf = (path: string, type: string, fc = 'ST'): Attribute => ({ ref: `LD/XCBR1.Pos.${path}`, fc, path, type });

test('Explorer maps MMS type words onto kinds by FC and type, not by name', () => {
  assert.equal(basicTypeOf('float', 'f'), 'FLOAT32');
  assert.equal(basicTypeOf('UTC time', 't'), 'Timestamp');
  assert.equal(attributeKind('MX', 'mag.f', 'float'), 'MV');
  assert.equal(attributeKind('ST', 'stVal', 'boolean'), 'SP');
  assert.equal(attributeKind('ST', 'stVal', 'bit string'), 'DP', 'a bit-string status value is a double position');
  assert.equal(attributeKind('ST', 'stVal', 'integer'), 'IS');
  assert.equal(attributeKind('ST', 'q', 'bit string'), 'AT');
  assert.equal(attributeKind('ST', 't', 'UTC time'), 'AT');
  assert.equal(attributeKind('SP', 'setMag.f', 'float'), 'CF');
  assert.equal(attributeKind('DC', 'd', 'visible string'), 'TX');
  assert.equal(attributeKind('ST', 'x', 'structure'), 'UN', 'unknown is shown, never guessed');
  assert.equal(attributeKind('ST', 'x', 'type 99'), 'UN');
});

test('only the control value of a CO structure is a command; SPC and DPC are told apart by the status value', () => {
  assert.equal(attributeKind('CO', 'Oper.ctlVal', 'boolean'), 'CM', 'on its own a boolean command is a plain command');
  assert.equal(attributeKind('CO', 'Oper.ctlVal', 'float'), 'SE');
  for (const helper of ['Oper.ctlNum', 'Oper.T', 'Oper.Test', 'Oper.Check', 'Oper.origin.orCat']) {
    assert.equal(attributeKind('CO', helper, 'integer'), 'AT', helper);
  }
  const breaker = [leaf('stVal', 'bit string'), leaf('q', 'bit string'), leaf('Oper.ctlVal', 'boolean', 'CO'), leaf('Oper.T', 'UTC time', 'CO')];
  assert.deepEqual(objectKinds(breaker), ['DP', 'AT', 'DC', 'AT']);
  const single = [leaf('stVal', 'boolean'), leaf('Oper.ctlVal', 'boolean', 'CO')];
  assert.deepEqual(objectKinds(single), ['SP', 'SC']);
});

const item = (ref: string, fc: string, type: string, status: WatchItem['status'] = 'ok'): WatchItem =>
  ({ ref, fc, type, text: '1', status, at: null });

test('the Explorer watch list and node table filter by type chip and hide quality and time helpers', () => {
  const power = item('LD/MMXU1.TotW.mag.f', 'MX', 'float');
  const position = item('LD/XCBR1.Pos.stVal', 'ST', 'bit string');
  assert.equal(watchKind(power), 'MV');
  assert.equal(watchKind(position), 'DP');
  assert.equal(WATCH_FILTER.groups?.[0].key, 'kind');
  const all = initialValues(WATCH_FILTER);
  assert.ok(watchMatches(power, all, 0, 0) && watchMatches(position, all, 0, 0));
  assert.ok(watchMatches(power, { ...all, kind: 'MV' }, 0, 0));
  assert.ok(!watchMatches(position, { ...all, kind: 'MV' }, 0, 0));
  const quality = leaf('q', 'bit string');
  const node = initialValues(NODE_FILTER);
  assert.ok(attributeShown('Pos', quality, 'AT', node));
  assert.ok(!attributeShown('Pos', quality, 'AT', { ...node, helpers: true }));
  assert.ok(attributeShown('Pos', leaf('stVal', 'bit string'), 'DP', { ...node, kind: 'DP' }));
  assert.ok(!attributeShown('Pos', leaf('stVal', 'bit string'), 'DP', { ...node, kind: 'SP' }));
  assert.ok(attributeShown('Pos', leaf('stVal', 'bit string'), 'DP', { ...node, text: 'pos' }), 'the object name is searched too');
});

test('the tool header is one slim row: back, title with a "?", and the tool\'s own switch', () => {
  const page = read('features/tools/ToolsPage.tsx');
  assert.match(page, /<HelpTip text=\{toolHelp\(card\)\} label=\{TOOLS_COPY\.whatIsThis\} \/>/);
  const snmp = read('features/tools/snmp/SnmpTool.tsx');
  assert.doesNotMatch(snmp, /snmp-role-sentence/, 'the long role sentence moved behind the "?"');
  assert.match(snmp, /SNMP_ROLES\[role\]\.sentence\} \$\{SNMP_COPY\.roleHint/);
  assert.match(read('features/tools/tools.css'), /\.tool-header > button \{ min-height: 32px; \}/);
});

test('SNMP values are one line each with edit pins, the type badge and a menu for the rare actions', () => {
  const row = read('features/tools/snmp/simulate/PointRow.tsx');
  assert.match(row, /<KindBadge kind=\{kind\} raw=/);
  assert.match(row, /\{\.\.\.kindRow\(kind\)\}/);
  assert.match(read('features/tools/snmp/simulate/PointName.tsx'), /<EditableText/);
  const editor = read('features/tools/snmp/simulate/ValueEditor.tsx');
  assert.match(editor, /<AutoValue/);
  assert.match(editor, /<EditableText/);
  const table = read('features/tools/snmp/simulate/PointsTable.tsx');
  assert.match(table, /snmp-values compact-rows/);
  assert.match(table, /spec=\{SIM_FILTER\}/);
  const actions = read('features/tools/snmp/simulate/PointsActions.tsx');
  assert.match(actions, /<Menu label=\{POINTS_COPY\.moreLabel\}/);
  assert.match(actions, /exportCsv/);
  const panel = read('features/tools/snmp/simulate/DevicePanel.tsx');
  assert.match(panel, /<Menu label=\{SNMP_COPY\.deviceMenu\}/);
  assert.match(panel, /window\.confirm\(SNMP_COPY\.removeConfirm\(device\.name\)\)/, 'Delete device still asks first');
  const readTable = read('features/tools/snmp/read/ValuesTable.tsx');
  assert.match(readTable, /snmp-values compact-rows/);
  assert.match(readTable, /spec=\{READ_FILTER\}/);
  assert.match(read('features/tools/snmp/read/TrapLog.tsx'), /<KindBadge/);
});

test('my CSS takes its colours from tokens, and the explorer confirm box is the shared one', () => {
  for (const path of ['features/tools/explorer/explorer.css', 'features/tools/snmp/snmp.css', 'features/tools/tools.css',
    'features/tools/find/find.css', 'features/network/goose.css']) {
    assert.doesNotMatch(read(path), /#[0-9a-fA-F]{3,6}\b/, `${path} has a hard-coded colour`);
  }
  assert.match(read('features/tools/explorer/CommandPanel.tsx'), /className="confirm-box command-confirm"/);
  assert.match(read('features/network/goose.css'), /\.goose-confirm \{ border: 1\.5px solid var\(--error-line\)/);
  assert.match(read('features/tools/explorer/ObjectTable.tsx'), /<KindBadge kind=\{kinds\[index\]\} raw=\{attribute\.type\} \/>/);
});
