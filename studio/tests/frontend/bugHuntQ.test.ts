// Lane Q bug hunt (2026-10-20): defects found by testing every page of the running app at 1366x768.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { countOf } from '../../frontend/src/copy/count.ts';
import { changeSentence, putLabel } from '../../frontend/src/features/network/setupWords.ts';
import { runWords, stepAction, stepCount } from '../../frontend/src/features/test/sequenceWords.ts';
import { createActionQueue } from '../../frontend/src/features/tools/gateway/actionQueue.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('a setting saved on blur finishes before the Start pressed in the same gesture', async () => {
  const queue = createActionQueue();
  const order: string[] = [];
  let release!: () => void;
  const saving = queue(() => new Promise<void>(resolve => { release = () => { order.push('save'); resolve(); }; }));
  const starting = queue(async () => { order.push('start'); });
  await Promise.resolve();
  assert.deepEqual(order, [], 'start waits for the save');
  release();
  await Promise.all([saving, starting]);
  assert.deepEqual(order, ['save', 'start']);
});

test('a failed save does not block the next request and still reports its own error', async () => {
  const queue = createActionQueue();
  const failed = queue(async () => { throw new Error('port in use'); });
  await assert.rejects(failed, /port in use/);
  assert.equal(await queue(async () => 'started'), 'started');
});

test('SCADA tools save settings quietly so the Start/Connect press is not swallowed', () => {
  for (const path of ['features/tools/gateway/GatewayTool.tsx', 'features/tools/gateway/master/MasterTool.tsx',
    'features/tools/gateway/converter/ConverterTool.tsx']) {
    const source = read(path);
    assert.match(source, /createActionQueue/, path);
    assert.match(source, /saveConfig\([^)]*\)[^\n]*, true\)/, `${path}: saves pass quiet = true`);
  }
});

test('a slow master connect and gateway start say what is happening', () => {
  assert.match(read('features/tools/gateway/master/MasterConnect.tsx'), /<Spinner label=\{MASTER_COPY\.connecting\(/);
  assert.match(read('features/tools/gateway/GatewaySettings.tsx'), /<Spinner label=\{copy\.starting\}/);
});

test('the network sentence names the firewall ports of the project, not always 102', () => {
  const adapter = { guid: 'G', name: 'Ethernet 2', description: '', status: 'Up', index: 1, speed: '', kind: 'ethernet',
    default_route: false, addresses: [] };
  const preview = (ports?: number[]) => ({ adapter, rows: [], fingerprint: 'f', ready: true, needs_admin: true,
    items: [{ kind: 'alias' as const, action: 'add' as const, owned: true, description: '' },
      { kind: 'firewall' as const, action: 'add' as const, owned: true, description: '', ports }] });
  assert.match(changeSentence(preview([20102])), /allow port 20102 for them/);
  assert.match(changeSentence(preview([102, 20102])), /allow ports 102, 20102 for them/);
  assert.match(changeSentence(preview()), /allow port 102 for them/);
});

test('the network button never shows a gap where the count is missing', () => {
  assert.equal(putLabel(0), 'Put IEDs on this card');
  assert.equal(putLabel(1, 'Ethernet 2'), 'Put 1 IED on “Ethernet 2”');
  assert.equal(putLabel(4, 'VMnet1'), 'Put 4 IEDs on “VMnet1”');
});

test('timed sequences use the Action words and say which step failed', () => {
  assert.equal(stepAction('set'), 'Set value');
  assert.equal(stepAction('assert'), 'Check value');
  assert.equal(runWords(undefined, undefined, 4), 'Ready');
  assert.equal(runWords('running', 1, 4), 'Running');
  assert.equal(runWords('completed', 4, 4), 'Finished');
  assert.equal(runWords('cancelled', 2, 4), 'Stopped');
  assert.equal(runWords('failed', 3, 4), 'Failed at step 4 of 4');
  assert.equal(stepCount(1), '1 step');
  assert.equal(stepCount(4), '4 steps');
});

test('counts read naturally: 1 note, 2 notes', () => {
  assert.equal(countOf(1, 'note'), '1 note');
  assert.equal(countOf(2, 'note'), '2 notes');
  assert.equal(countOf(1, 'logical device'), '1 logical device');
  assert.equal(countOf(0, 'check'), '0 checks');
});

test('Add test point explains why it is off while running', () => {
  assert.match(read('features/test/TestPoints.tsx'), /title=\{running \? STOP_TO_ADD : undefined\}/);
});

test('Devices list rows wrap full names and hints instead of cutting them', () => {
  assert.match(read('features/devices/devices.css'),
    /\.list-row\.compact strong, \.list-row\.compact small \{ white-space: normal; overflow-wrap: anywhere; \}/);
});

test('every terminal is both a wire start and a wire end, so any side can be wired and drawn', () => {
  const node = read('equipment/EquipmentNode.tsx');
  assert.match(node, /<Handle id=\{id\} type="target" position=\{position\} \/><Handle id=\{id\} type="source"/);
  for (const side of ['north', 'west', 'east', 'south']) assert.match(node, new RegExp(`<Terminal id="${side}"`));
  assert.doesNotMatch(node, /<Handle id="(north|west|east|south)"/);
});
