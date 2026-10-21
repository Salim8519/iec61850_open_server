// Change 0142: the three IED roles have owner-approved names and one plain sentence each;
// a monitored real device shows its own address, not the simulated one.
import assert from 'node:assert/strict';
import test from 'node:test';
import { ROLE_COPY, ROLE_HELP, ROLE_ORDER } from '../../frontend/src/copy/roles.ts';
import { roleStatus } from '../../frontend/src/copy/status.ts';
import { realEndpointOf } from '../../frontend/src/features/network/endpoints.ts';
import { LOOPBACK_CHOICE } from '../../frontend/src/features/network/setupWords.ts';

test('the roles read Simulate, Monitor real device and Skip, each with one sentence', () => {
  assert.deepEqual(ROLE_ORDER.map(role => ROLE_COPY[role].label), ['Simulate', 'Monitor real device', 'Skip']);
  for (const role of ROLE_ORDER) {
    const sentence = ROLE_COPY[role].sentence;
    assert.ok(sentence.endsWith('.') && sentence.split('. ').length === 1, sentence);
    assert.ok(ROLE_HELP.includes(`${ROLE_COPY[role].label}: ${sentence}`));
  }
  assert.ok(!/Be this IED|Read real|Leave out|Left out/.test(ROLE_HELP));
});

test('list badges use the new words', () => {
  assert.equal(roleStatus('off', 'stopped').label, 'Skipped');
  assert.equal(roleStatus('real', 'stopped').label, 'Monitors real device');
  assert.equal(roleStatus('simulated', 'stopped').label, 'Simulate · stopped');
});

test('a real device is reached at the typed address, else its SCD address', () => {
  const ied = (live: { ip: string; port: number }, scd: string | null) => ({
    live: { ...live, access: 'auto_reports' as const, poll_ms: 1000 },
    scl_address: scd === null ? null : { ip: scd, subnet: '', gateway: '', access_point: 'S1' } });
  assert.deepEqual(realEndpointOf(ied({ ip: '192.168.1.20', port: 102 }, '10.0.0.5')), { host: '192.168.1.20', port: 102 });
  assert.deepEqual(realEndpointOf(ied({ ip: '', port: 2102 }, '10.0.0.5')), { host: '10.0.0.5', port: 2102 });
  assert.deepEqual(realEndpointOf(ied({ ip: '', port: 102 }, null)), { host: '', port: 102 });
});

test('the network picker offers this PC only under a value no adapter uses', () => {
  assert.equal(LOOPBACK_CHOICE, 'loopback');
});
