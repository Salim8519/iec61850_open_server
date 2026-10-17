import assert from 'node:assert/strict';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';
import { TOOLS } from '../../frontend/src/copy/tools.ts';

const SRC = new URL('../../frontend/src/', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
const read = (path: string) => readFileSync(join(SRC, path), 'utf8');
const files = (folder: string): string[] => readdirSync(join(SRC, folder)).flatMap(name => {
  const path = `${folder}/${name}`;
  return statSync(join(SRC, path)).isDirectory() ? files(path) : /\.tsx$/.test(name) ? [path] : [];
});

test('every tool card has a name, one plain line and a hint', () => {
  assert.ok(TOOLS.length >= 1);
  assert.equal(new Set(TOOLS.map(card => card.id)).size, TOOLS.length);
  for (const card of TOOLS) {
    assert.ok(card.name.length > 3 && card.name.length <= 40, card.id);
    assert.ok(card.line.length > 20 && card.line.length <= 100, card.id);
    assert.ok(card.hint.length > 40 && card.hint.length <= 220, card.id);
  }
});

test('every listed tool opens a built tool view', () => {
  const page = read('features/tools/ToolsPage.tsx');
  for (const card of TOOLS) {
    assert.match(page, new RegExp(`tool === '${card.id}' && <`), `${card.id} has no view`);
    assert.match(page, new RegExp(`${card.id}: `), `${card.id} has no icon`);
  }
});

test('GOOSE and Sampled Values traffic is shown only in Tools', () => {
  const users = files('features').filter(path => /<(GooseTraffic|SvTraffic) /.test(read(path)));
  assert.deepEqual(users, ['features/tools/TrafficTool.tsx']);
  assert.match(read('features/settings/NetworkSettings.tsx'), /openTraffic/);
});

test('an open tool always offers the way back to the list', () => {
  assert.match(read('features/tools/ToolsPage.tsx'), /onClick=\{\(\) => openTool\(null\)\}/);
});

test('find IEDs proposes each adapter network, at most a /24, plus this PC', async () => {
  const { networkOf, rangeChoices } = await import('../../frontend/src/features/tools/find/ranges.ts');
  assert.equal(networkOf('192.168.10.37', 24), '192.168.10.0/24');
  assert.equal(networkOf('172.16.5.20', 16), '172.16.5.0/24');
  assert.equal(networkOf('10.0.0.37', 28), '10.0.0.32/28');
  const adapter = (status: string, ip: string) => ({ guid: ip, name: 'Lab', description: '', status, index: 1, speed: '',
    kind: 'ethernet', default_route: false, addresses: [{ ip, prefix: 24, skip_as_source: false, origin: '' }] });
  const choices = rangeChoices([adapter('Up', '10.1.2.3'), adapter('Down', '10.9.9.9'), adapter('Up', '169.254.1.1')], 'This PC');
  assert.deepEqual(choices.map(item => item.range), ['10.1.2.0/24', '127.50.0.0/24']);
});

test('explorer sends a command only through the confirmation step and never writes', () => {
  const sources = files('features/tools/explorer').map(read).join('\n') + read('api/tools.ts');
  for (const word of ['writeObject', '/write', 'setValue(']) {
    assert.ok(!sources.includes(word), `explorer must not mention ${word}`);
  }
  const callers = files('features/tools').filter(path => read(path).includes('explorerApi.operate('));
  assert.deepEqual(callers, ['features/tools/explorer/CommandPanel.tsx']);
  const panel = read('features/tools/explorer/CommandPanel.tsx');
  assert.ok(/onClick=\{\(\) => send\(pending\)\}/.test(panel), 'operate runs only from the confirm button');
  assert.ok(read('api/tools.ts').includes("confirm: true"), 'the API call carries the confirmation');
});

test('command choices follow the MMS types Explorer shows', async () => {
  const { commandChoices } = await import('../../frontend/src/features/tools/explorer/commandChoices.ts');
  const leaf = (path: string, type: string, fc = 'ST') => ({ ref: `X/Y.Pos.${path}`, fc, path, type });
  const breaker = commandChoices([leaf('stVal', 'bit string'), leaf('Oper.ctlVal', 'boolean', 'CO')]);
  assert.deepEqual(Array.isArray(breaker) && breaker.map(choice => choice.label), ['Close', 'Open']);
  const single = commandChoices([leaf('stVal', 'boolean'), leaf('Oper.ctlVal', 'boolean', 'CO')]);
  assert.deepEqual(Array.isArray(single) && single.map(choice => choice.label), ['On', 'Off']);
  assert.equal(commandChoices([leaf('Oper.ctlVal', 'integer', 'CO')]), 'number');
  assert.equal(commandChoices([leaf('stVal', 'boolean')]), null);
});
