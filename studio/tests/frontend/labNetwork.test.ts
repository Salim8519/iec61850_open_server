import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { blocking, type LabRow } from '../../frontend/src/api/lab.ts';
import { LAB_COPY, LAB_ROW_STATUS, pingWords, realCardLine } from '../../frontend/src/copy/labNetwork.ts';
import { TONES } from '../../frontend/src/copy/status.ts';

const read = (path: string) => readFileSync(new URL(`../../${path}`, import.meta.url), 'utf8');

test('every lab row status the Go engine decides has words and an icon (ADR-0120)', () => {
  const rows = read('engine/internal/labsetup/rows.go');
  const statuses = new Set([...rows.matchAll(/(?:row\.Status(?:, row\.Hint)? = |case ")"([a-z_]+)"/g)].map(match => match[1]));
  for (const status of ['ok', 'same_range', 'real_device', 'checking', 'duplicate', 'other_card', 'invalid', 'missing']) {
    assert.ok(statuses.has(status), `Go decides ${status}`);
  }
  for (const status of statuses) {
    assert.ok(LAB_ROW_STATUS[status], `${status} needs words`);
    assert.ok(TONES[LAB_ROW_STATUS[status].tone].icon);
  }
});

test('the page blocks exactly the rows the engine refuses', () => {
  const go = read('engine/internal/labsetup/rows.go');
  const refused = go.slice(go.indexOf('func (r Row) Blocking'), go.indexOf('// Real is'));
  for (const status of Object.keys(LAB_ROW_STATUS)) {
    const row = { status } as LabRow;
    assert.equal(blocking(row), refused.includes(`"${status}"`), status);
  }
  assert.ok(LAB_ROW_STATUS.real_device.tone === 'warning', 'a real device warns; it never blocks');
});

test('ping and real card lines read like Windows', () => {
  assert.equal(pingWords('192.168.216.1', { ok: true, ms: 0, ttl: 128 }), 'Reply from 192.168.216.1: time<1ms TTL=128');
  assert.equal(pingWords('10.0.0.1', { ok: true, ms: 4, ttl: 128 }), 'Reply from 10.0.0.1: time=4ms TTL=128');
  assert.equal(pingWords('10.0.0.1', 'failed'), 'No reply from 10.0.0.1');
  assert.equal(realCardLine({ name: 'Ethernet', kind: 'ethernet', connected: false, networks: ['192.168.40.111/24'] }),
    'Ethernet 192.168.40.111/24 · no cable');
});

test('an old lab service is offered Repair in plain words (ADR-0128)', () => {
  assert.ok(read('engine/internal/labsetup/service.go').includes('json:"service_update"'), 'Go decides service_update');
  assert.ok(read('frontend/src/features/network/LabNetwork.tsx').includes('status.service_update'));
  for (const words of [LAB_COPY.serviceUpdate, LAB_COPY.serviceUpdateHint, LAB_COPY.serviceUpdateStopFirst]) {
    assert.ok(words.length > 20 && words.endsWith('.'), words);
  }
});
