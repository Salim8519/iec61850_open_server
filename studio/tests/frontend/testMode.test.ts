import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { MODE_STATUS, deviceBadge, modeStatus } from '../../frontend/src/copy/testMode.ts';
import { TONES } from '../../frontend/src/copy/status.ts';

const card = readFileSync(new URL('../../frontend/src/features/devices/TestModeCard.tsx', import.meta.url), 'utf8');
const node = readFileSync(new URL('../../frontend/src/equipment/EquipmentNode.tsx', import.meta.url), 'utf8');
const devices = readFileSync(new URL('../../frontend/src/features/devices/DevicesPage.tsx', import.meta.url), 'utf8');
const health = readFileSync(new URL('../../frontend/src/features/devices/HealthView.tsx', import.meta.url), 'utf8');
const modes = readFileSync(new URL('../../backend/studio_app/domain/health_reads.py', import.meta.url), 'utf8');

test('real ied has no mode controls', () => {
  const realBranch = card.slice(card.indexOf("if (ied.role === 'real')"), card.indexOf("if (ied.role !== 'simulated'"));
  assert.ok(realBranch.length > 50);
  assert.doesNotMatch(realBranch, /<select|<input|api\(|onChange/);
  assert.match(card, /if \(ied\.role !== 'simulated' \|\| !devices\.length\) return null;/);
  assert.match(card, /\/test-mode`, 'PUT'/);
});

test('mode badges have text and icon', () => {
  const names = [...modes.matchAll(/\d: "([a-z/-]+)"/g)].map(match => match[1]);
  assert.deepEqual(names, ['on', 'on-blocked', 'test', 'test/blocked', 'off']);
  for (const name of [...names, 'simulation', 'not available']) {
    const status = modeStatus(name);
    assert.equal(status, MODE_STATUS[name], name);
    assert.ok(status.label && status.hint.length > 20 && TONES[status.tone].icon, name);
  }
  assert.equal(deviceBadge({ worst: 'on', simulation: false }), null);
  assert.equal(deviceBadge({ worst: 'on', simulation: true }), 'simulation');
  assert.equal(deviceBadge({ worst: 'test', simulation: false }), 'test');
  // Shown on device rows, on bound diagram equipment and in the health check (mode rows).
  assert.match(devices, /deviceBadge\(modes\.ieds\[item\.id\]\)/);
  assert.match(node, /equipment-mode.*modeStatus\(badge\)\.label/);
  assert.match(health, /Simulation traffic present/);
});
