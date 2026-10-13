import assert from 'node:assert/strict';
import test from 'node:test';
import { defaultRelayLabel, pulseTarget, relayRowLabel, relayValue, shortRef }
  from '../../frontend/src/equipment/relayDisplay.ts';
import type { RelayStatus, Signal } from '../../frontend/src/api/types.ts';

const entry = (label: string, ref: string): RelayStatus =>
  ({ id: 'e1', label, source: { ied_id: 'i1', ref } });
const signal = (ref: string, description = ''): Signal => ({ ref, description, fc: 'ST',
  kind: 'BOOLEAN', value: false, writable: true, quality: 0, timestamp: 0, enums: {} });

test('short refs drop only the IED name and keep the logical-device instance', () => {
  assert.equal(shortRef('IED1Master/LLN0.Mod.stVal', 'IED1'), 'Master/LLN0.Mod.stVal');
  assert.equal(shortRef('IED1Prot/LLN0.Mod.stVal', 'IED1'), 'Prot/LLN0.Mod.stVal');
});

test('short refs leave anything they cannot shorten unambiguously unchanged', () => {
  assert.equal(shortRef('IED1Master/LLN0.Mod.stVal'), 'IED1Master/LLN0.Mod.stVal');
  assert.equal(shortRef('OTHERMaster/LLN0.Mod.stVal', 'IED1'), 'OTHERMaster/LLN0.Mod.stVal');
  assert.equal(shortRef('IED1/LLN0.Mod.stVal', 'IED1'), 'IED1/LLN0.Mod.stVal');
});

test('row labels shorten untouched default names but keep user names', () => {
  assert.equal(relayRowLabel(entry('IED1Master/GGIO1.Ind1.stVal', 'IED1Master/GGIO1.Ind1.stVal'), 'IED1'),
    'Master/GGIO1.Ind1.stVal');
  assert.equal(relayRowLabel(entry('Trip alarm', 'IED1Master/GGIO1.Ind1.stVal'), 'IED1'), 'Trip alarm');
});

test('new signals are named from the description, else the short ref', () => {
  assert.equal(defaultRelayLabel(signal('IED1Master/GGIO1.Ind1.stVal', ' Breaker fail '), 'IED1'), 'Breaker fail');
  assert.equal(defaultRelayLabel(signal('IED1Master/GGIO1.Ind1.stVal'), 'IED1'), 'Master/GGIO1.Ind1.stVal');
});

test('values always carry explicit text, not only a state class', () => {
  assert.deepEqual(relayValue(true), { text: '1', state: 'on' });
  assert.deepEqual(relayValue(false), { text: '0', state: 'off' });
  assert.deepEqual(relayValue(undefined), { text: '—', state: 'none' });
  assert.deepEqual(relayValue(null), { text: '—', state: 'none' });
  assert.deepEqual(relayValue(7), { text: '7', state: 'value' });
});

test('a pulse drives the opposite of the current value', () => {
  assert.equal(pulseTarget(true), false);
  assert.equal(pulseTarget(false), true);
  assert.equal(pulseTarget(undefined), true);
});
