import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { REACH_STATES, reachStatus, shareWarning } from '../../frontend/src/copy/gooseReach.ts';
import { TONES } from '../../frontend/src/copy/status.ts';

const read = (path: string) => readFileSync(new URL(`../../${path}`, import.meta.url), 'utf8');

test('every reach state the Go engine reports has words, an icon and a hint', () => {
  const backend = read('engine/internal/acquisition/goosedom/reach_auto.go')
    + read('engine/internal/acquisition/goosedom/reach_share.go');
  const states = new Set([...backend.matchAll(/return "([a-z_]+)", /g)]
    .map(match => match[1]));
  for (const state of ['sharing', 'listening', 'share_held', 'confirm', 'inside']) {
    assert.ok(states.has(state), `Go reports ${state}`);
  }
  for (const state of states) {
    assert.ok(REACH_STATES[state], `reach state ${state} needs words`);
    assert.ok(reachStatus(state).hint && TONES[reachStatus(state).tone].icon);
  }
  assert.notEqual(reachStatus('sharing').tone, 'good', 'sharing is never shown as plainly fine');
});

test('GOOSE-25: the share warning is stronger when real devices may act on it', () => {
  const normal = shareWarning({ level: 'warning', reasons: [] }, 'Ethernet');
  const high = shareWarning({ level: 'high', reasons: ['real addresses'] }, 'Ethernet');
  assert.match(high.title, /High risk/);
  assert.doesNotMatch(normal.title, /High risk/);
  assert.match(high.body, /trip/);
  assert.notEqual(high.confirm, normal.confirm);
  assert.match(normal.guard, /3 seconds/);
  const dialog = read('frontend/src/features/network/GooseShareWarning.tsx');
  assert.match(dialog, /role="alertdialog"/);
  assert.match(dialog, /onClick=\{cancel\} autoFocus/, 'Cancel has the focus');
  assert.match(dialog, /className="danger"/);
});

test('ADR-0122: GOOSE is shared only after the warning is confirmed with the adapter shown', () => {
  const reach = read('frontend/src/features/network/GooseReach.tsx');
  assert.match(reach, /reach: 'outside', adapter_guid: share\.uplink\.guid, confirm: true/);
  assert.match(reach, /onChange=\{\(\) => setAsking\(true\)\}/, 'choosing to share opens the warning first');
  assert.match(reach, /send\(\{ reach: 'inside' \}\)/, 'one click keeps GOOSE inside');
  assert.match(reach, /NPCAP_URL = 'https:\/\/npcap\.com\/#download'/, 'Npcap has a download link');
  assert.doesNotMatch(reach, /reach: 'auto'|reach: 'local'/, 'the 0.1.28 follow-the-network choices are gone');
  const conflicts = read('frontend/src/features/network/GooseConflicts.tsx');
  assert.match(conflicts, /confirm: choice === 'continue'/);
  assert.match(conflicts, /sure === conflict\.ied_id/);
});

test('no screen says GOOSE needs the lab network any more', () => {
  for (const file of ['frontend/src/copy/hints.ts', 'frontend/src/copy/status.ts', 'frontend/src/copy/signalLayers.ts',
    'frontend/src/features/network/GooseTraffic.tsx', 'frontend/src/features/devices/DeviceOverview.tsx']) {
    assert.doesNotMatch(read(file), /goose_needs_lab|GOOSE (only travels|is sent only after)|works on the lab network only/, file);
  }
});
