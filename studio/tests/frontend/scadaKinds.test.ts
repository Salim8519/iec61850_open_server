// ADR-0095 in the SCADA gateway, master and converter lists: type chips, counts, command and traffic types.
import assert from 'node:assert/strict';
import test from 'node:test';
import { gatewayCommandKind, gatewayCommandRaw, kindsById } from '../../frontend/src/features/tools/gateway/commandKinds.ts';
import { kindCounts } from '../../frontend/src/features/tools/gateway/kindCounts.ts';
import { COMMAND_FILTER, gatewayCommandMatches, masterCommandMatches }
  from '../../frontend/src/features/tools/gateway/logFilters.ts';
import { parseAddress, parseScale, typeChoices } from '../../frontend/src/features/tools/gateway/pointEdits.ts';
import { GATEWAY_POINT_FILTER, gatewayPointMatches, MASTER_POINT_FILTER, masterPointMatches, SERVED_POINT_FILTER,
  servedPointMatches } from '../../frontend/src/features/tools/gateway/pointFilters.ts';
import { trafficKind } from '../../frontend/src/features/tools/gateway/trafficKind.ts';
import { kindOfGatewayPoint, kindOfMasterGroup } from '../../frontend/src/points/pointKind.ts';
import { initialValues, restore } from '../../frontend/src/ui/filterState.ts';
import { ALL_KINDS } from '../../frontend/src/ui/kindFilter.ts';

const gateway = (kind: string, extra = {}) => ({ id: kind, kind: kind as never, address: 1, ied: 'i1', ref: `LD/${kind}`, scale: 1,
  description: '', ...extra });
const master = (group: string, extra = {}) => ({ key: `${group}:1`, group, group_name: group, type: 'M_XX', address: 1, value: 0,
  value_text: '0', quality_text: 'good', good: true, time_ms: null, cause: '', changes: 0, label: '', ticked: true, stale: false,
  command: null, ...extra });
const logged = (extra = {}) => ({ seq: 1, at: 0, label: 'Close', protocol: 'manual', peer: '', request: '', applied: 'operated',
  observed: '', ...extra });
const withKind = (spec: typeof COMMAND_FILTER, kind: string) => ({ ...initialValues(spec), kind });

test('the point lists offer the shared type chips, not the old Status / Measured / Commands', () => {
  for (const spec of [GATEWAY_POINT_FILTER, MASTER_POINT_FILTER, SERVED_POINT_FILTER, COMMAND_FILTER]) {
    const group = spec.groups?.find(item => item.key === 'kind');
    assert.ok(group?.kinds, 'a kind chip group');
    assert.equal(group.initial, ALL_KINDS);
    assert.ok(group.options.some(option => option.value === 'DP') && group.options.some(option => option.value === 'SP'),
      'single and double points are separate chips');
  }
  // A choice remembered from the old chips is ignored instead of hiding every row.
  assert.equal(restore(GATEWAY_POINT_FILTER, { kind: 'status' }).kind, ALL_KINDS);
  assert.equal(restore(MASTER_POINT_FILTER, { kind: 'commands' }).kind, ALL_KINDS);
  assert.equal(restore(GATEWAY_POINT_FILTER, { kind: 'DP' }).kind, 'DP');
});

test('type chips come with the number of rows of each kind; kinds without rows are left out', () => {
  const rows = [gateway('single'), gateway('single'), gateway('double'), gateway('command_double')];
  const counts = kindCounts(rows, row => kindOfGatewayPoint(row.kind)).kind;
  assert.deepEqual(counts, { all: 4, SP: 2, DP: 1, DC: 1 });
  assert.deepEqual(kindCounts([], kindOfGatewayPoint as never), { kind: { all: 0 } });
  assert.deepEqual(kindCounts([master('scaled'), master('float')], row => kindOfMasterGroup(row.group), 'type'),
    { type: { all: 2, MV: 2 } });
});

test('gateway points: single and double points are told apart, commands by their own type', () => {
  const all = initialValues(GATEWAY_POINT_FILTER);
  assert.ok(gatewayPointMatches(gateway('single'), { ...all, kind: 'SP' }));
  assert.ok(!gatewayPointMatches(gateway('double'), { ...all, kind: 'SP' }));
  assert.ok(gatewayPointMatches(gateway('command_single'), { ...all, kind: 'SC' }));
  assert.ok(!gatewayPointMatches(gateway('command_single'), { ...all, kind: 'DC' }));
  assert.ok(gatewayPointMatches(gateway('float'), { ...all, kind: 'MV' }));
  assert.ok(gatewayPointMatches(gateway('double'), { ...all, text: 'dp' }), 'the kind letters are searchable');
});

test('master points: type chip, point group and the "has a command" toggle work together', () => {
  const all = initialValues(MASTER_POINT_FILTER);
  const scaled = master('scaled', { group_name: 'Measured (scaled)', command: 'setpoint_scaled' });
  assert.ok(masterPointMatches(scaled, { ...all, kind: 'MV' }));
  assert.ok(masterPointMatches(master('float'), { ...all, kind: 'MV' }), 'every measured group is one kind');
  assert.ok(masterPointMatches(scaled, { ...all, kind: 'MV', group: 'Measured (scaled)' }));
  assert.ok(!masterPointMatches(master('float', { group_name: 'Measured (float)' }), { ...all, kind: 'MV', group: 'Measured (scaled)' }));
  assert.ok(masterPointMatches(scaled, { ...all, commandable: true }));
  assert.ok(!masterPointMatches(master('single'), { ...all, commandable: true }));
  assert.ok(masterPointMatches(master('counter'), { ...all, kind: 'CT' }));
  assert.ok(!masterPointMatches(master('double_binary'), { ...all, kind: 'SP' }));
});

