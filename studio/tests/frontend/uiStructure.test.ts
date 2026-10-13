import assert from 'node:assert/strict';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';
import { PLACES } from '../../frontend/src/app/places.ts';
import { advancedFields, basicFields } from '../../frontend/src/features/diagram/equipmentFields.ts';

const SRC = new URL('../../frontend/src/', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
const read = (path: string) => readFileSync(join(SRC, path), 'utf8');
const files = (folder: string): string[] => readdirSync(join(SRC, folder)).flatMap(name => {
  const path = `${folder}/${name}`;
  return statSync(join(SRC, path)).isDirectory() ? files(path) : /\.(ts|tsx)$/.test(name) ? [path] : [];
});
const token = (name: string) => {
  const match = read('styles/tokens.css').match(new RegExp(`--${name}:\\s*([^;]+);`));
  assert.ok(match, `token --${name} missing`);
  return match[1].trim();
};
const luminance = (hex: string) => {
  const [r, g, b] = [1, 3, 5].map(index => parseInt(hex.slice(index, index + 2), 16) / 255)
    .map(value => value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};
const contrast = (a: string, b: string) => {
  const [high, low] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (high + 0.05) / (low + 0.05);
};

test('sidebar has exactly four places', () => {
  assert.deepEqual(PLACES.map(item => item.id), ['diagram', 'devices', 'test', 'settings']);
  assert.ok(PLACES.every(item => item.subtitle.length > 20));
});

test('tokens keep text and controls large', () => {
  assert.ok(parseFloat(token('font-size-base')) >= 15);
  assert.ok(parseFloat(token('font-size-small')) >= 13);
  assert.ok(parseFloat(token('control-height')) >= 40);
  assert.ok(parseFloat(token('control-height-large')) >= 44);
});

test('token colours meet AA contrast', () => {
  for (const name of ['text', 'text-2', 'accent', 'success', 'warning', 'error']) {
    assert.ok(contrast(token(name), token('surface')) >= 4.5, `--${name} on surface`);
  }
  assert.ok(contrast(token('text-2'), token('bg')) >= 4.5, '--text-2 on background');
  assert.ok(contrast('#ffffff', token('accent')) >= 4.5, 'white on accent');
});

test('rare equipment options live under Advanced', () => {
  for (const kind of ['CB', 'DS', 'ES', 'CT', 'VT', 'SOURCE', 'RELAY', 'TRIP_RELAY', 'PUSH_BUTTON', 'JUNCTION'] as const) {
    assert.ok(basicFields(kind).length <= 2, `${kind} shows too many basic fields`);
  }
  assert.deepEqual(advancedFields({ kind: 'TRIP_RELAY' }), ['trip_command_polarity', 'trip_state_polarity', 'colors']);
  assert.ok(advancedFields({ kind: 'PUSH_BUTTON', push_mode: 'self_resetting' }).includes('push_reset_ms'));
  const sheet = read('features/diagram/SheetSettings.tsx');
  const basic = sheet.slice(sheet.indexOf('export function BasicSettings'), sheet.indexOf('export function AdvancedSettings'));
  for (const field of ['push_mode', 'trip_command_polarity', 'AppearanceEditor']) assert.ok(!basic.includes(field), field);
  assert.ok(sheet.slice(sheet.indexOf('export function AdvancedSettings')).includes('<Disclosure>'));
});

test('retired equipment kinds are gone from the frontend', () => {
  for (const path of files('.')) {
    assert.doesNotMatch(read(path), /'(BUSBAR|TRANSFORMER|LOAD)'/, `${path} still mentions a retired kind`);
  }
});

test('one value control and one generator editor', () => {
  const owners: Record<string, string[]> = {};
  for (const path of files('.')) {
    const text = read(path);
    for (const route of ['/signals/update', '/signals/pulse', '/signals/release-override', '/signals/source']) {
      if (text.includes(`'${route}'`)) (owners[route] ||= []).push(path);
    }
    for (const name of ['ValueControl', 'SourceEditor']) {
      if (text.includes(`export function ${name}(`)) (owners[name] ||= []).push(path);
    }
  }
  for (const route of ['/signals/update', '/signals/pulse', '/signals/release-override', '/signals/source']) {
    assert.deepEqual(owners[route], ['./features/test/signalActions.ts'], route);
  }
  assert.deepEqual(owners.ValueControl, ['./features/test/ValueControl.tsx']);
  assert.deepEqual(owners.SourceEditor, ['./features/test/SourceEditor.tsx']);
});
