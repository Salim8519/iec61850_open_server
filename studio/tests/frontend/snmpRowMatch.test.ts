// Change 0151: the SNMP values search also finds the user's own labels.
import assert from 'node:assert/strict';
import test from 'node:test';
import type { ValueRow } from '../../frontend/src/api/snmp.ts';
import { rowMatches } from '../../frontend/src/features/tools/snmp/rowMatch.ts';

const row: ValueRow = { oid: '1.3.6.1.2.1.2.2.1.8.1', label: 'Uplink port 1', type: 'integer', value: '1',
  name: 'IF-MIB::ifOperStatus.1', meaning: 'up', writable: false };

test('search matches label, name, OID, value and meaning, ignoring case', () => {
  for (const text of ['uplink', 'ifOperStatus', '2.2.1.8', 'UP', '']) assert.equal(rowMatches(row, text), true, text);
  assert.equal(rowMatches(row, 'downlink'), false);
  assert.equal(rowMatches({ ...row, label: undefined }, 'uplink'), false);
});
