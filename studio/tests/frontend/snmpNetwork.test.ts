// ADR-0092 amendment: SNMP devices follow the IEDs onto the network card chosen in Network setup.
import assert from 'node:assert/strict';
import test from 'node:test';
import type { PlanPreview } from '../../frontend/src/api/network.ts';
import { changeSentence, putLabel, setupTitle, snmpFollowLine } from '../../frontend/src/features/network/setupWords.ts';

const card = { guid: 'G', name: 'Ethernet 2', description: '', status: 'Up', index: 1, speed: '', kind: 'ethernet',
  default_route: false, addresses: [] };
const row = (id: string, kind?: 'snmp') => ({ ied_id: id, name: id, scd_ip: '', proposed: '192.168.10.9', prefix: 24,
  status: 'ok', hint: '', kind });
const preview = (items: PlanPreview['items'], rows = [row('ied1')]): PlanPreview =>
  ({ adapter: card, rows, items, fingerprint: 'f', ready: true, needs_admin: true });
const item = (kind: PlanPreview['items'][number]['kind'], ports?: number[]) =>
  ({ kind, action: 'add' as const, owned: true, description: '', ports });

test('the sentence names the IED port and the SNMP UDP port separately', () => {
  const both = changeSentence(preview([item('alias'), item('alias'), item('firewall', [102]), item('snmp_firewall', [161])]));
  assert.match(both, /add 2 addresses to “Ethernet 2” and allow port 102 for the IEDs and UDP port 161 for the SNMP devices in Windows Firewall/);
  const snmpOnly = changeSentence(preview([item('alias'), item('snmp_firewall', [161, 1161])]));
  assert.match(snmpOnly, /allow UDP ports 161, 1161 for the SNMP devices in Windows Firewall/);
  assert.match(changeSentence(preview([item('alias'), item('firewall', [102])])), /allow port 102 for them in Windows Firewall/);
});

test('the button and title count IEDs and SNMP devices in words', () => {
  assert.equal(putLabel(4, 'Ethernet 2', 2), 'Put 4 IEDs and 2 SNMP devices on “Ethernet 2”');
  assert.equal(putLabel(0, 'Ethernet 2', 1), 'Put 1 SNMP device on “Ethernet 2”');
  assert.equal(putLabel(1, undefined), 'Put 1 IED on this card');
  assert.equal(putLabel(0), 'Put IEDs on this card');
  assert.equal(setupTitle(preview([], [row('ied1'), row('snmp:snmp1', 'snmp')])),
    'Put your published IEDs and SNMP devices on a network');
  assert.equal(setupTitle(preview([])), 'Put your published IEDs on a network');
});

test('the settings line says SNMP devices follow the IEDs, in correct English for one or many', () => {
  assert.equal(snmpFollowLine(1, false), 'Your 1 SNMP device follows the IEDs and stays on this PC too.');
  assert.equal(snmpFollowLine(3, false), 'Your 3 SNMP devices follow the IEDs and stay on this PC too.');
  assert.equal(snmpFollowLine(1, true), 'Your 1 SNMP device follows the IEDs: it answers at its own address on this card.');
  assert.equal(snmpFollowLine(2, true), 'Your 2 SNMP devices follow the IEDs: each answers at its own address on this card.');
});
