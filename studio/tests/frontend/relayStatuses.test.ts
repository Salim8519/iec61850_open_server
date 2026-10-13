import assert from 'node:assert/strict';
import test from 'node:test';
import { relayBindings, relayValueKey, stateBindings } from '../../frontend/src/equipment/stateBindings.ts';
import type { EquipmentData } from '../../frontend/src/api/types.ts';

const data: EquipmentData = { kind: 'RELAY', label: 'Relay 1', rotation: 0,
  relay_statuses: [
    { id: 's1', label: 'Trip', source: { ied_id: 'ied-a', ref: 'GGIO1.Ind1.stVal' } },
    { id: 's2', label: 'Alarm', source: { ied_id: 'ied-b', ref: 'GGIO2.Ind2.stVal' } },
  ] };

test('relay bindings collect every named status entry', () => {
  assert.deepEqual(relayBindings(data), [
    { ied_id: 'ied-a', ref: 'GGIO1.Ind1.stVal' },
    { ied_id: 'ied-b', ref: 'GGIO2.Ind2.stVal' }]);
  assert.deepEqual(relayBindings({ kind: 'RELAY', label: 'x', rotation: 0 }), []);
});

test('state mappings and relay entries are separate sources', () => {
  assert.deepEqual(stateBindings(data.state_mapping), []);
  const switchData = { kind: 'CB' as const, label: 'x', rotation: 0,
    state_mapping: { mode: 'direct' as const, source: { ied_id: 'a', ref: 'r' } } };
  assert.deepEqual(stateBindings(switchData.state_mapping), [{ ied_id: 'a', ref: 'r' }]);
});

test('relay live values key on node and entry identifiers', () => {
  assert.equal(relayValueKey('r1', 's2'), 'r1:s2');
});
