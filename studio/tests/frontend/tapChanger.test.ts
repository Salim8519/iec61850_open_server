import assert from 'node:assert/strict';
import test from 'node:test';
import {
  avrLine, isAuto, lvText, modeWords, newTapChanger, rangeProblem, stepBlock, tapControls, tapText,
} from '../../frontend/src/equipment/tapChanger.ts';
import { defaultLabel, kindInfo, setupItems } from '../../frontend/src/equipment/kindInfo.ts';
import type { EquipmentData, Ied, TapChanger } from '../../frontend/src/api/types.ts';

// Tap-changer transformer (ADR-0109): the browser only turns what the engine publishes into words.

const binding = (ref: string) => ({ ied_id: 'ied1', report_ref: null, ref });
const bound = (patch: Partial<TapChanger> = {}): TapChanger =>
  ({ ...newTapChanger(), tap_source: binding('LD0/YLTC1.TapChg.valWTr.posVal'), command_mode: 'spc_pair',
    raise_control: { ied_id: 'ied1', ref: 'LD0/GGIO1.SPCSO1' }, lower_control: { ied_id: 'ied1', ref: 'LD0/GGIO1.SPCSO2' },
    ...patch });

test('a new tap changer matches the engine defaults', () => {
  const tap = newTapChanger();
  assert.equal(tap.command_mode, 'none');
  assert.deepEqual([tap.min_tap, tap.max_tap, tap.nominal_tap], [1, 17, 9]);
  assert.equal(tap.avr.enabled, false);
  assert.equal(tap.avr.target_kv, 11);
  assert.notEqual(newTapChanger().avr, tap.avr, 'every transformer gets its own regulator settings');
});

test('the tap reads "Tap 9 / 1–17" and says when it is missing or not known', () => {
  assert.equal(tapText(bound(), 9), 'Tap 9 / 1–17');
  assert.equal(tapText(bound(), undefined), 'Awaiting tap');
  assert.equal(tapText(newTapChanger(), 9), 'Tap not mapped');
  assert.equal(tapText(bound({ min_tap: -8, max_tap: 8 }), 0), 'Tap 0 / -8–8');
});

test('Auto/Manual comes from the bound status, else from the regulator', () => {
  const withStatus = bound({ mode_source: binding('LD0/ATCC1.Auto.stVal'), auto_value: 2 });
  assert.equal(isAuto(withStatus, true, undefined), true);
  assert.equal(isAuto(withStatus, false, { auto: true }), false);
  assert.equal(isAuto(withStatus, 2, undefined), true);
  assert.equal(isAuto(withStatus, 1, undefined), false);
  assert.equal(isAuto(withStatus, undefined, undefined), undefined);
  assert.equal(isAuto(bound(), undefined, { auto: true }), true);
  assert.equal(isAuto(bound(), undefined, undefined), false);
  assert.equal(modeWords(true), 'Auto');
  assert.equal(modeWords(false), 'Manual');
  assert.equal(modeWords(undefined), 'Mode unknown');
});

test('Raise and Lower say why they cannot be pressed', () => {
  assert.equal(stepBlock(bound(), 9, false, 'raise'), null);
  assert.match(stepBlock(bound(), 17, false, 'raise') ?? '', /highest tap \(17\)/);
  assert.match(stepBlock(bound(), 1, false, 'lower') ?? '', /lowest tap \(1\)/);
  assert.match(stepBlock(bound(), 9, true, 'raise') ?? '', /Switch to Manual/);
  assert.match(stepBlock(newTapChanger(), 9, false, 'raise') ?? '', /Choose the raise and lower command/);
  assert.equal(stepBlock(bound(), undefined, false, 'raise'), null, 'an unknown tap leaves the check to the engine');
  assert.match(stepBlock(bound({ lower_control: null }), 9, false, 'raise') ?? '', /Choose the raise and lower command/,
    'a pair still being chosen cannot be sent');
});

test('the regulator line and LV voltage are words', () => {
  assert.equal(avrLine(undefined), '');
  assert.equal(avrLine({ regulator: false, auto: true }), '');
  assert.equal(avrLine({ regulator: true, state: 'in_band' }), 'Voltage in band');
  assert.equal(avrLine({ regulator: true, state: 'timing', remaining_s: 12 }), 'Out of band, next step in 12 s');
  assert.equal(avrLine({ regulator: true, state: 'manual' }), '', 'the mode badge already says Manual');
  assert.match(avrLine({ regulator: true, state: 'blocked', reason: 'real_ied' }), /never commands real IEDs/);
  assert.equal(lvText(11.1375, undefined), 'LV 11.14 kV');
  assert.equal(lvText(undefined, { lv_kv: 10.8625 }), 'LV 10.86 kV');
  assert.equal(lvText(undefined, undefined), '');
});

test('controls are offered per role: BSC/ISC steps, SPC pairs, SPC/ENC modes', () => {
  const ied = { id: 'ied1', name: 'RTCC1', controls: [{ ref: 'LD0/GGIO1.SPCSO1', cdc: 'SPC' }, { ref: 'LD0/CSWI1.Pos', cdc: 'DPC' }],
    extended_controls: [{ ref: 'LD0/YLTC1.TapChg', cdc: 'BSC' }, { ref: 'LD0/YLTC2.TapPos', cdc: 'ISC' },
      { ref: 'LD0/ATCC1.Mod', cdc: 'ENC' }, { ref: 'LD0/MMXU1.SetV', cdc: 'APC' }] } as unknown as Ied;
  const offered = tapControls(ied);
  assert.deepEqual(offered.step.map(item => item.cdc), ['BSC', 'ISC']);
  assert.deepEqual(offered.spc.map(item => item.ref), ['LD0/GGIO1.SPCSO1']);
  assert.deepEqual(offered.mode.map(item => item.cdc), ['SPC', 'ENC']);
  const before = tapControls({ ...ied, extended_controls: undefined });
  assert.equal(before.step.length, 0, 'without typed controls (before change 0167) only SPC pairs are offered');
});

test('the tap range is checked in plain words', () => {
  assert.equal(rangeProblem({ min_tap: 1, max_tap: 17, nominal_tap: 9 }), null);
  assert.match(rangeProblem({ min_tap: 5, max_tap: 5, nominal_tap: 5 }) ?? '', /smaller than the highest/);
  assert.match(rangeProblem({ min_tap: 1, max_tap: 17, nominal_tap: 20 }) ?? '', /inside the range/);
});

test('the kind has a name, a label and setup items', () => {
  assert.equal(kindInfo('TAP_TRANSFORMER').name, 'Tap-changer transformer');
  assert.equal(defaultLabel('TAP_TRANSFORMER', 2), 'T-2');
  const data = { kind: 'TAP_TRANSFORMER', label: 'T-1', rotation: 0, tap_changer: newTapChanger() } as EquipmentData;
  assert.deepEqual(setupItems(data).map(item => [item.label, item.done]), [['Tap position', false], ['Raise/Lower command', false]]);
  const ready = { ...data, tap_changer: bound() };
  assert.ok(setupItems(ready).every(item => item.done));
});
