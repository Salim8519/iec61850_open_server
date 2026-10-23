// ADR-0113: the smart point picker. The Go engine counts, filters and suggests; the browser chooses the
// slot's type, shows chips and objects with the engine's counts and falls back when an engine has none.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { buildSignalQuery } from '../../frontend/src/components/signalQuery.ts';
import { preferredKind, takesHelpers } from '../../frontend/src/components/signalPicker/preferredKind.ts';
import { allCount, isSmart, kindChips, kindOfPicked, objectOptions, type PickedSignal }
  from '../../frontend/src/components/signalPicker/pickerAnswer.ts';
import { PICKER_COPY } from '../../frontend/src/copy/signalPicker.ts';
import { NUMERIC_KINDS } from '../../frontend/src/equipment/measurementDisplay.ts';
import { MODE_KINDS, TAP_KINDS } from '../../frontend/src/equipment/tapChanger.ts';

const source = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('each slot opens on the type it takes', () => {
  assert.equal(preferredKind({ kinds: ['BOOLEAN'] }), 'SP', 'open or closed contact: on/off points');
  assert.equal(preferredKind({ kinds: ['Dbpos', 'BOOLEAN'] }), 'DP', 'one position point: double points when allowed');
  assert.equal(preferredKind({ kinds: NUMERIC_KINDS }), 'MV', 'measurement list');
  assert.equal(preferredKind({ kinds: NUMERIC_KINDS, fc: 'MX' }), 'MV', 'analog source, LV voltage');
  assert.equal(preferredKind({ kinds: TAP_KINDS, fc: 'ST' }), 'IS', 'tap position');
  assert.equal(preferredKind({ kinds: [...NUMERIC_KINDS.filter(kind => kind.startsWith('INT')), 'Enum'] }), 'IS');
  assert.equal(preferredKind({ kinds: MODE_KINDS, fc: 'ST' }), undefined, 'on/off or number: no single type');
  assert.equal(preferredKind({}), undefined, 'relay rows and test points take any point');
  assert.equal(preferredKind({ kinds: ['FLOAT32'], fc: 'CO' }), 'SE', 'a number command is a set point');
  assert.equal(preferredKind({ kinds: ['BOOLEAN'], fc: 'CO' }), undefined, 'SPC or DPC is not known from BOOLEAN');
  assert.equal(preferredKind({ kinds: ['BOOLEAN'], fc: 'CO', command: 'DC' }), 'DC', 'a command slot names its kind');
});

test('chips show only the types that have rows, the chosen one always, in the kinds order', () => {
  const counts = { DP: 3, SP: 120, CF: 4 };
  assert.deepEqual(kindChips(counts, 'all').map(chip => chip.kind), ['SP', 'DP', 'CF']);
  assert.deepEqual(kindChips(counts, 'MV').map(chip => `${chip.kind}${chip.count}`), ['SP120', 'DP3', 'MV0', 'CF4']);
  assert.equal(allCount(counts), 127);
  assert.equal(allCount({}), 0);
});

test('objects are listed in natural order with their counts; one logical device shows the node alone', () => {
  const counts = { 'System/ConInGGIO10': 5, 'System/ConInGGIO1': 200, 'System/Disc0XSWI2': 3, 'System/ConInGGIO2': 8 };
  assert.deepEqual(objectOptions(counts, '').map(option => `${option.label} (${option.count})`),
    ['ConInGGIO1 (200)', 'ConInGGIO2 (8)', 'ConInGGIO10 (5)', 'Disc0XSWI2 (3)']);
  assert.deepEqual(objectOptions({ 'A/GGIO1': 1, 'B/GGIO1': 2 }, '').map(option => option.label), ['A/GGIO1', 'B/GGIO1']);
  assert.deepEqual(objectOptions({ 'A/GGIO1': 1 }, 'A/XCBR1').map(option => `${option.value}=${option.count}`),
    ['A/GGIO1=1', 'A/XCBR1=0'], 'the chosen object stays even with nothing left');
});

