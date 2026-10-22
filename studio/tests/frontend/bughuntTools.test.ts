// Change 0178 (bug hunt, Tools): the pure parts of the gateway, master, converter and SNMP fixes.
import assert from 'node:assert/strict';
import test from 'node:test';
import { ADD_COPY } from '../../frontend/src/copy/gatewayAdd.ts';
import { refusedWrite } from '../../frontend/src/copy/snmpRead.ts';
import { converterTabs } from '../../frontend/src/features/tools/gateway/converter/converterTabs.ts';
import { hasWordOrder } from '../../frontend/src/features/tools/gateway/master/wordOrder.ts';

test('C3: only the 2-register Modbus writes offer a word order', () => {
  assert.deepEqual(['registers_int32', 'registers_float', 'register', 'coil', 'setpoint_float'].map(kind => hasWordOrder(kind as never)),
    [true, true, false, false, false]);
});

test('C4: a converter reading a Modbus device has the What to read tab after the device points', () => {
  assert.deepEqual(converterTabs(true).slice(0, 2), [['device', 'Device points'], ['ranges', 'What to read']]);
  assert.ok(!converterTabs(false).some(([key]) => key === 'ranges'));
  assert.equal(converterTabs(true).length, converterTabs(false).length + 1);
});

test('C5: a refused SNMP change explains the write community; other errors stay as they are', () => {
  const text = refusedWrite('the device refused 1.3.6.1.2.1.1.5.0: no access');
  assert.match(text, /^the device refused 1\.3\.6\.1\.2\.1\.1\.5\.0: no access\. Changing values usually needs the write community/);
  assert.equal(refusedWrite('No answer from 127.0.0.1:161'), 'No answer from 127.0.0.1:161');
});

test('C10: the status-pairing example uses the addresses of the chosen protocol', () => {
  assert.match(ADD_COPY.withStatusHint('iec104'), /20003 for command 50003/);
  for (const protocol of ['dnp3', 'modbus'] as const) {
    assert.doesNotMatch(ADD_COPY.withStatusHint(protocol), /50003/);
  }
  assert.match(ADD_COPY.withStatusHint('dnp3'), /index 3/);
  assert.match(ADD_COPY.withStatusHint('modbus'), /coil 3/);
});
