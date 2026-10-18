// Change 0093: the shared filter model and every list's filter rules (display only).
import assert from 'node:assert/strict';
import test from 'node:test';
import { choicesOf, countText, effective, initialValues, isFiltered, rememberable, restore, textMatch,
  type FilterSpec } from '../../frontend/src/ui/filterState.ts';
import { GATEWAY_POINT_FILTER, gatewayKind, gatewayPointMatches, MASTER_POINT_FILTER, masterPointMatches }
  from '../../frontend/src/features/tools/gateway/pointFilters.ts';
import { COMMAND_FILTER, gatewayCommandMatches, gatewayResult, isKeepAlive, masterCommandMatches, TRAFFIC_FILTER,
  trafficMatches } from '../../frontend/src/features/tools/gateway/logFilters.ts';
import { FIND_FILTER, foundMatches, knownAddresses } from '../../frontend/src/features/tools/find/findFilters.ts';
import { nodeKind, nodeMatches, trackChanges, TREE_FILTER, WATCH_FILTER, watchMatches }
  from '../../frontend/src/features/tools/explorer/explorerFilters.ts';
import { compareMatches, COMPARE_FILTER, DEVICE_FILTER, deviceMatches, roleCounts, SIGNAL_FILTER, signalQuery }
  from '../../frontend/src/features/devices/deviceFilters.ts';

const SPEC: FilterSpec = {
  search: 'Search',
  groups: [{ key: 'kind', label: 'Kind', initial: 'all', options: [{ value: 'all', label: 'All' }, { value: 'x', label: 'X' }] }],
  toggles: [{ key: 'bad', label: 'Bad' }, { key: 'pause', label: 'Pause', remember: false }],
  selects: [{ key: 'ied', label: 'IED', all: 'All IEDs' }],
};
const values = (spec: FilterSpec, changes: Record<string, string | boolean> = {}) => ({ ...initialValues(spec), ...changes });

test('the filter bar starts empty and offers Clear only after a change', () => {
  const start = initialValues(SPEC);
  assert.deepEqual(start, { text: '', kind: 'all', bad: false, pause: false, ied: '' });
  assert.equal(isFiltered(SPEC, start), false);
  assert.equal(isFiltered(SPEC, { ...start, text: '   ' }), false, 'spaces are not a search');
  assert.equal(isFiltered(SPEC, { ...start, kind: 'x' }), true);
  assert.equal(countText(12, 340, true), '12 of 340');
  assert.equal(countText(340, 340, false), '340');
});

test('remembered filters keep chips and dropdowns, never the search text or Pause', () => {
  const chosen = { ...initialValues(SPEC), text: 'secret', kind: 'x', bad: true, pause: true, ied: 'FEED1' };
  const stored = rememberable(SPEC, chosen);
  assert.deepEqual(stored, { kind: 'x', bad: true, ied: 'FEED1' });
  assert.deepEqual(restore(SPEC, stored), { text: '', kind: 'x', bad: true, pause: false, ied: 'FEED1' });
  // Damaged or old choices fall back to the start values instead of hiding rows.
  assert.deepEqual(restore(SPEC, { kind: 'gone', bad: 'yes', other: true, pause: true }), initialValues(SPEC));
  assert.deepEqual(restore(SPEC, null), initialValues(SPEC));
});

test('a remembered dropdown choice that is not in today\'s list counts as all', () => {
  const chosen = values(SPEC, { ied: 'OLD' });
  assert.equal(effective(SPEC, chosen, { ied: [{ value: 'NEW', label: 'NEW' }] }).ied, '');
  assert.equal(effective(SPEC, values(SPEC, { ied: 'NEW' }), { ied: [{ value: 'NEW', label: 'NEW' }] }).ied, 'NEW');
  assert.deepEqual(choicesOf(['b', 'a', '', null, 'a', 'B10', 'B9']).map(choice => choice.value), ['a', 'b', 'B9', 'B10']);
});

test('search matches every word in any field, ignoring case', () => {
  assert.ok(textMatch('', 'anything'));
  assert.ok(textMatch('feed1 pos', 'FEED1LD1/XCBR1.Pos', 'breaker'));
  assert.ok(!textMatch('feed2', 'FEED1LD1/XCBR1.Pos'));
  assert.ok(textMatch('30001', 30001));
});

