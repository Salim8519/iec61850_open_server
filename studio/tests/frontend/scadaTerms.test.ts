import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { toolActivity } from '../../frontend/src/app/running.ts';
import { GLOSSARY } from '../../frontend/src/copy/glossary.ts';
import { MASTER_COPY, ROLE_NAMES, SIDES, TARGET_WORD } from '../../frontend/src/copy/master.ts';

const SRC = new URL('../../frontend/src/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, SRC), 'utf8');
const COPY = ['copy/master.ts', 'copy/gateway.ts', 'copy/glossary.ts', 'copy/tools.ts', 'copy/running.ts', 'copy/converter.ts',
  'features/tools/gateway/RoleSwitch.tsx'];

test('each protocol uses its own official words for both sides (ADR-0061)', () => {
  assert.deepEqual(SIDES.iec104, { us: 'controlling station (104 master)', device: 'controlled station (104 slave)',
    usShort: '104 master', deviceShort: '104 slave' });
  assert.equal(SIDES.dnp3.device, 'DNP3 outstation');
  assert.deepEqual([SIDES.modbus.us, SIDES.modbus.device], ['Modbus client', 'Modbus server']);
  assert.deepEqual(Object.values(ROLE_NAMES), ['Station gateway', 'SCADA master', 'Protocol converter']);
  for (const path of COPY) {
    const text = read(path);
    for (const wrong of [/DNP3 slave/i, /Modbus master/i, /Modbus slave/i, /outstation master/i, /104 client/i]) {
      assert.ok(!wrong.test(text), `${path} uses "${wrong.source}"`);
    }
  }
  for (const term of ['controlling_station', 'controlled_station', 'outstation', 'modbus_roles', 'rtu',
    'station_gateway', 'protocol_converter', 'upstream', 'integrity_poll']) {
    assert.ok(GLOSSARY[term]?.plain.length > 40, term);
  }
});

test('master commands leave the page only from the REAL-device confirmation', () => {
  const form = read('features/tools/gateway/master/MasterCommandForm.tsx');
  assert.ok(form.includes('onClick={send}') && form.includes('MASTER_COPY.realWarning'));
  assert.equal((form.match(/masterApi\.command\(/g) ?? []).length, 1);
  assert.ok(!read('features/tools/gateway/master/MasterPointTable.tsx').includes('masterApi.command('));
  assert.ok(/command', 'POST', \{ \.\.\.command, confirm: true \}/.test(read('api/master.ts')));
});

test('the tool card says what the master is doing', () => {
  const activity = { gateway: { state: 'stopped' as const, protocol: 'iec104', port: null, clients: 0 },
    explorer: { state: 'disconnected', host: '', polling: false, watched: 0 },
    scan: { state: 'idle', probed: 0, total: 0 }, sequences: 0,
    master: { state: 'connected' as const, protocol: 'iec104', host: '10.0.0.5', points: 412 } };
  assert.equal(toolActivity(activity).gateway, 'SCADA master: IEC 104 to 10.0.0.5 · 412 points');
  const lost = { ...activity, master: { ...activity.master, state: 'lost' as const } };
  assert.ok(toolActivity(lost).gateway?.includes('link lost, retrying'));
});

test('Modbus hints and forms never borrow IEC 104 or DNP3 ideas (change 0089)', () => {
  assert.ok(!MASTER_COPY.hostHint.includes('minutes'));
  assert.ok(!/master/i.test(MASTER_COPY.allowList), 'shown for Modbus too, where the word is client');
  assert.ok(MASTER_COPY.intervalHint.iec104.includes('all values'));
  assert.ok(!/minutes|interrogat|all values/.test(MASTER_COPY.intervalHint.modbus));
  const form = read('features/tools/gateway/master/MasterCommandForm.tsx');
  assert.ok(form.includes("protocol !== 'modbus' && <label"), 'Modbus has no select before operate');
  assert.ok(form.includes('coil: ON_OFF'), 'a coil is written ON or OFF, not typed as a number');
  const table = read('features/tools/gateway/master/MasterPointTable.tsx');
  assert.ok(table.includes('useEffect(() => setDraft(null), [status.protocol])'));
});

test('the command confirmation names a DNP3 output by its index (change 0089)', () => {
  assert.equal(MASTER_COPY.confirmTitle('Binary output (CROB)', 1, '192.168.98.1', TARGET_WORD.dnp3),
    'Send Binary output (CROB) to index 1 on 192.168.98.1?');
  assert.ok(MASTER_COPY.confirmTitle('Double command (C_DC)', 50002, 'h', TARGET_WORD.iec104).includes('address 50002'));
});
