import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { gooseStatus, TONES } from '../../frontend/src/copy/status.ts';
import { endpointOf, shortBlock } from '../../frontend/src/features/network/endpoints.ts';

const native = ['goose_traffic.cpp', 'goose_traffic.hpp']
  .map(name => readFileSync(new URL(`../../native/live/${name}`, import.meta.url), 'utf8')).join('\n');

test('every goose state has text icon and hint', () => {
  const states = new Set([...native.matchAll(/state = "([a-z]+)"/g), ...native.matchAll(/"state", "([a-z]+)"/g)]
    .map(match => match[1]));
  for (const match of native.matchAll(/auto state = .*\? "([a-z]+)" : "([a-z]+)"/g)) { states.add(match[1]); states.add(match[2]); }
  states.add('stopped');
  const problems = new Set([...native.matchAll(/problem = "([a-z_]+)"/g)].map(match => match[1]));
  assert.ok(states.has('receiving') && states.has('timeout') && problems.has('conf_rev_mismatch'));
  for (const state of states) {
    const status = gooseStatus(state);
    assert.ok(status.label && status.hint && TONES[status.tone].icon, `GOOSE state ${state}`);
  }
  for (const problem of problems) {
    const status = gooseStatus('problem', problem);
    assert.notEqual(status.label, gooseStatus('problem').label, `Problem ${problem} needs its own wording`);
    assert.ok(status.hint.length > 20);
  }
});

test('endpoints follow the network mode like the backend', () => {
  const ied = { ip: '127.50.0.2', port: 102, lab_ip: '192.168.50.21' };
  assert.deepEqual(endpointOf(ied, { mode: 'loopback', adapter_guid: '', adapter_label: '' }), { host: '127.50.0.2', port: 102 });
  assert.deepEqual(endpointOf(ied, { mode: 'lab', adapter_guid: '{1}', adapter_label: 'x' }), { host: '192.168.50.21', port: 102 });
  assert.deepEqual(endpointOf({ ...ied, lab_ip: '' }, { mode: 'lab', adapter_guid: '{1}', adapter_label: 'x' }).host, '127.50.0.2');
  assert.equal(shortBlock('IED1LD0/LLN0$GO$gcbEvents'), 'gcbEvents');
});