test('SCADA points filter by type, IED, bad quality and not served', () => {
  const row = { id: 'a', kind: 'double' as const, address: 10, ied: 'i1', ied_name: 'FEED1', ref: 'FEED1LD1/XCBR1.Pos.stVal',
    scale: 1, description: 'Breaker', online: true };
  const all = values(GATEWAY_POINT_FILTER);
  assert.equal(gatewayKind('command_single'), 'commands');
  assert.equal(gatewayKind('float'), 'measured');
  assert.ok(gatewayPointMatches(row, { ...all, kind: 'status', ied: 'FEED1', text: 'breaker' }));
  assert.ok(!gatewayPointMatches(row, { ...all, kind: 'measured' }));
  assert.ok(!gatewayPointMatches(row, { ...all, ied: 'FEED2' }));
  assert.ok(!gatewayPointMatches(row, { ...all, bad: true }));
  assert.ok(gatewayPointMatches({ ...row, online: false }, { ...all, bad: true }));
  assert.ok(!gatewayPointMatches(row, { ...all, unserved: true }));
  assert.ok(gatewayPointMatches({ ...row, id: undefined }, { ...all, unserved: true }), 'a row without id is not saved yet');
});

test('master points filter by group, kind, quality and watched', () => {
  const row = { key: 'single:1', group: 'single', group_name: 'Single point', type: 'M_SP', address: 1, value: 1,
    value_text: 'ON', quality_text: 'good', good: true, time_ms: null, cause: '', changes: 0, label: 'Alarm',
    ticked: true, stale: false, command: 'single' as const };
  const all = values(MASTER_POINT_FILTER);
  assert.ok(masterPointMatches(row, { ...all, kind: 'status' }));
  assert.ok(masterPointMatches(row, { ...all, kind: 'commands' }), 'a commandable status point is in both');
  assert.ok(!masterPointMatches(row, { ...all, kind: 'measured' }));
  assert.ok(masterPointMatches({ ...row, group: 'float', command: null }, { ...all, kind: 'measured' }));
  assert.ok(!masterPointMatches(row, { ...all, bad: true }));
  assert.ok(!masterPointMatches({ ...row, ticked: false }, { ...all, watched: true }));
  assert.ok(!masterPointMatches(row, { ...all, group: 'Double point' }));
});

test('traffic filters direction, errors and keep-alives', () => {
  const line = (dir: string, text: string) => ({ seq: 1, at: 0, dir, peer: '10.0.0.5', text });
  const all = values(TRAFFIC_FILTER);
  assert.ok(trafficMatches(line('tx', 'C_IC_NA_1 station interrogation'), { ...all, dir: 'tx' }));
  assert.ok(!trafficMatches(line('rx', 'C_IC_NA_1'), { ...all, dir: 'tx' }));
  assert.ok(trafficMatches(line('rx', 'C_RD_NA_1 IOA 5: unknown IOA (COT 47)'), { ...all, errors: true }));
  assert.ok(!trafficMatches(line('rx', 'STARTDT con: data transfer started'), { ...all, errors: true }));
  assert.ok(isKeepAlive('TESTFR act') && isKeepAlive('DNP3 link status request'));
  assert.ok(!trafficMatches(line('rx', 'TESTFR act'), { ...all, quiet: true }));
  assert.ok(TRAFFIC_FILTER.toggles?.every(toggle => toggle.remember !== false), 'Pause is a button, not a filter');
});

test('commands filter by result and by who sent them', () => {
  const entry = { seq: 1, at: 0, label: 'Close FEED1 CB', protocol: 'manual', peer: '', request: '', applied: 'operated', observed: '' };
  const all = values(COMMAND_FILTER);
  assert.equal(gatewayResult(entry), 'done');
  assert.equal(gatewayResult({ ...entry, applied: 'refused: blocked' }), 'refused');
  assert.equal(gatewayResult({ ...entry, applied: '' }), 'waiting');
  assert.ok(gatewayCommandMatches(entry, { ...all, result: 'done', from: 'me' }));
  assert.ok(!gatewayCommandMatches(entry, { ...all, from: 'scada' }));
  assert.ok(gatewayCommandMatches({ ...entry, protocol: 'iec104' }, { ...all, from: 'scada' }));
  const sent = { seq: 2, at: 0, who: 'You · this page', kind: 'double' as const, address: 7, value: 2, select: true,
    state: 'refused' as const, text: 'negative confirmation' };
  assert.ok(masterCommandMatches(sent, { ...all, result: 'refused', from: 'me' }));
  assert.ok(masterCommandMatches({ ...sent, who: 'SCADA via the converter' }, { ...all, from: 'scada' }));
  assert.ok(!masterCommandMatches(sent, { ...all, result: 'done' }));
});

