// Change 0221, trial N8: the engine watches a running gateway's listen address; the page only
// shows what the engine says (address_gone and its message), in words, not colour alone.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { GATEWAY_COPY } from '../../frontend/src/copy/gateway.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('a gone listen address turns the running pill into words and an alert', () => {
  const source = read('features/tools/gateway/GatewaySettings.tsx');
  assert.equal(GATEWAY_COPY.addressGone, 'Running, address gone');
  assert.match(source, /const gone = running && Boolean\(status\.address_gone\)/);
  assert.match(source, /gone \? \{ label: copy\.addressGone, tone: 'problem'/);
  assert.match(source, /\{gone && <p className="side-error" role="alert">\{status\.message\}<\/p>\}/);
  assert.match(source, /\{running && !gone && <p className="side-line"/);
  assert.match(read('api/gateway.ts'), /address_gone\?: boolean/);
});
