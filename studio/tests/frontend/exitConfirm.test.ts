import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { rememberAppWindow } from '../../frontend/src/app/useExitConfirm.ts';

const SRC = new URL('../../frontend/src/', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
const read = (path: string) => readFileSync(SRC + path, 'utf8');

function fakeStorage() {
  const items = new Map<string, string>();
  (globalThis as { sessionStorage?: unknown }).sessionStorage = {
    getItem: (key: string) => items.get(key) ?? null, setItem: (key: string, value: string) => items.set(key, value),
  };
  return items;
}

test('only the packaged app window asks before closing', () => {
  const items = fakeStorage();
  rememberAppWindow(new URLSearchParams('session=abc'));
  assert.equal(items.size, 0);
  rememberAppWindow(new URLSearchParams('session=abc&window=app'));
  assert.equal(items.get('scadabench-app-window'), '1');
  assert.match(read('api/client.ts'), /rememberAppWindow\(fragment\);[\s\S]*replaceState/);
  assert.match(read('app/App.tsx'), /useExitConfirm\(\);/);
});

test('Save as never navigates away: a Save dialog, else a download link (ADR-0135)', () => {
  const saveAs = read('state/saveProjectAs.ts');
  assert.doesNotMatch(saveAs, /location\.href/);
  assert.match(saveAs, /showSaveFilePicker/);
  assert.match(saveAs, /link\.download = suggestedName/);
});
