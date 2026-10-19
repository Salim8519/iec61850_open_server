import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { REACH_STATES, reachStatus, shareWarning } from '../../frontend/src/copy/gooseReach.ts';
import { TONES } from '../../frontend/src/copy/status.ts';

const read = (path: string) => readFileSync(new URL(`../../${path}`, import.meta.url), 'utf8');

test('every reach state the backend reports has words, an icon and a hint', () => {
  const backend = read('backend/studio_app/domain/goose_reach.py') + read('backend/studio_app/acquisition/goose_monitor.py');
  const states = new Set([...backend.matchAll(/return "([a-z_]+)", /g), ...backend.matchAll(/"state": "([a-z_]+)"/g)]
    .map(match => match[1]));
  assert.ok(states.has('sharing') && states.has('listening') && states.has('share_held'));
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

test('sharing is sent only with confirm and the adapter shown; a conflict needs two clicks to continue', () => {
  const reach = read('frontend/src/features/network/GooseReach.tsx');
  assert.match(reach, /reach: 'outside', adapter_guid: share\.uplink\.guid, confirm: true/);
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
