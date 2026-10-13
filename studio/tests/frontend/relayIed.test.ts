import assert from 'node:assert/strict';
import test from 'node:test';
import { isMixedRelay, relayIedIds, resolveRelayIedId } from '../../frontend/src/equipment/relayIed.ts';

const first = { id: 'a', label: 'Trip', source: { ied_id: 'ied-1', ref: 'GGIO1.Ind1.stVal' } };
const second = { id: 'b', label: 'Alarm', source: { ied_id: 'ied-1', ref: 'GGIO1.Ind2.stVal' } };
const foreign = { id: 'c', label: 'Other', source: { ied_id: 'ied-2', ref: 'GGIO1.Ind1.stVal' } };

test('relay IED ids are unique in first-seen order', () => {
  assert.deepEqual(relayIedIds([first, second, foreign]), ['ied-1', 'ied-2']);
  assert.deepEqual(relayIedIds([]), []);
});

test('empty relay has no IED; otherwise the first entry wins', () => {
  assert.equal(resolveRelayIedId([]), null);
  assert.equal(resolveRelayIedId([first, second]), 'ied-1');
});

test('mixed relays are detected for reassignment', () => {
  assert.equal(isMixedRelay([first, second]), false);
  assert.equal(isMixedRelay([first, foreign]), true);
});
