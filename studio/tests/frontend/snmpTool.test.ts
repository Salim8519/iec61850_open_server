// ADR-0092: the SNMP devices tool's plain words, its search and its running dot.
import assert from 'node:assert/strict';
import test from 'node:test';
import type { Activity } from '../../frontend/src/api/activity.ts';
import type { ValueRow } from '../../frontend/src/api/snmp.ts';
import { toolActivity } from '../../frontend/src/app/running.ts';
import { AUTH_NAMES, LEVELS, PRIV_NAMES, SNMP_ROLES } from '../../frontend/src/copy/snmp.ts';
import { rowMatches } from '../../frontend/src/features/tools/snmp/rowMatch.ts';

const idle: Activity = {
  gateway: { state: 'stopped', protocol: 'iec104', port: null, clients: 0 },
  explorer: { state: 'idle', host: '', polling: false, watched: 0 },
  scan: { state: 'idle', probed: 0, total: 0 },
  master: { state: 'disconnected', protocol: 'iec104', host: '', points: 0 },
  sequences: 0,
};

test('the two roles read Simulate devices and Read devices, each with one sentence', () => {
  assert.deepEqual(Object.values(SNMP_ROLES).map(role => role.label), ['Simulate devices', 'Read devices']);
  for (const role of Object.values(SNMP_ROLES)) {
    assert.ok(role.sentence.endsWith('.') && role.sentence.split('. ').length === 1, role.sentence);
  }
});

test('security choices are short enough to show in full in a narrow column', () => {
  for (const text of [...Object.values(LEVELS), ...Object.values(AUTH_NAMES)]) assert.ok(text.length <= 24, text);
  assert.deepEqual(Object.keys(PRIV_NAMES), ['des', 'aes128', 'aes192', 'aes256', 'aes192c', 'aes256c']);
  assert.deepEqual(Object.keys(AUTH_NAMES), ['md5', 'sha', 'sha224', 'sha256', 'sha384', 'sha512']);
});

test('the value search finds names, OIDs, values and meanings', () => {
  const row: ValueRow = { oid: '1.3.6.1.2.1.2.2.1.8.3', type: 'integer', value: '2', name: 'IF-MIB::ifOperStatus.3',
    meaning: 'down', writable: false };
  for (const text of ['ifoperstatus', '2.2.1.8', 'DOWN', '', '  ']) assert.ok(rowMatches(row, text), text);
  assert.ok(!rowMatches(row, 'sysName'));
});

test('the Tools card says what SNMP is doing, and nothing when it is idle', () => {
  assert.equal(toolActivity({ ...idle, snmp: { running: 0, reader: 'disconnected', target: '', listening: false } }).snmp,
    undefined);
  const busy = toolActivity({ ...idle, snmp: { running: 2, reader: 'connected', target: '127.60.0.1:161 (v2c, community public)',
    listening: true } }).snmp;
  assert.equal(busy, 'SNMP: 2 simulated devices answering · SNMP: reading 127.60.0.1:161 · SNMP: receiving traps');
  assert.equal(toolActivity(idle).snmp, undefined); // the Python engine has no SNMP status
});
