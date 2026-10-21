// SNMP devices (ADR-0092): a device that could not start says so, and no card or button text is clipped.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { SNMP_COPY } from '../../frontend/src/copy/snmp.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('a Start that leaves devices unstarted says so in the left column, in plain words', () => {
  assert.equal(SNMP_COPY.notStarted(1, 0), 'The device could not start. The reason is under it in the list.');
  assert.equal(SNMP_COPY.notStarted(2, 0), '2 devices could not start. The reason is under each one in the list.');
  assert.equal(SNMP_COPY.notStarted(1, 2), '1 device could not start. The reason is under it in the list.');
  const view = read('features/tools/snmp/simulate/SimulateView.tsx');
  assert.match(view, /item\.state === 'failed'\)\.length/);
  assert.match(view, /role="alert">\s*⚠ \{SNMP_COPY\.notStarted\(failed, status\.running\)\}/);
});

test('device cards and the Connect button wrap long text instead of cutting it', () => {
  const css = read('features/tools/snmp/snmp.css');
  const card = css.slice(css.indexOf('.snmp-device {'), css.indexOf('.snmp-device.active'));
  assert.match(card, /grid-template-columns: minmax\(0, 1fr\)/);
  assert.match(card, /justify-content: stretch/);
  assert.match(card, /white-space: normal/);
  const connect = css.slice(css.indexOf('.snmp-connect > button'));
  assert.match(connect.slice(0, connect.indexOf('}')), /white-space: normal/);
});
