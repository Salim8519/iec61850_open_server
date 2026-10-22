// Bug hunt 2026-10-23, group 2 (change 0177): Devices, Test, IED Explorer and the app shell.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { createConnectionHealth } from '../../frontend/src/api/connectionHealth.ts';
import { countOf } from '../../frontend/src/copy/count.ts';
import { DEVICES_COPY, friendlyImportError } from '../../frontend/src/copy/devices.ts';
import { FIND_COPY, noneFound } from '../../frontend/src/copy/tools.ts';
import { freeSampleName, shownSelection } from '../../frontend/src/features/devices/devicePick.ts';
import { eventDetails } from '../../frontend/src/features/event-log/eventDetails.ts';
import { rememberIedNames } from '../../frontend/src/features/event-log/iedNames.ts';
import { reportFoldKey } from '../../frontend/src/features/event-log/reportControl.ts';
import { periodProblem, PERIOD_PROBLEM } from '../../frontend/src/features/test/sourceCheck.ts';
import { bulkReadable, selectsOnRead } from '../../frontend/src/features/tools/explorer/readSafety.ts';
import { createEventLogStore } from '../../frontend/src/state/eventLogStore.ts';
import { createFrameBatch } from '../../frontend/src/state/frameBatch.ts';
import { closesOnAway } from '../../frontend/src/ui/menuAway.ts';
import { fakeFrames } from './fakeClock.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const leaf = (ref: string, fc: string) => ({ ref, fc, path: ref.split('.').slice(2).join('.'), type: 'x' });

test('A1: opening a data object never reads a command field, so SBO is never read (no Select by reading)', () => {
  const attributes = [leaf('LD/GGIO2.AnOut1.SBO', 'CO'), leaf('LD/GGIO2.AnOut1.Oper.ctlVal', 'CO'),
    leaf('LD/GGIO2.AnOut1.ctlModel', 'CF'), leaf('LD/GGIO2.AnOut1.mxVal.f', 'MX')];
  assert.deepEqual(bulkReadable(attributes).map(item => item.fc), ['CF', 'MX']);
  assert.equal(selectsOnRead(attributes[0]), true);
  assert.equal(selectsOnRead({ ref: 'LD/GGIO2$CO$AnOut1$SBO', fc: 'co' }), true);
  assert.equal(selectsOnRead(leaf('LD/GGIO2.AnOut1.SBOw.ctlVal', 'CO')), false);
  assert.equal(selectsOnRead(leaf('LD/GGIO2.AnOut1.SBO', 'ST')), false);
  assert.match(read('features/tools/explorer/NodeDetail.tsx'), /bulkReadable\(attributes\)\.slice\(0, 200\)/);
  assert.match(read('features/tools/explorer/ObjectTable.tsx'), /disabled=\{sbo \|\| watched\.has\(key\)\}/);
  assert.match(read('features/tools/explorer/ExplorerTool.tsx'), /items\.filter\(item => !selectsOnRead\(item\)\)/);
});

test('A3: "report created" folds into one Activity row per block and minute; other events stay rows', () => {
  const frames = fakeFrames();
  rememberIedNames([{ id: 'TR1', name: 'Transformer 1' }, { id: 'TR2', name: 'Transformer 2' }]);
  const log = createEventLogStore(createFrameBatch(frames.clock), { describe: eventDetails, fold: reportFoldKey });
  const report = (id: number, ied: string, timestamp: number) =>
    ({ id, kind: 'report_control', timestamp, ied_id: ied, name: 'urcbA', native_kind: 9, parameter: '', peer: '' });
  for (let id = 1; id <= 600; id++) log.add(report(id, id % 2 ? 'TR1' : 'TR2', 1200 + id * 0.05));
  log.add({ id: 601, kind: 'signal', timestamp: 1231 });
  frames.frame();
  assert.equal(log.size(), 3, 'two folded rows (TR1, TR2) and the signal, not 500 report rows');
  assert.equal(log.count('report_control'), 600, 'every event is still counted');
  assert.equal(log.details(log.at(1)!), 'Transformer 2 urcbA / 300 reports created', 'names the IED');
  log.add(report(602, 'TR1', 1260));  // a new minute starts a new row
  assert.equal(log.size(), 4);
  assert.equal(log.details(log.at(0)!), 'Transformer 1 urcbA / report created');
  const enabled = { id: 9, kind: 'report_control', timestamp: 1, name: 'brcbB', native_kind: 4, peer: '127.0.0.1:5000' };
  assert.equal(eventDetails(enabled), 'brcbB / enabled by 127.0.0.1:5000');
  assert.equal(reportFoldKey(enabled), null);
  assert.equal(eventDetails({ ...enabled, native_kind: 1, parameter: 'TrgOps', error: 0 }), 'brcbB / TrgOps changed by 127.0.0.1:5000');
});

