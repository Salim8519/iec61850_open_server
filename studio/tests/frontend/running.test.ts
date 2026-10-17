import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { placeActivity, toolActivity } from '../../frontend/src/app/running.ts';
import { blockedReason, commandLabels } from '../../frontend/src/features/tools/gateway/gatewayCommands.ts';

const SRC = new URL('../../frontend/src/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, SRC), 'utf8');
const idle = { gateway: { state: 'stopped' as const, protocol: 'iec104', port: null, clients: 0 },
  explorer: { state: 'disconnected', host: '', polling: false, watched: 0 },
  scan: { state: 'idle', probed: 0, total: 0 }, sequences: 0 };
const project = (runtime: string) => ({ runtime, ieds: [{ role: 'simulated' }, { role: 'off' }] }) as never;

test('each place gets its own running dot with its own words', () => {
  assert.deepEqual(placeActivity(project('stopped'), idle), {});
  const run = placeActivity(project('running'), idle);
  assert.deepEqual(run.devices, { labels: ['1 IED running'], paused: false });
  assert.ok(run.diagram && !run.tools && !run.test);
  assert.equal(placeActivity(project('paused'), idle).devices?.paused, true);
  const busy = { ...idle, sequences: 2, gateway: { state: 'running' as const, protocol: 'iec104', port: 2404, clients: 1 },
    explorer: { state: 'connected', host: '10.0.0.5', polling: true, watched: 40 } };
  const places = placeActivity(project('stopped'), busy);
  assert.ok(!places.devices, 'a running gateway does not mean the IEDs run');
  assert.deepEqual(places.test?.labels, ['2 sequences running']);
  assert.deepEqual(places.tools?.labels, ['Gateway: IEC 104 on port 2404 · 1 SCADA master',
    'IED Explorer: connected to 10.0.0.5 · watching 40']);
  assert.deepEqual(Object.keys(toolActivity(busy)).sort(), ['explorer', 'gateway']);
});

test('the gateway never starts Run by itself; Run is offered as a button', () => {
  const notices = read('features/tools/gateway/GatewayNotices.tsx');
  assert.ok(notices.includes('onClick={runIeds}'));
  for (const path of ['features/tools/gateway/GatewayTool.tsx', 'features/tools/gateway/GatewayNotices.tsx']) {
    assert.ok(!/useEffect\([^)]*runIeds/s.test(read(path)), path);
  }
});

test('gateway commands are sent only from the warning, with the confirmation flag', () => {
  const buttons = read('features/tools/gateway/CommandSend.tsx');
  assert.ok(!buttons.includes('gatewayApi'), 'the Send buttons only open the warning');
  assert.ok(read('features/tools/gateway/PointTable.tsx').includes('send={() => send(pending)}'));
  assert.ok(/command', 'POST', \{ point_id, value, confirm: true \}/.test(read('api/gateway.ts')));
  const row = { id: 'p', kind: 'command_double' as const, address: 1, ied: 'a', ref: 'X/Y.Pos', scale: 1, description: '' };
  assert.deepEqual(commandLabels(row), ['Close', 'Open']);
  assert.deepEqual(commandLabels({ ...row, kind: 'command_single' }), ['On', 'Off']);
  assert.equal(blockedReason(row, false), '');
  assert.equal(blockedReason(row, true), 'Save the point list first');
  assert.ok(blockedReason({ ...row, monitor_only: true }, false).includes('real IEDs are off'));
});
