import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { buyUrl, isOtherComputer, offlineUrl, type LicenseStatus } from '../../frontend/src/api/license.ts';
import { bindingText, LICENSE_COPY, licenseState } from '../../frontend/src/copy/license.ts';

const base: LicenseStatus = { licensed: false, mode: 'unlicensed', plan: null, days_remaining: null, expires_at: null,
  reason: 'This computer has no license yet.', license_id: null, email: null, enforced: true,
  server_url: 'https://license.example.com', masked_key: null, hwid: 'ab'.repeat(32), transfer_available: false };
const boundary = readFileSync(new URL('../../frontend/src/features/license/LicenseBoundary.tsx', import.meta.url), 'utf8');
const main = readFileSync(new URL('../../frontend/src/main.tsx', import.meta.url), 'utf8');
const client = readFileSync(new URL('../../frontend/src/api/client.ts', import.meta.url), 'utf8');
const actions = readFileSync(new URL('../../frontend/src/features/license/LicenseActions.tsx', import.meta.url), 'utf8');
const api = readFileSync(new URL('../../frontend/src/api/license.ts', import.meta.url), 'utf8');

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

test('a license works on one computer only: no move is offered anywhere', () => {
  assert.doesNotMatch(actions, /activate\(true\)|bound_elsewhere|moveButton|move: /);
  assert.doesNotMatch(api, /move\s*[=:,)]|\/transfer/);  // no move argument or call (comments may say "move")
  assert.match(api, /export const activateLicense = \(licenseKey: string\) =>/);
  assert.equal('moveButton' in LICENSE_COPY, false);
  assert.match(LICENSE_COPY.keyHint, /one computer only and cannot be moved/);
  assert.ok(LICENSE_COPY.offlineSteps.length === 3);
  assert.match(LICENSE_COPY.safeData, /projects are safe/);
});

test('the license server address is shown but cannot be changed', () => {
  assert.doesNotMatch(api, /updateLicenseServerUrl|\/config/);
  assert.doesNotMatch(actions, /updateLicenseServerUrl|setServer/);
  assert.match(actions, /<input readOnly value=\{status\.server_url\}/);
  assert.match(LICENSE_COPY.serverHint, /cannot be changed/);
});

test('a license file made for another computer still lets this computer start a trial', () => {
  const other = { ...base, plan: 'lifetime', reason: 'This license belongs to a different computer. Each license works on one computer only; buy a new license for this computer, or start a trial.' };
  assert.equal(isOtherComputer(other), true);
  assert.equal(isOtherComputer({ ...base, plan: 'trial', reason: 'Your free trial has ended.' }), false);
  assert.equal(isOtherComputer(base), false);
});

test('a new computer can start the trial; only an expired trial disables it', () => {
  const gate = readFileSync(new URL('../../frontend/src/features/license/LicenseGate.tsx', import.meta.url), 'utf8');
  assert.match(gate, /const trialEnded = status\.plan === 'trial' && !isOtherComputer\(status\);/);
  assert.doesNotMatch(gate, /transfer_available/);
  assert.doesNotMatch(gate, /reason\?\.includes\('trial'\)/);
});

test('a trial that could not confirm the date online offers "Check online now"', () => {
  const gate = readFileSync(new URL('../../frontend/src/features/license/LicenseGate.tsx', import.meta.url), 'utf8');
  assert.match(gate, /status\.online_check_required && /);
  assert.match(gate, /online\.run\(checkOnline, changed\)/);
  assert.equal(LICENSE_COPY.checkOnlineButton, 'Check online now');
});

test('a trial the server wants confirmed asks for an email, and a mail outage never blocks', () => {
  const start = readFileSync(new URL('../../frontend/src/features/license/TrialStart.tsx', import.meta.url), 'utf8');
  assert.match(start, /'email_required'/);
  assert.match(start, /answer\.status === 'no_code_needed'\) await trial\.run\(\(\) => startTrial\(email\), changed\)/);
  assert.match(start, /startTrial\(email, code\)/);
  assert.match(LICENSE_COPY.emailHint, /one free trial per person/);
});

test('a license says how it is tied to this computer, in plain words', () => {
  const lifetime = { ...base, licensed: true, mode: 'lifetime' as const, plan: 'lifetime', reason: null };
  assert.equal(bindingText({ ...lifetime, binding: 'tpm' }), "This computer's security chip");
  assert.equal(bindingText({ ...lifetime, binding: 'lease' }), 'Confirms online every 30 days');
  assert.equal(bindingText(lifetime), null);
  assert.match(licenseState({ ...lifetime, binding: 'tpm' }).hint, /tied to this computer's security chip/);
  assert.match(licenseState({ ...lifetime, binding: 'tpm' }).hint, /works offline/);
  assert.match(licenseState({ ...lifetime, binding: 'lease' }).hint, /up to 30 days/);
  assert.equal(licenseState(lifetime).label, 'Lifetime license');
  const section = readFileSync(new URL('../../frontend/src/features/settings/LicenseSection.tsx', import.meta.url), 'utf8');
  assert.match(section, /LICENSE_COPY\.bindingLabel, binding/);
});

test('a lease that needs one online check is shown with the same button and its own hint', () => {
  const gate = readFileSync(new URL('../../frontend/src/features/license/LicenseGate.tsx', import.meta.url), 'utf8');
  assert.match(gate, /status\.plan === 'trial' \? COPY\.checkOnlineHint : COPY\.checkOnlineLeaseHint/);
  assert.match(LICENSE_COPY.checkOnlineLeaseHint, /Confirms your license/);
  assert.match(api, /binding\?: 'tpm' \| 'lease'/);
});
