import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { EquipmentData, Ied } from '../../frontend/src/api/types.ts';
import { testAvailability, testButtons } from '../../frontend/src/equipment/testButtons.ts';
import { runStateWords } from '../../frontend/src/copy/running.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const point = { ied_id: 'a', ref: 'LD/XCBR1.Pos.stVal', report_ref: null };
const cb = { kind: 'CB', label: 'Q0', rotation: 0, state_mapping: { mode: 'direct', source: point } } as EquipmentData;
const ied = (role: string) => ({ id: 'a', name: 'C03_Y1105', role }) as unknown as Ied;

test('every equipment kind gets its test buttons (change 0097)', () => {
  assert.deepEqual(testButtons(cb).map(item => item.label), ['Open', 'Closed', 'Moving', 'Bad state']);
  assert.deepEqual(testButtons({ ...cb, kind: 'TRIP_RELAY' }).map(item => item.label), ['Trip', 'Reset']);
  assert.deepEqual(testButtons({ ...cb, kind: 'CT' }).map(item => item.label), ['Set value', 'Bad quality']);
  assert.deepEqual(testButtons({ ...cb, kind: 'SOURCE' }).map(item => item.label), ['Live', 'Dead']);
  assert.deepEqual(testButtons({ ...cb, kind: 'JUNCTION' }), []);
  assert.deepEqual(testButtons({ ...cb, kind: 'SOURCE', state_mapping: null }), [], 'only when bound to a point');
  const relay = { ...cb, kind: 'RELAY', relay_statuses: [{ id: 'r', label: 'Trip L1', source: point }] } as EquipmentData;
  assert.deepEqual(testButtons(relay).map(item => item.label), ['Trip L1 on', 'Trip L1 off']);
});

test('test buttons only for running simulated IEDs; real IEDs use commands', () => {
  assert.equal(testAvailability(cb, [ied('simulated')], new Set(['a'])).buttons.length, 4);
  assert.deepEqual(testAvailability(cb, [ied('simulated')], new Set()), { buttons: [], reason: 'not_running' });
  assert.deepEqual(testAvailability(cb, [ied('real')], new Set(['a'])), { buttons: [], reason: 'real' });
  const sheet = read('features/diagram/EquipmentSheet.tsx');
  assert.match(sheet, /locked && <EquipmentTest/);
  assert.match(read('equipment/testButtons.ts'), /moved by itself. No command is sent/);
  assert.match(read('features/diagram/DiagramPage.tsx'), /\/test`/, 'writes go through the backend');
});

test('the top bar says how many IEDs run', () => {
  assert.equal(runStateWords(1, 2, false), 'Running 1 of 2 IEDs');
  assert.equal(runStateWords(2, 2, false), 'Running · 2 IEDs');
  assert.match(read('features/devices/RunThisIed.tsx'), /runtime\/ieds\//);
});

test('Devices rows put status pills on their own line so nothing is cut', () => {
  const row = read('features/devices/DeviceListRow.tsx');
  assert.ok(row.indexOf('list-pills') > row.indexOf('data points'), 'pills come after the name and points');
  assert.match(read('features/devices/devices.css'), /\.list-pills \{ display: flex; flex-wrap: wrap/);
});