test('Find IEDs results filter by answer, IEC 61850, project and vendor', () => {
  const device = { ip: '192.168.1.20', port: 102, tcp_ms: 2, identified: true, vendor: 'OpenSCD', model: 'Sim', revision: '1',
    devices: ['FEED1LD1'], reason: '' };
  const all = values(FIND_FILTER);
  const known = new Set(knownAddresses({ ieds: [{ ip: '127.50.0.2', live: { ip: '192.168.1.20' }, lab_ip: '',
    scl_address: null } as never] }));
  assert.ok(known.has('192.168.1.20') && known.has('127.50.0.2') && !known.has(''));
  assert.ok(!foundMatches(device, { ...all, unknown: true }, known));
  assert.ok(foundMatches(device, { ...all, answered: true, iec: true, vendor: 'OpenSCD', text: 'feed1' }, new Set()));
  assert.ok(!foundMatches({ ...device, identified: null }, { ...all, answered: true }, known));
  assert.ok(!foundMatches({ ...device, devices: [] }, { ...all, iec: true }, known));
});

test('Explorer tree groups logical nodes by class; the watch list tracks changes', () => {
  assert.equal(nodeKind('XCBR1'), 'controls');
  assert.equal(nodeKind('Q0CSWI1'), 'controls');
  assert.equal(nodeKind('MMXU1'), 'measured');
  assert.equal(nodeKind('PTOC2'), 'settings');
  assert.equal(nodeKind('LLN0'), 'other');
  assert.ok(nodeMatches('MMXU1', { ...values(TREE_FILTER), kind: 'measured', text: 'mmxu' }));
  assert.ok(!nodeMatches('LLN0', { ...values(TREE_FILTER), kind: 'controls' }));
  const item = { ref: 'LD/XCBR1.Pos.stVal', fc: 'ST', type: 'bit', text: 'closed', status: 'ok' as const, at: 1 };
  const first = trackChanges(new Map(), [item], 1000);
  assert.equal(first.get('LD/XCBR1.Pos.stVal|ST')?.at, 0, 'the first reading is not a change');
  const changed = trackChanges(first, [{ ...item, text: 'open' }], 5000);
  assert.equal(changed.get('LD/XCBR1.Pos.stVal|ST')?.at, 5000);
  const watch = values(WATCH_FILTER);
  assert.ok(watchMatches(item, { ...watch, recent: true }, 5000, 30_000));
  assert.ok(!watchMatches(item, { ...watch, recent: true }, 5000, 70_000));
  assert.ok(!watchMatches(item, { ...watch, bad: true }, 0, 0));
});

test('Devices list, data points and comparison filters', () => {
  const ied = { id: '1', name: 'FEED1', role: 'simulated', state: 'stopped', warnings: [], goose: { diagnostics: [] },
    scl_address: { ip: '172.16.1.20' }, live: { ip: '' } } as never;
  const all = values(DEVICE_FILTER);
  assert.ok(deviceMatches(ied, { ...all, role: 'simulated', text: '172.16' }));
  assert.ok(!deviceMatches(ied, { ...all, role: 'real' }));
  assert.ok(!deviceMatches(ied, { ...all, problems: true }));
  assert.ok(deviceMatches({ ...(ied as object), warnings: ['x'] } as never, { ...all, problems: true }));
  assert.deepEqual(roleCounts([ied, { ...(ied as object), role: 'real' } as never]), { all: 2, real: 1, simulated: 1, off: 0 });
  const query = new URLSearchParams(signalQuery({ ...values(SIGNAL_FILTER), text: ' pos ', kind: 'measured', ld: 'FEED1LD1',
    bad: true, dataset: true, report: 'R1' }, 200));
  assert.deepEqual(Object.fromEntries(query), { q: 'pos', offset: '200', fc: 'MX', ld: 'FEED1LD1', report: 'R1',
    bad_quality: 'true', in_dataset: 'true' });
  assert.deepEqual(Object.fromEntries(new URLSearchParams(signalQuery(values(SIGNAL_FILTER), 0))), { q: '', offset: '0' });
  const compare = values(COMPARE_FILTER);
  assert.equal(compare.show, 'problems', 'Differences only is the default');
  assert.ok(!compareMatches({ ref: 'a', fc: 'ST', category: 'match' }, compare));
  assert.ok(compareMatches({ ref: 'a', fc: 'ST', category: 'match' }, { ...compare, show: 'all' }));
  assert.ok(compareMatches({ ref: 'a', fc: 'ST', category: 'type_mismatch' }, { ...compare, show: 'type_mismatch' }));
});
