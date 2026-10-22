// "Save always" on SNMP device settings, the gateway point list, test sequences and Add point:
// the rules that keep a wrong value out of the save, and the places where the Save buttons used to be.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { GatewayPoint } from '../../frontend/src/api/gateway.ts';
import type { SnmpDevice } from '../../frontend/src/api/snmp.ts';
import type { Scenario } from '../../frontend/src/api/types.ts';
import { deviceProblem, isIpv4, pickSettings, SETTING_KEYS, trapsProblem, type DeviceSettings }
  from '../../frontend/src/features/tools/snmp/simulate/deviceRules.ts';
import { clearDraft, isLockedAnswer, keepDraft, keptDraft, saveProblems } from '../../frontend/src/features/tools/gateway/pointSave.ts';
import { canSave, sequenceProblem, withSequence } from '../../frontend/src/features/test/sequenceSave.ts';

const SRC = new URL('../../frontend/src/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, SRC), 'utf8');

const device: SnmpDevice = { id: 'sw1', name: 'Switch', profile: 'switch', ip: '127.60.0.1', port: 161, v1: false, v2c: true, v3: false,
  read_community: 'public', write_community: 'private', users: [], engine_id: '800000090300001122334455', traps: [],
  points: [{ oid: '1.3.6.1.2.1.1.5.0', type: 'octets', value: 'x' }] };
const settings = (change: Partial<DeviceSettings> = {}): DeviceSettings => ({ ...pickSettings(device), ...change });
const v3User = { name: 'admin', level: 'authPriv' as const, auth: 'sha256', auth_password: 'longpassword', priv: 'aes128', priv_password: 'longpassword' };

test('SNMP settings: a good device has no problem, and only the settings fields go out (never the values)', () => {
  assert.equal(deviceProblem(settings()), null);
  assert.deepEqual(Object.keys(pickSettings(device)).sort(), [...SETTING_KEYS].sort());
  assert.ok(!('points' in pickSettings(device)) && !('id' in pickSettings(device)));
});

test('SNMP settings: each wrong value names its field, in words, and is not sent', () => {
  const field = (change: Partial<DeviceSettings>) => deviceProblem(settings(change))?.field;
  assert.equal(field({ name: '   ' }), 'name');
  assert.equal(field({ name: 'x'.repeat(61) }), 'name');
  assert.equal(field({ ip: '127.60.0' }), 'ip', 'a partial address');
  assert.equal(field({ ip: '300.1.1.1' }), 'ip');
  assert.equal(field({ ip: '01.2.3.4' }), 'ip', 'the engine refuses leading zeros');
  assert.equal(field({ ip: '' }), 'ip');
  assert.equal(field({ port: 0 }), 'port');
  assert.equal(field({ port: 70000 }), 'port');
  assert.equal(field({ v1: false, v2c: false, v3: false }), 'versions');
  assert.equal(field({ read_community: '' }), 'community');
  assert.equal(field({ engine_id: '8000' }), 'engine', 'fewer than 5 bytes');
  assert.equal(field({ engine_id: '80000009030000112233445' }), 'engine', 'an odd number of digits');
  assert.equal(field({ engine_id: 'zz00000903000011' }), 'engine');
  assert.match(deviceProblem(settings({ ip: '1.2' }))!.text, /not an IPv4 address/);
  assert.match(deviceProblem(settings({ port: 0 }))!.text, /1 to 65535/);
});

test('SNMP settings: a v3 user needs a name, both passwords of 8 characters and a unique name', () => {
  const v3 = (users: SnmpDevice['users']) => deviceProblem(settings({ v3: true, v2c: false, users }));
  assert.equal(v3([])?.field, 'users');
  assert.equal(v3([v3User]), null);
  assert.equal(v3([{ ...v3User, name: '' }])?.field, 'users');
  assert.match(v3([{ ...v3User, auth_password: 'short' }])!.text, /authentication password needs at least 8/);
  assert.match(v3([{ ...v3User, priv_password: '' }])!.text, /encryption password needs at least 8/);
  assert.equal(v3([{ ...v3User, level: 'authNoPriv', priv_password: '' }]), null, 'no encryption asked for');
  assert.equal(v3([{ ...v3User, level: 'noAuthNoPriv', auth_password: '' }]), null);
  assert.match(v3([v3User, v3User])!.text, /used twice/);
});

test('SNMP traps: receiver address, port, community and v3 user are checked like the engine does', () => {
  const trap = { host: '127.0.0.1', port: 162, version: 'v2c' as const, community: 'public' };
  const problem = (target: object) => trapsProblem(settings({ traps: [{ ...trap, ...target }] }))?.field;
  assert.equal(problem({}), undefined);
  assert.equal(problem({ host: 'monitor' }), 'traps');
  assert.equal(problem({ port: 0 }), 'traps');
  assert.equal(problem({ community: '' }), 'traps');
  assert.equal(problem({ version: 'v3', user: 'ghost' }), 'traps');
  assert.equal(problem({ version: 'v1', inform: true }), 'traps');
  assert.equal(deviceProblem(settings({ traps: [{ ...trap, host: 'x' }] }))?.field, 'traps', 'a bad receiver also holds back the settings');
  assert.ok(isIpv4('10.0.0.255') && !isIpv4('10.0.0.256'));
});

test('the gateway point list: a refusal because the gateway runs is told apart from a real problem', () => {
  assert.ok(isLockedAnswer(new Error('Stop the gateway before changing its point list')));
  assert.ok(!isLockedAnswer(new Error('Some points need attention before they can be saved')));
  assert.ok(!isLockedAnswer('Stop the gateway before changing its point list'), 'only real errors count');
  assert.deepEqual(saveProblems(new Error('x'), ['Point 3: address 70000 is not valid']), ['Point 3: address 70000 is not valid']);
  assert.deepEqual(saveProblems(new Error('The engine is busy'), []), ['The engine is busy']);
});

test('a point list kept while the gateway runs comes back for its own protocol until it is saved or discarded', () => {
  const row: GatewayPoint = { kind: 'single', address: 1, ied: 'a', ref: 'X', scale: 1, description: 'kept' };
  keepDraft('iec104', [row]);
  assert.equal(keptDraft('modbus'), undefined);
  assert.deepEqual(keptDraft('iec104'), [row]);
  assert.deepEqual(keptDraft('iec104'), [row], 'asking twice (React mounts twice in development) still finds it');
  clearDraft('iec104');
  assert.equal(keptDraft('iec104'), undefined);
});

const step = { id: 's1', at: 0, ied_id: 'a', ref: 'X', action: 'set' as const, value: 1 };
const sequence: Scenario = { id: 'q1', name: 'Trip test', enabled: false, steps: [step] };

test('sequences: a name and at least one step are needed, a new empty one is only "not saved yet"', () => {
  assert.equal(sequenceProblem(sequence, true), null);
  assert.equal(sequenceProblem({ ...sequence, name: ' ' }, true)?.field, 'name');
  assert.equal(sequenceProblem({ ...sequence, name: 'x'.repeat(101) }, false)?.field, 'name');
  assert.equal(sequenceProblem({ ...sequence, steps: [] }, false), null, 'a new sequence waits for its first step');
  assert.equal(canSave({ ...sequence, steps: [] }, false), false);
  assert.equal(sequenceProblem({ ...sequence, steps: [] }, true)?.field, 'steps', 'a saved one cannot lose its last step');
  assert.equal(canSave(sequence, false), true);
  assert.equal(sequenceProblem({ ...sequence, steps: Array.from({ length: 1001 }, () => step) }, true)?.field, 'steps');
});

test('sequences: the edited one keeps its place in the saved list, a new one goes last', () => {
  const other: Scenario = { ...sequence, id: 'q0', name: 'First' };
  const list = withSequence([other, sequence], { ...sequence, name: 'Renamed' });
  assert.deepEqual(list.map(item => item.name), ['First', 'Renamed']);
  assert.deepEqual(withSequence([other], sequence).map(item => item.id), ['q0', 'q1']);
});

test('the Save buttons are gone from the four places; each saves through the shared saver', () => {
  const settingsPage = read('features/tools/snmp/simulate/DeviceSettings.tsx');
  assert.doesNotMatch(settingsPage, /type="submit"|Save settings|SETTINGS_COPY\.save/);
  assert.match(settingsPage, /<SaveStatus/);
  assert.doesNotMatch(read('features/tools/snmp/simulate/TrapPanel.tsx'), /SETTINGS_COPY\.save|onClick=\{store\}/);
  const toolbar = read('features/tools/gateway/PointToolbar.tsx');
  assert.doesNotMatch(toolbar, /GATEWAY_COPY\.save|unsaved|Save point list/);
  assert.match(toolbar, /<SaveStatus state=\{state\}/);
  assert.doesNotMatch(read('features/test/SequenceEditor.tsx'), /Save sequence/);
  assert.match(read('features/test/SequenceEditor.tsx'), /useAutoSave<Scenario>/);
  assert.match(read('features/tools/gateway/usePointList.ts'), /useAutoSave<PointSave>/);
  assert.match(read('features/tools/snmp/simulate/useDeviceDraft.ts'), /useAutoSave<DeviceSettings>/);
});

test('the old unsaved-changes words are gone from the gateway copy', () => {
  const copy = read('copy/gateway.ts');
  assert.doesNotMatch(copy, /Save point list|You have unsaved changes|'Discard changes'|Save the point list/);
  assert.match(copy, /willSave: 'Will save when the gateway stops'/);
  const snmp = read('copy/snmp.ts');
  assert.doesNotMatch(snmp, /Save settings/);
});

test('whole-list actions keep a way back: Remove, Propose and Import offer Undo', () => {
  const table = read('features/tools/gateway/PointTable.tsx');
  assert.match(table, /edit\(value\.points, rows\)/, 'propose and import hand over the list before');
  assert.match(table, /edit\(left, rows\)/, 'removing hands over the list before');
  assert.match(table, /GATEWAY_COPY\.undo/);
  assert.match(read('features/tools/gateway/usePointList.ts'), /discard/);
});
