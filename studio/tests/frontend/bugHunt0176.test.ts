// Change 0176 (bug hunt 2026-10-23, group 1): SLD and runtime fixes that the browser owns.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { SBO_IDLE, sboReducer, type SboEvent, type SboState } from '../../frontend/src/equipment/selectOperate.ts';
import { createSavedSwitch } from '../../frontend/src/equipment/savedSwitch.ts';
import { operateWords } from '../../frontend/src/equipment/operateWords.ts';
import { controlChoices } from '../../frontend/src/equipment/tapChanger.ts';
import { settingFromText } from '../../frontend/src/equipment/measurementDisplay.ts';
import { freeSpot, NEW_SIZE } from '../../frontend/src/equipment/placeNew.ts';
import { skippedReason, sldSummary } from '../../frontend/src/features/diagram/sldPreview.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const run = (events: SboEvent<string>[]) =>
  events.reduce((state: SboState<string>, event) => sboReducer(state, event), SBO_IDLE as SboState<string>);

test('B1: a Cancel the device refuses keeps the selection and says why', () => {
  const held = run([{ type: 'select', choice: 'close' }, { type: 'selected', windowMs: 27000, now: 1000 },
    { type: 'cancel' }, { type: 'cancelRefused', message: "This device's model has no Cancel" }]);
  assert.equal(held.phase, 'selected', 'the device still holds it, so Operate stays available');
  assert.equal(held.deadline, 28000, 'the countdown keeps running');
  assert.equal(held.tone, 'failed');
  assert.match(held.message, /no Cancel/);
  const operated = sboReducer(sboReducer(held, { type: 'operate' }), { type: 'operated', message: '' });
  assert.equal(operated.phase, 'idle');
  assert.equal(sboReducer(SBO_IDLE as SboState<string>, { type: 'cancelRefused', message: 'x' }).phase, 'idle',
    'a late answer never revives an ended selection');
  assert.match(read('features/diagram/sbo/useSelectOperate.ts'), /type: 'cancelRefused'/);
});

test('B17: a test command says the equipment did not move', () => {
  assert.equal(operateWords({ action: 'close', wire_value: true, message: 'Test command accepted; equipment not moved' }),
    'Test command accepted; equipment not moved');
  assert.equal(operateWords({ action: 'close', wire_value: true }), 'Close done · sent 1');
  assert.equal(operateWords({ action: 'open', wire_value: null }), 'Open done');
});

test('B15: a failed save of the step-by-step switch moves it back and keeps the reason', async () => {
  let fail = true;
  const saved = createSavedSwitch(async () => { if (fail) throw new Error('No ScadaBench session'); });
  let seen = 0;
  saved.subscribe(() => { seen++; });
  await saved.set(true);
  assert.equal(saved.get(), false, 'not saved, so it is off again');
  assert.equal(saved.error(), 'No ScadaBench session');
  assert.ok(seen >= 2, 'listeners hear the move and the move back');
  fail = false;
  await saved.set(true);
  assert.equal(saved.get(), true);
  assert.equal(saved.error(), '', 'a good save clears the reason');
});

test('B2: a saved tap control is always an option, never "Not mapped"', () => {
  const saved = { ied_id: 'T', ref: 'LD/ATCC1.TapChg' };
  const none = controlChoices([], saved, 'T');
  assert.equal(none.savedMissing, true);
  assert.deepEqual(none.choices.map(item => item.ref), ['LD/ATCC1.TapChg'], 'shown while the typed list is loading');
  const listed = controlChoices([{ ref: 'LD/ATCC1.TapChg', cdc: 'BSC' }], saved, 'T');
  assert.equal(listed.savedMissing, false);
  assert.deepEqual(listed.choices.map(item => item.cdc), ['BSC']);
  assert.equal(controlChoices([], saved, 'Other').choices.length, 0, 'another IED does not list it');
  const fields = read('features/diagram/tap/TapFields.tsx');
  assert.doesNotMatch(fields, /setIed\(event\.target\.value\); change\(null\)/, 'picking an IED never wipes the binding');
  assert.match(read('features/diagram/tap/TapChangerSheet.tsx'), /useTypedControlsOf\(project\.ieds\)/);
});

test('B4 and B3: Enter in a number box keeps the question open; the list starts on no value', () => {
  const source = read('features/diagram/points/PointCommand.tsx');
  assert.match(source, /event\.key === 'Enter'\) \{ event\.preventDefault\(\); apply\(\); \}/);
  assert.match(source, /useState\(''\);\r?\n  if \(!labels\.length\)/,'the first label ("off") is not preselected');
  assert.match(source, /<option value="" disabled>\{LIST_PLACEHOLDER\}<\/option>/);
});

test('B10: a measurement scale of 0 is refused like the point panel', () => {
  assert.match(settingFromText('scale', '0').error ?? '', /Scale cannot be 0/);
  assert.deepEqual(settingFromText('scale', '0.001').patch, { scale: 0.001 });
  assert.deepEqual(settingFromText('offset', '0').patch, { offset: 0 }, 'an offset of 0 is fine');
});

test('B13: PTR and IFL point to the Add menu instead of "no symbol yet"', () => {
  assert.match(skippedReason({ type: 'PTR', reason: 'ScadaBench has no symbol for this equipment type yet' }),
    /Add a Tap-changer transformer from the Add menu/);
  assert.match(skippedReason({ type: 'IFL', reason: 'r' }), /Add an Infeed from the Add menu/);
  assert.match(skippedReason({ type: 'PowerTransformer', reason: 'r' }), /Tap-changer transformer/,
    'a <PowerTransformer> element is listed under its element name');
  assert.equal(skippedReason({ type: 'GEN', reason: 'engine words' }), 'engine words');
  const lines = sldSummary({ diagram: { nodes: [], edges: [] }, revision: 1, layout: 'automatic', removed: 0, drawn: [],
    skipped: [{ path: 'p', type: 'PTR', reason: 'r' }], unbound: [], diagnostics: [], already_present: [] });
  assert.ok(!lines.some(line => line.includes('no symbol yet')));
});

test('B7: a clicked symbol lands in the visible middle at a free spot', () => {
  const center = { x: 500, y: 300 };
  const first = freeSpot([], center);
  assert.deepEqual(first, { x: 440, y: 240 }, 'centred on the view, on the 20 px grid');
  const taken = [{ x: first.x, y: first.y, ...NEW_SIZE }];
  const second = freeSpot(taken, center);
  assert.notDeepEqual(second, first);
  const overlap = second.x < first.x + NEW_SIZE.width && first.x < second.x + NEW_SIZE.width &&
    second.y < first.y + NEW_SIZE.height && first.y < second.y + NEW_SIZE.height;
  assert.equal(overlap, false, 'never on top of another symbol');
  assert.ok(Math.abs(second.x - first.x) <= 200 && Math.abs(second.y - first.y) <= 200, 'still near the middle');
});

test('B6: the tidy note floats, so the toolbar buttons do not move', () => {
  assert.match(read('features/diagram/diagram.css'), /\.tidy-note \{ position: absolute;/);
  assert.match(read('features/diagram/TidyButton.tsx'), /className="tidy-wrap"/);
});
