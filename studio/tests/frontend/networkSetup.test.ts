// Change 0098: the one-page network setup describes cards and Windows changes in plain words.
import assert from 'node:assert/strict';
import test from 'node:test';
import { adapterLabel, changeSentence, usableAdapters } from '../../frontend/src/features/network/setupWords.ts';

const card = (extra = {}) => ({ guid: 'G', name: 'Ethernet 2', description: 'USB', status: 'Up', index: 1, speed: '1 Gbps',
  kind: 'ethernet', default_route: false, addresses: [{ ip: '192.168.230.5', prefix: 24, skip_as_source: false, origin: 'Manual' }],
  ...extra });
const preview = (items: { kind: 'alias' | 'firewall'; action: 'add' | 'keep' }[]) => ({ adapter: card(), rows: [],
  items: items.map(item => ({ ...item, owned: true, description: '' })), fingerprint: 'f', ready: true, needs_admin: true });

test('a card label names the card, its kind, its addresses and a missing cable', () => {
  assert.equal(adapterLabel(card()), 'Ethernet 2 · Ethernet · 192.168.230.5/24');
  assert.equal(adapterLabel(card({ status: 'Disconnected', default_route: true, addresses: [] })),
    'Ethernet 2 · Ethernet · no IPv4 address · no cable · internet');
});

test('VPN and Bluetooth cards are never offered', () => {
  const names = usableAdapters([card(), card({ guid: 'V', kind: 'vpn' }), card({ guid: 'B', kind: 'bluetooth' })])
    .map(item => item.guid);
  assert.deepEqual(names, ['G']);
});

test('one sentence counts the addresses and mentions the firewall, the single approval and undo', () => {
  const sentence = changeSentence(preview([{ kind: 'alias', action: 'add' }, { kind: 'alias', action: 'add' },
    { kind: 'firewall', action: 'add' }]));
  assert.match(sentence, /add 2 addresses to “Ethernet 2” and allow port 102/);
  assert.match(sentence, /Windows asks once/);
  assert.match(sentence, /Clean up/);
  assert.match(changeSentence(preview([{ kind: 'alias', action: 'keep' }])), /Nothing changes in Windows/);
});