test('served points can be filtered by type and searched, read only', () => {
  const all = initialValues(SERVED_POINT_FILTER);
  const row = gateway('double', { description: 'Breaker Q0', address: 4012 });
  assert.ok(servedPointMatches(row, { ...all, kind: 'DP' }));
  assert.ok(!servedPointMatches(row, { ...all, kind: 'MV' }));
  assert.ok(servedPointMatches(row, { ...all, text: 'breaker 4012' }));
  assert.ok(!servedPointMatches(row, { ...all, text: 'feeder' }));
});

test('a gateway command takes the type of the point it operated, else it stays unknown', () => {
  const kinds = kindsById([gateway('command_double', { id: 'p1' }), gateway('command_single', { id: 'p2' }), { kind: 'float' }]);
  assert.equal(kinds.size, 2, 'a point without an id cannot be looked up');
  assert.equal(gatewayCommandKind(logged({ point_id: 'p1' }), kinds), 'DC');
  assert.equal(gatewayCommandKind(logged({ point_id: 'p2' }), kinds), 'SC');
  assert.equal(gatewayCommandKind(logged({ point_id: 'gone' }), kinds), 'UN');
  assert.equal(gatewayCommandKind(logged(), kinds), 'UN');
  assert.equal(gatewayCommandRaw(logged({ point_id: 'p1' }), kinds), 'Double command');
  assert.equal(gatewayCommandRaw(logged({ point_id: 'gone' }), kinds), undefined);
  assert.ok(gatewayCommandMatches(logged(), withKind(COMMAND_FILTER, 'DC'), 'DC'));
  assert.ok(!gatewayCommandMatches(logged(), withKind(COMMAND_FILTER, 'DC'), 'SC'));
  assert.ok(gatewayCommandMatches(logged(), initialValues(COMMAND_FILTER)), 'no chip chosen: unknown types still show');
});

test('master commands are told apart by their command type', () => {
  const entry = (kind: string) => ({ seq: 1, at: 0, who: 'You · this page', kind: kind as never, address: 5, value: 1, select: false,
    state: 'confirmed' as const, text: '' });
  assert.ok(masterCommandMatches(entry('double'), withKind(COMMAND_FILTER, 'DC')));
  assert.ok(!masterCommandMatches(entry('single'), withKind(COMMAND_FILTER, 'DC')));
  assert.ok(masterCommandMatches(entry('setpoint_float'), withKind(COMMAND_FILTER, 'SE')));
  assert.ok(masterCommandMatches(entry('binary_output'), withKind(COMMAND_FILTER, 'SC')), 'a CROB is an on/off command');
  assert.ok(masterCommandMatches(entry('clock'), withKind(COMMAND_FILTER, 'UN')));
});

test('a traffic line gets a type only when it starts with a standard 104 type id', () => {
  assert.deepEqual(trafficKind('M_DP_TB_1 spont IOA 5 = ON'), { kind: 'DP', token: 'M_DP_TB_1' });
  assert.deepEqual(trafficKind('M_ME_NC_1 spontaneous IOA 100 = 49.98'), { kind: 'MV', token: 'M_ME_NC_1' });
  assert.deepEqual(trafficKind('C_SC_NA_1 select IOA 7'), { kind: 'SC', token: 'C_SC_NA_1' });
  assert.deepEqual(trafficKind('C_SE_NC_1'), { kind: 'SE', token: 'C_SE_NC_1' });
  assert.equal(trafficKind('C_IC_NA_1 station interrogation: answered with 12 points'), null, 'a control frame is no point type');
  assert.equal(trafficKind('STARTDT act (start data transfer)'), null);
  assert.equal(trafficKind('Master connected'), null);
  assert.equal(trafficKind('CROB operate index 3 CLOSE/ON'), null, 'DNP3 lines carry no type id');
  assert.equal(trafficKind('warning: M_DP_TB_1 is mentioned later'), null, 'only a leading id counts');
  assert.equal(trafficKind('M_DP_TB_1x'), null);
});

test('typed address and scale are turned into numbers as before', () => {
  assert.equal(parseAddress('1 042'), 1042);
  assert.equal(parseAddress('ioa 7'), 7);
  assert.equal(parseAddress(''), 0);
  assert.equal(parseScale('0.5'), 0.5);
  assert.equal(parseScale(' 0,25 '), 0.25, 'a comma is a decimal point');
  assert.equal(parseScale('-1'), -1);
  assert.equal(parseScale('abc'), 0, 'not a number becomes 0, which Save refuses');
  const choices = typeChoices('iec104');
  assert.deepEqual(choices.map(choice => choice.value), ['single', 'double', 'float', 'integer', 'command_single', 'command_double',
    'step_position', 'command_step', 'command_setpoint']); // ADR-0107 kinds follow the six
  assert.equal(choices[1].label, 'Double point · M_DP');
  assert.match(typeChoices('modbus')[0].label, /Discrete input/);
});
