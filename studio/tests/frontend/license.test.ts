import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { buyUrl, offlineUrl, type LicenseStatus } from '../../frontend/src/api/license.ts';
import { LICENSE_COPY, licenseState } from '../../frontend/src/copy/license.ts';

const base: LicenseStatus = { licensed: false, mode: 'unlicensed', plan: null, days_remaining: null, expires_at: null,
  reason: 'This computer has no license yet.', license_id: null, email: null, enforced: true,
  server_url: 'https://license.example.com', masked_key: null, hwid: 'ab'.repeat(32), transfer_available: false };
const boundary = readFileSync(new URL('../../frontend/src/features/license/LicenseBoundary.tsx', import.meta.url), 'utf8');
const main = readFileSync(new URL('../../frontend/src/main.tsx', import.meta.url), 'utf8');
const client = readFileSync(new URL('../../frontend/src/api/client.ts', import.meta.url), 'utf8');
const actions = readFileSync(new URL('../../frontend/src/features/license/LicenseActions.tsx', import.meta.url), 'utf8');

test('license state uses words and tone, never colour alone', () => {
  assert.equal(licenseState({ ...base, licensed: true, mode: 'development' }).label, 'Development mode');
  assert.equal(licenseState({ ...base, licensed: true, mode: 'lifetime' }).tone, 'good');
  const trial = licenseState({ ...base, licensed: true, mode: 'trial', days_remaining: 1 });
  assert.equal(trial.label, 'Free trial: 1 day left');
  assert.equal(trial.tone, 'warning');
  const blocked = licenseState(base);
  assert.equal(blocked.label, 'Not activated');
  assert.equal(blocked.hint, base.reason);
});

test('purchase and offline links come from the configured server', () => {
  assert.equal(buyUrl(base), `https://license.example.com/buy?hwid=${'ab'.repeat(32)}`);
  assert.equal(offlineUrl(base), 'https://license.example.com/activate');
  assert.equal(buyUrl({ ...base, server_url: '' }), null);
});

test('the app is wrapped by the license boundary and reacts to 402', () => {
  assert.match(main, /<LicenseBoundary><App \/><\/LicenseBoundary>/);
  assert.match(client, /response\.status === 402/);
  assert.match(client, /studio-license-required/);
  assert.match(boundary, /LICENSE_REQUIRED_EVENT/);
  assert.match(boundary, /status\?\.licensed\) return <>\{children\}<\/>/);
});

test('a key bound to another computer offers a move, with plain wording', () => {
  assert.match(actions, /code === 'bound_elsewhere'/);
  assert.match(actions, /activate\(true\)/);
  assert.ok(LICENSE_COPY.offlineSteps.length === 3);
  assert.match(LICENSE_COPY.safeData, /projects are safe/);
});

test('a new computer can start the trial; only an expired trial disables it', () => {
  const gate = readFileSync(new URL('../../frontend/src/features/license/LicenseGate.tsx', import.meta.url), 'utf8');
  assert.match(gate, /const trialEnded = status\.plan === 'trial' && !status\.transfer_available;/);
  assert.doesNotMatch(gate, /reason\?\.includes\('trial'\)/);
});
