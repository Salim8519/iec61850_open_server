// ADR-0107: tap position, step command and set point gateway kinds are listed wherever kinds are.
import assert from 'node:assert/strict';
import test from 'node:test';
import { KIND_HINTS, KIND_LABELS, KIND_WHERE, OWN_KIND_LABELS } from '../../frontend/src/copy/gateway.ts';
import { TYPED_KINDS } from '../../frontend/src/copy/gatewayTypedKinds.ts';
import { commandLabels, commandValues, parseSetpoint } from '../../frontend/src/features/tools/gateway/gatewayCommands.ts';
import { ownTargets, parseOwnNumber } from '../../frontend/src/features/tools/gateway/ownPoints.ts';
import { kindOfGatewayPoint } from '../../frontend/src/points/pointKind.ts';

const TYPED = ['step_position', 'command_step', 'command_setpoint'] as const;

test('every typed kind has a name, type IDs per protocol and a hint', () => {
  for (const kind of TYPED) {
    assert.ok(KIND_LABELS[kind] && OWN_KIND_LABELS[kind] && KIND_HINTS[kind], kind);
    for (const protocol of ['iec104', 'modbus', 'dnp3'] as const) assert.ok(KIND_WHERE[protocol][kind], `${kind} ${protocol}`);
  }
  assert.equal(KIND_WHERE.iec104.command_step, 'C_RC (47 / 60)');
  assert.equal(KIND_WHERE.iec104.command_setpoint, 'C_SE (48-50 / 61-63)');
  assert.equal(KIND_WHERE.iec104.step_position, 'M_ST (5 / 32)');
  assert.equal(Object.keys(TYPED_KINDS).length, 3);
  assert.equal(KIND_LABELS.command_double, 'Double command', 'the six old kinds keep their words');
});

test('typed kinds get the universal badge kinds', () => {
  assert.equal(kindOfGatewayPoint('step_position'), 'IS');
  assert.equal(kindOfGatewayPoint('command_step'), 'SE');
  assert.equal(kindOfGatewayPoint('command_setpoint'), 'SE');
  assert.equal(kindOfGatewayPoint('command_double'), 'DC');
});

test('the command test buttons send raise/lower and set point numbers', () => {
  assert.deepEqual(commandLabels({ kind: 'command_step' }), ['Raise', 'Lower']);
  assert.deepEqual(commandValues({ kind: 'command_step' }), ['higher', 'lower']);
  assert.deepEqual(commandValues({ kind: 'command_double' }), [true, false]);
  assert.deepEqual(commandLabels({ kind: 'command_double' }), ['Close', 'Open']);
  assert.deepEqual(parseSetpoint(' 12,5 '), { value: 12.5 });
  assert.ok('error' in parseSetpoint('abc'));
  assert.ok('error' in parseSetpoint(''));
});

test('own tap positions and what own step and set point commands drive', () => {
  assert.deepEqual(parseOwnNumber('5', 'step_position'), { value: 5 });
  assert.ok('error' in parseOwnNumber('64', 'step_position'));
  assert.ok('error' in parseOwnNumber('2.5', 'step_position'));
  const own = (id: string, kind: 'single' | 'step_position' | 'float') =>
    ({ id, kind, source: 'manual', address: 1, ied: '', ref: '', scale: 1, description: id });
  const rows = [own('a', 'single'), own('b', 'step_position'), own('c', 'float')];
  assert.deepEqual(ownTargets(rows).map(choice => choice.value), ['a']);
  assert.deepEqual(ownTargets(rows, 'command_step').map(choice => choice.value), ['b']);
  assert.deepEqual(ownTargets(rows, 'command_setpoint').map(choice => choice.value), ['b', 'c']);
});