test('the smart query hides helpers by default and asks for counts, kinds, object and suggestions', () => {
  const smart = new URLSearchParams(buildSignalQuery({ q: '106', smart: true, ln: 'System/ConInGGIO1', near: ' Q11 ',
    pointKind: 'SP', kinds: ['BOOLEAN'] }));
  assert.equal(smart.get('helpers'), 'hide');
  assert.equal(smart.get('kind_counts'), '1');
  assert.equal(smart.get('ln_counts'), '1');
  assert.equal(smart.get('annotate'), '1');
  assert.equal(smart.get('ln'), 'System/ConInGGIO1');
  assert.equal(smart.get('near'), 'Q11');
  assert.equal(smart.get('point_kind'), 'SP');
  assert.equal(new URLSearchParams(buildSignalQuery({ smart: true, helpers: true })).get('helpers'), null);
  assert.equal(new URLSearchParams(buildSignalQuery({ smart: true, near: '  ' })).get('near'), null);
  const plain = new URLSearchParams(buildSignalQuery({ q: 'trip' }));
  for (const key of ['helpers', 'kind_counts', 'ln_counts', 'annotate', 'ln', 'near']) assert.equal(plain.get(key), null);
  assert.equal(takesHelpers(undefined), true, 'a slot for any point can show quality and time');
  assert.equal(takesHelpers(['BOOLEAN']), false, 'a contact slot never lists them: no switch');
  assert.equal(takesHelpers(['Quality']), true);
});

test('rows use the kind from the engine; an answer without it (older engine) falls back to the browser rule', () => {
  const base = { ref: 'LD/XCBR1.Pos.Oper.ctlVal', fc: 'CO', kind: 'BOOLEAN', value: false, writable: true, quality: 0,
    timestamp: 0, description: '', enums: {} } satisfies PickedSignal;
  assert.equal(kindOfPicked({ ...base, point_kind: 'DC' }), 'DC');
  assert.equal(kindOfPicked(base), 'CM');
  assert.equal(kindOfPicked(base, [{ ref: 'LD/XCBR1.Pos', cdc: 'DPC' }]), 'DC');
  assert.equal(kindOfPicked({ ...base, point_kind: 'nonsense' }), 'CM');
  assert.equal(isSmart({ items: [], total: 0 }), false);
  assert.equal(isSmart({ items: [], total: 0, kind_counts: {} }), true);
});

test('the picker shows suggestions first, keeps the old type list for older engines and says why a type is chosen', () => {
  const picker = source('components/SignalSearchPicker.tsx');
  assert.ok(picker.indexOf('<SuggestionBlock') < picker.indexOf('className="signal-search-list"'), 'suggestions first');
  assert.match(picker, /answer && !smart && <KindSelect/, 'older engines keep the dropdown');
  assert.match(picker, /legacy \? ALL_KINDS : preferred/, 'older engines start on All types as before');
  assert.match(picker, /<HelperToggle shown=\{!!view\.helpers\}/, 'quality and time hidden until ticked');
  assert.match(source('components/signalPicker/KindChips.tsx'), /<KindBadge kind=\{kind\} \/>/, 'chips use the shared badge');
  assert.match(source('components/signalPicker/KindChips.tsx'), /<Check size=\{13\}/, 'the chosen chip has a tick');
  assert.match(source('components/signal-search-picker.css'), /\.picker-chips \{[^}]*repeat\(auto-fill/, 'chips wrap into columns');
  assert.match(source('components/signal-search-picker.css'), /\.picker-chip \{[^}]*white-space: normal/, 'a long name wraps');
  assert.equal(PICKER_COPY.why('SP', 'Open contact'),
    'Open contact: showing single points (on/off). Pick All types to see every point.');
  assert.equal(PICKER_COPY.mentions('Q11', 'description'), 'Mentions Q11 in its description');
});

test('every SLD picker passes the component name and its slot', () => {
  assert.match(source('features/diagram/StateMappingEditor.tsx'), /near=\{data\.label\} slot=\{slotName\(key\)\}/);
  assert.match(source('features/diagram/MeasurementEditor.tsx'), /near=\{data\.label\} slot="Measurement list"/);
  assert.match(source('features/diagram/RelayStatusEditor.tsx'), /<RelaySignalAdder ied=\{relayIed\} near=\{data\.label\}/);
  assert.match(source('features/diagram/tap/TapFields.tsx'), /near=\{near\} slot=\{title\}/);
  assert.match(source('features/diagram/points/PointAdder.tsx'), /<ObjectSelect counts=\{lnCounts\}/);
});
