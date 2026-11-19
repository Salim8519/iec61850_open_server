// Fix plan A17: a project read with the same saved settings keeps what the user typed.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { LiveSettings } from '../../frontend/src/api/types.ts';
import { draftKey } from '../../frontend/src/features/devices/liveDraft.ts';

const saved: LiveSettings = { ip: '192.168.1.20', port: 102, access: 'auto_reports', poll_ms: 1000 };

test('an equal live object from a new project read keeps the draft', () => {
  assert.equal(draftKey('ied-1', { ...saved }), draftKey('ied-1', saved));
});

test('another IED or new saved values start the form again', () => {
  assert.notEqual(draftKey('ied-2', saved), draftKey('ied-1', saved));
  assert.notEqual(draftKey('ied-1', { ...saved, port: 103 }), draftKey('ied-1', saved));
  assert.notEqual(draftKey('ied-1', { ...saved, allow_commands: true }), draftKey('ied-1', saved));
});

test('the form resets on the saved values, not on the live object itself', () => {
  const form = readFileSync(new URL('../../frontend/src/features/devices/DeviceConnection.tsx', import.meta.url), 'utf8');
  assert.match(form, /const saved = draftKey\(ied\.id, ied\.live\);/);
  assert.match(form, /useEffect\(\(\) => setDraft\(ied\.live\), \[saved\]\);/);
  assert.doesNotMatch(form, /\[ied\.id, ied\.live\]/);
});
