// A typed address in Network setup gets its first answer at once (duplicates and the PC's other cards), and spaces
// around it are dropped. The engine's answer still replaces it and alone decides if apply works.
import assert from 'node:assert/strict';
import test from 'node:test';
import { cleanManual, provisionalChecks } from '../../frontend/src/features/network/typedAddressCheck.ts';

const row = (id: string, name: string, proposed: string, extra = {}) =>
  ({ ied_id: id, name, scd_ip: proposed, proposed, prefix: 24, status: 'ok', hint: '', ...extra });
const card = (guid: string, name: string, ip: string) => ({ guid, name, description: '', status: 'Up', index: 1, speed: '',
  kind: 'ethernet', default_route: false, addresses: [{ ip, prefix: 24, skip_as_source: false, origin: 'Manual' }] });
const rows = [row('a', 'FEED1', '172.16.1.21'), row('b', 'FEED2', '172.16.1.22'), row('c', 'FEED3', '172.16.1.23')];
const cards = [card('LAB', 'Lab NIC', '172.16.1.5'), card('WIFI', 'Wi-Fi', '192.168.0.20')];
const typed = (ip: string) => ({ ip, prefix: 24 });

test('spaces around a typed address are dropped before it is sent', () => {
  assert.deepEqual(cleanManual({ a: typed(' 172.16.1.20 '), b: typed('\t172.16.1.30') }),
    { a: typed('172.16.1.20'), b: typed('172.16.1.30') });
});

test('an address another IED already has is a duplicate at once, even with a trailing space', () => {
  const found = provisionalChecks(rows, { c: typed('172.16.1.21 ') }, {}, cards, 'LAB');
  assert.deepEqual(found.c, { status: 'duplicate', hint: 'FEED1 already uses 172.16.1.21. Each IED needs its own address.' });
});

test('two typed rows with one address: the one higher in the list keeps it', () => {
  const found = provisionalChecks(rows, { b: typed('172.16.1.50'), c: typed('172.16.1.50') }, {}, cards, 'LAB');
  assert.equal(found.b.status, 'checking');
  assert.equal(found.c.status, 'duplicate');
  assert.match(found.c.hint, /^FEED2 already uses 172.16.1.50/);
});

test('a row nobody typed for keeps its address before a typed row above it', () => {
  const found = provisionalChecks(rows, { a: typed('172.16.1.22') }, {}, cards, 'LAB');
  assert.equal(found.a.status, 'duplicate');
  assert.match(found.a.hint, /^FEED2 already uses 172.16.1.22/);
});

test("this PC's other card is named, the chosen card is not", () => {
  const other = provisionalChecks(rows, { a: typed('192.168.0.20') }, {}, cards, 'LAB');
  assert.deepEqual(other.a, { status: 'used_by_other_adapter',
    hint: "192.168.0.20 already belongs to 'Wi-Fi'. Pick another address or disconnect that adapter." });
  assert.equal(provisionalChecks(rows, { a: typed('172.16.1.5') }, {}, cards, 'LAB').a.status, 'checking');
});

test('a free address is only "checking" until the engine answers, and an answered row says nothing', () => {
  assert.equal(provisionalChecks(rows, { a: typed('172.16.1.77') }, {}, cards, 'LAB').a.status, 'checking');
  assert.equal(provisionalChecks(rows, { a: typed('') }, {}, cards, 'LAB').a.status, 'checking');
  assert.deepEqual(provisionalChecks(rows, { a: typed('172.16.1.77') }, { a: typed('172.16.1.77 ') }, cards, 'LAB'), {});
});

test('an SNMP device row says "device"', () => {
  const withSnmp = [...rows, row('snmp:1', 'SNMP · Switch', '172.16.1.60', { kind: 'snmp' })];
  const found = provisionalChecks(withSnmp, { 'snmp:1': typed('172.16.1.21') }, {}, cards, 'LAB');
  assert.match(found['snmp:1'].hint, /Each device needs its own address\.$/);
});
