import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { LicenseStatus } from '../../frontend/src/api/license.ts';
import { licenseState, LICENSE_COPY } from '../../frontend/src/copy/license.ts';
import { VM_HOST_COPY, vmHostState, vmHostSummary } from '../../frontend/src/copy/vmHost.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const base: LicenseStatus = { licensed: true, mode: 'vm-host', plan: 'lifetime', days_remaining: null, expires_at: null,
  reason: null, license_id: 'SUBS-AAAA-BBBB-CCCC-DDDD', email: null, enforced: true, server_url: 'https://license.example.com',
  masked_key: null, hwid: 'ab'.repeat(32), transfer_available: false, host_key: 'AbCdEf...WxYz' };

test('the switch has the agreed name and every word is plain English', () => {
  assert.equal(VM_HOST_COPY.switchLabel, 'Let my virtual machines use this license');
  const everything = Object.values(VM_HOST_COPY).join(' ');
  for (const jargon of ['Ed25519', 'DPAPI', 'nonce', 'HKCU', 'registry', 'TCP', 'handshake', 'cryptograph']) {
    assert.ok(!everything.includes(jargon), `"${jargon}" is not beginner wording`);
  }
  assert.match(VM_HOST_COPY.how, /never on your office or internet network/);
  assert.match(VM_HOST_COPY.how, /Turn the switch off and the helper stops by itself/);
  assert.match(VM_HOST_COPY.firewallWhy, /Clean up removes it again/);
  assert.match(VM_HOST_COPY.firewallAdmin, /administrator permission/);
});

test('a virtual machine says it is licensed through its host PC and shows the host key', () => {
  const state = licenseState(base);
  assert.equal(state.label, 'Licensed through your host PC');
  assert.equal(state.tone, 'good');
  assert.match(state.hint, /Let my virtual machines use this license/);
  assert.match(state.hint, /Host key: AbCdEf\.\.\.WxYz\./);
  assert.equal(vmHostState({ ...base, host_key: undefined }).hint, VM_HOST_COPY.insideVmHint);
  assert.equal(licenseState({ ...base, mode: 'lifetime' }).label, 'Lifetime license');
});

test('the sentence under the switch follows the switch and the helper', () => {
  assert.equal(vmHostSummary(false, false, true), VM_HOST_COPY.stateOff);
  assert.equal(vmHostSummary(true, true, true), VM_HOST_COPY.stateOn);
  assert.equal(vmHostSummary(true, false, true), VM_HOST_COPY.stateOnWaiting);
  assert.equal(vmHostSummary(false, false, false), VM_HOST_COPY.noHelper);
  assert.match(VM_HOST_COPY.stateOnWaiting, /next time you sign in/);
});

test('the routes are the Go-only license routes and the firewall call sends what was reviewed', () => {
  const api = read('api/vmHost.ts');
  assert.match(api, /api<VmHostStatus>\('\/license\/host-key'\)/);
  assert.match(api, /api<VmHostStatus>\('\/license\/host-key', 'PUT', \{ enabled \}\)/);
  assert.match(api, /api<VmFirewallPreview>\('\/license\/host-key\/firewall'\)/);
  assert.match(api, /'\/license\/host-key\/firewall', 'POST', \{ fingerprint, confirm: true \}/);
  assert.match(api, /api<\{ code: string \}>\('\/license\/activation-code'\)/);
  const firewall = read('features/settings/VmHostFirewall.tsx');
  assert.match(firewall, /applyVmFirewall\(preview!\.fingerprint\)/);
  assert.match(firewall, /setConfirming\(true\)/);  // review first, apply only after the second click
  assert.match(firewall, /preview\.items\.filter\(item => item\.action === 'add'\)/);
});

test('turning it off is always possible, turning it on needs a shareable license, and a missing backend hides it', () => {
  const section = read('features/settings/VmHostSection.tsx');
  assert.match(section, /const blocked = !status\.available && !status\.enabled;/);
  assert.match(section, /disabled=\{busy \|\| blocked\}/);
  assert.match(section, /catch\(\(\) => setHidden\(true\)\)/);
  assert.match(section, /if \(hidden\) return null;/);
  assert.match(section, /status\.reason/);
  assert.match(section, /status\.enabled && status\.available && <VmHostFirewall/);
});

test('the section is on the License tab only for an enforced lifetime license, and the code is in the offline options', () => {
  const section = read('features/settings/LicenseSection.tsx');
  assert.match(section, /status\.enforced && status\.mode === 'lifetime' && <VmHostSection \/>/);
  assert.match(section, /VM_HOST_COPY\.hostKeyRow, status\.host_key/);
  const actions = read('features/license/LicenseActions.tsx');
  assert.match(actions, /<ActivationCode \/>/);
  const code = read('features/license/ActivationCode.tsx');
  assert.match(code, /if \(!code\) return null;/);
  assert.match(code, /navigator\.clipboard\.writeText\(code\)/);
  assert.equal(VM_HOST_COPY.copyActivation, 'Copy activation code');
  assert.equal(LICENSE_COPY.offlineSteps.length, 3);
  assert.match(LICENSE_COPY.offlineSteps.join(' '), /activation code/);
  assert.match(read('api/license.ts'), /'vm-host'/);
});

test('the activation screen asks again while a virtual machine may be finding its host', () => {
  const gate = read('features/license/LicenseGate.tsx');
  assert.match(gate, /<VmHostNotice changed=\{changed\} \/>/);
  const notice = read('features/license/VmHostNotice.tsx');
  assert.match(notice, /if \(status\.licensed\) changed\(status\)/);
  assert.match(notice, /window\.clearInterval\(timer\)/);
  assert.match(VM_HOST_COPY.gateSteps[1], /tick "Let my virtual machines use this license"/);
});

test('nothing in the new screens is cut off: text wraps and no ellipsis is used', () => {
  const css = read('features/settings/vm-host.css') + read('features/license/license-gate.css');
  assert.doesNotMatch(css, /text-overflow|ellipsis|white-space:\s*nowrap/);
  assert.match(css, /\.vm-host-switch span \{ overflow-wrap: anywhere; \}/);
  assert.match(read('features/license/ActivationCode.tsx'), /<textarea readOnly rows=\{3\}/);
  for (const file of ['VmHostSection.tsx', 'VmHostFirewall.tsx']) {
    assert.doesNotMatch(read(`features/settings/${file}`), /text-overflow|ellipsis|\.slice\(|substring/);
  }
});

test('the activation screen has a clear "Using a virtual machine?" box beside offline activation', () => {
  assert.equal(VM_HOST_COPY.gateTitle, 'Using a virtual machine?');
  assert.equal(VM_HOST_COPY.gateSteps.length, 3);
  assert.match(VM_HOST_COPY.gateSteps[1], /Let my virtual machines use this license/);
  assert.match(VM_HOST_COPY.gateSteps[2], /continues on its own/);
  assert.match(VM_HOST_COPY.gateIntro, /no key, no file and no internet/);
  const gate = read('features/license/LicenseGate.tsx');
  assert.ok(gate.indexOf('<ServerOptions') < gate.indexOf('<VmHostNotice'), 'the box sits right after the offline options');
  const box = read('features/license/VmHostNotice.tsx');
  assert.match(box, /<Disclosure title=\{COPY\.gateTitle\}>/);  // closed like the offline options
  assert.match(box, /<ol className="small license-steps">/);
  assert.match(box, /<Spinner label=\{COPY\.gateLooking\}/);
});