test('A2: "Add a sample IED" picks the first free SampleN, also after a delete', () => {
  assert.equal(freeSampleName([]), 'Sample1');
  const ieds = ['Feeder01', 'Sample1', 'Sample3', 'sample2'].map(name => ({ name }));
  assert.equal(freeSampleName(ieds), 'Sample4');
  const many = Array.from({ length: 10 }, (_, index) => ({ name: `IED${index}` })).concat({ name: 'Sample12' }, { name: 'Sample11' });
  assert.notEqual(freeSampleName(many), 'Sample12');
  assert.match(read('features/devices/DevicesPage.tsx'), /addTemplate\(freeSampleName\(project\.ieds\)\)/);
});

test('A10: the detail pane never shows an IED the list filter hides', () => {
  assert.equal(shownSelection('feeder', ['a', 'b']), 'a');
  assert.equal(shownSelection('b', ['a', 'b']), 'b');
  assert.equal(shownSelection('feeder', []), 'feeder', 'nothing shown: keep the choice');
  assert.equal(shownSelection('feeder', ['a'], 'feeder'), 'feeder', 'asked for from elsewhere: keep it');
});

test('A5: a menu closes on resize without asking whether it "contains" the window', () => {
  const throwing = () => { throw new TypeError("parameter 1 is not of type 'Node'"); };
  assert.equal(closesOnAway('resize', {}, throwing), true);
  assert.equal(closesOnAway('scroll', {}, throwing), true, 'a non-element target closes and never throws');
});

test('A8: a cycle time under 0.1 s is shown under the field and not sent', () => {
  assert.equal(periodProblem('sine', 0), PERIOD_PROBLEM);
  assert.equal(periodProblem('sine', -5), PERIOD_PROBLEM);
  assert.equal(periodProblem('ramp', Number.NaN), PERIOD_PROBLEM);
  assert.equal(periodProblem('sine', 0.1), '');
  assert.equal(periodProblem('csv', 0), '');
  const editor = read('features/test/SourceEditor.tsx');
  assert.match(editor, /disabled=\{!!problem\}/);
  assert.match(editor, /aria-invalid=\{!!problem\}/);
});

test('A6: value lists, the Data points search and the import button keep their whole text', () => {
  assert.match(read('styles/workspace.css'), /\.signal-select \{ min-width: max-content; \}/);
  assert.match(read('components/SignalValueEditor.tsx'), /className="signal-select"/);
  const css = read('features/devices/devices.css');
  assert.match(css, /\.devices-list-head > \.primary \{[^}]*white-space: normal/);
  assert.match(css, /\.device-detail \.filter-bar \.filter-search \{[^}]*min-width: 190px/);
});

test('A7: buttons greyed out while running say why', () => {
  const page = read('features/devices/DevicesPage.tsx');
  assert.match(page, /title=\{locked \? DEVICES_COPY\.importLocked : undefined\}/);
  assert.match(page, /title=\{locked \? DEVICES_COPY\.sampleLocked : undefined\}/);
  assert.match(read('features/devices/DeviceDetail.tsx'), /title: locked \? DEVICES_COPY\.roleLocked : undefined/);
  assert.match(DEVICES_COPY.importLocked, /^Stop the run/);
});

test('A12, A9, A4: plain words and no contradicting answers', () => {
  assert.equal(countOf(1, 'IED'), '1 IED');
  assert.match(read('features/ieds/ImportSelection.tsx'), /countOf\(preview\.ieds\.length, 'IED'\)/);
  assert.match(read('features/devices/DeviceOverview.tsx'), /DEVICES_COPY\.noAddress/);
  assert.equal(noneFound('127.50.0.1-40'), FIND_COPY.noneThisPc);
  assert.match(FIND_COPY.noneThisPc, /Press Run/);
  assert.equal(noneFound('192.168.1.0/24'), FIND_COPY.none);
  assert.match(friendlyImportError('Missing DataTypeTemplates'), /no data type section/);
  assert.equal(friendlyImportError('Something else'), 'Something else');
  assert.match(read('features/tools/explorer/CommandPanel.tsx'), /setNotice\(COMMAND_COPY\.expired\); setAnswer\(null\);/);
});

test('B5: the connection follows real answers: a failed poll says the values may be old, a good one clears it', () => {
  const health = createConnectionHealth();
  let changes = 0;
  health.subscribe(() => { changes += 1; });
  health.failed();
  assert.equal(health.state(), 'connecting', 'nothing reached yet: still connecting');
  health.answered(200);
  assert.equal(health.state(), 'connected');
  health.answered(409);
  assert.equal(health.state(), 'connected', 'a refused action is still an answer');
  health.failed();
  assert.equal(health.state(), 'lost');
  health.answered(200);
  assert.equal(health.state(), 'connected');
  health.answered(401);
  assert.equal(health.state(), 'lost', 'session gone: nothing new will arrive');
  assert.equal(changes, 4);
  assert.match(read('api/client.ts'), /connectionHealth\.failed\(\); throw reason;/);
  assert.match(read('app/App.tsx'), /connection=\{connection\}/);
});
