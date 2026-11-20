import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { pathTopic } from '../../frontend/src/api/capabilities.ts';
import { createLiveFeed, type PushSource, type PushWatch } from '../../frontend/src/api/liveFeed.ts';
import { createTopicStates } from '../../frontend/src/api/pushStream.ts';
import { createRowParser } from '../../frontend/src/api/rowParse.ts';
import { sameData } from '../../frontend/src/ui/sameProps.ts';
import { fakeTimers, settle } from './fakeClock.ts';
import { deltaFrame, fullFrame, keyedText, type Keyed } from './pushFakes.ts';

// Change 0224 (screen trials U2, U4, U5).
const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8').replace(/\r\n/g, '\n');

function handSource(covered: (path: string) => boolean) {
  const senders = new Map<string, Omit<PushWatch, 'text'> & { text: PushWatch['text'] }>();
  const source: PushSource = { covers: covered,
    watch: (path, text, more) => { senders.set(path, { text, ...more }); return () => senders.delete(path); } };
  return { source, senders };
}

test('U2: after a project change a pushed path costs no request; a polled one is asked once, as before', async () => {
  const clock = fakeTimers();
  let asked = 0;
  const feed = createLiveFeed(async path => { asked += 1; return path === '/project/history' ? '{"can_undo":true}' : '{"on":true}'; },
    { timers: clock.timers });
  const { source, senders } = handSource(path => path === '/project/history');
  feed.setPush(source);
  const history: unknown[] = [];
  const autosave: unknown[] = [];
  feed.subscribe('/project/history', null, value => history.push(value));
  feed.subscribe('/project/autosave', null, value => autosave.push(value));
  await settle();
  assert.equal(asked, 2, 'each view reads once when it opens');
  feed.again('/project/history');  // a diagram edit moved the revision
  feed.again('/project/autosave');
  await settle();
  assert.equal(asked, 3, 'only the polled path was asked again');
  assert.equal(autosave.length, 2, 'the polled view is shown the answer again, as when it subscribed again');
  senders.get('/project/history')?.text('{"can_undo":true,"undo":"Diagram change"}');
  assert.deepEqual(history.at(-1), { can_undo: true, undo: 'Diagram change' }, 'the engine pushes the new Undo label');
  feed.again('/not/subscribed');
  await settle();
  assert.equal(asked, 3, 'asking again for a path nobody shows sends nothing');
});

test('U2: revision-keyed views subscribe once and ask again through useAskAgain', () => {
  const hooks: Record<string, RegExp[]> = {
    'state/useHistory.ts': [/liveFeed\.subscribe<HistoryStatus>\(HISTORY, null,/, /\}, \[loaded\]\);/,
      /useAskAgain\(loaded \? HISTORY : null, `\$\{project\?\.revision\}\|\$\{project\?\.runtime\}\|\$\{project\?\.session_id\}`\)/,
      /setStatus\(await api<HistoryStatus>\(`\/project\/\$\{direction\}`, 'POST'\)\)/],
    'state/useAutoSaveSwitch.ts': [/liveFeed\.subscribe<AutoSaveState>\(AUTOSAVE, null, setState/, /useAskAgain\(loaded \? AUTOSAVE : null, project\)/,
      /setState\(await setAutoSave\(!state\.on\)\)/],
    'features/devices/useModes.ts': [/\}, \[running\]\);\n  useAskAgain\(HEALTH, revision\);/, /\[badges, running\]\);/],
    'features/devices/useGooseLinks.ts': [/setMap\), \[running\]\);\n  useAskAgain\(LINK_MAP, revision\);/],
    'features/diagram/signals/useGooseMembers.ts': [/\}, \(\) => \{\}\), \[store\]\);\n  useAskAgain\(GOOSE_MEMBERS, revision\);/],
    'features/diagram/signals/useSignalLayers.ts': [/\}, \[active, running, store\]\);\n  useAskAgain\(active \? LINK_MAP : null, revision\);/],
    'features/network/GooseReach.tsx': [/setShare\), \[live\]\);\n  useAskAgain\(SHARE, revision\);/],
    'api/useLiveFeed.ts': [/\}, \[path, every\]\);\n  useAskAgain\(path, again\);/],
  };
  for (const [file, patterns] of Object.entries(hooks)) patterns.forEach(pattern => assert.match(read(file), pattern, file));
  const ask = read('api/useAskAgain.ts');
  assert.match(ask, /if \(path && before\.path === path && !Object\.is\(before\.key, key\)\) liveFeed\.again\(path\);/);
});

test('U4: a keyed answer parses only new row texts; unchanged rows keep their objects', () => {
  const parse = createRowParser();
  const rows = ['{"id":"a","value_text":"1"}', '{"id":"b","value_text":"2"}', '{"id":"c","value_text":"3"}'];
  const first = parse({ head: '{"protocol":"iec104","points":[', rows, tail: '],"n":3}' }) as { points: object[] };
  assert.deepEqual(first, JSON.parse(`{"protocol":"iec104","points":[${rows.join(',')}],"n":3}`));
  const next = [rows[0], '{"id":"b","value_text":"9"}', rows[2]];
  const second = parse({ head: '{"protocol":"iec104","points":[', rows: next, tail: '],"n":3}' }) as { points: object[] };
  assert.equal(second.points[0], first.points[0], 'unchanged row: same object');
  assert.notEqual(second.points[1], first.points[1]);
  assert.equal(second.points[2], first.points[2]);
  assert.deepEqual(second, JSON.parse(`{"protocol":"iec104","points":[${next.join(',')}],"n":3}`), 'same values as a whole parse');
  assert.equal(parse({ head: '{"a":{"points":[', rows, tail: ']}}' }), undefined, 'a nested list: the caller parses the whole text');
  assert.equal(createRowParser()({ head: '{"points": [', rows: [], tail: ']}' }) !== undefined, true, 'spaces and an empty list');
});

test('U4: push frames carry their rows; the live feed keeps every unchanged row object', () => {
  const rowsOf = (values: number[]) => new Map(values.map((value, index) => [String(index), `{"id":${index},"v":${value}}`]));
  const before: Keyed = { head: '{"points":[', keys: ['0', '1', '2'], rows: rowsOf([1, 2, 3]), tail: ']}' };
  const after: Keyed = { ...before, rows: rowsOf([1, 5, 3]) };
  const states = createTopicStates();
  states.track('gateway.points');
  const full = { ...states.apply(fullFrame('gateway.points', 1, before)), parts: states.parts('gateway.points') };
  const delta = { ...states.apply(deltaFrame('gateway.points', 2, 1, before, after) as string), parts: states.parts('gateway.points') };
  assert.ok(full.kind === 'text' && delta.kind === 'text');
  if (full.kind !== 'text' || delta.kind !== 'text') return;
  assert.equal(full.parts!.head + full.parts!.rows.join(',') + full.parts!.tail, keyedText(before));
  assert.equal(delta.parts!.head + delta.parts!.rows.join(',') + delta.parts!.tail, keyedText(after));
  assert.equal(states.parts('other'), undefined, 'an untracked topic has no parts');
  const clock = fakeTimers();
  const feed = createLiveFeed(async () => keyedText(before), { timers: clock.timers });
  const { source, senders } = handSource(path => path.startsWith('/tools/gateway/points'));
  feed.setPush(source);
  const seen: { points: object[] }[] = [];
  feed.subscribe<{ points: object[] }>('/tools/gateway/points?protocol=iec104', 1000, value => seen.push(value));
  senders.get('/tools/gateway/points?protocol=iec104')?.text(full.text, full.parts);
  senders.get('/tools/gateway/points?protocol=iec104')?.text(delta.text, delta.parts);
  assert.equal(seen.length, 2);
  assert.deepEqual(seen[1], JSON.parse(keyedText(after)));
  assert.equal(seen[1].points[0], seen[0].points[0], 'row 0 did not change: same object, its row does not render again');
  assert.notEqual(seen[1].points[1], seen[0].points[1]);
  assert.deepEqual(clock.intervals(), [], 'pushed: no timer');
});

test('U4: point rows are memoised on what they show; their handlers read the latest list', () => {
  const update = () => {};
  const row = { id: 'a' };
  assert.equal(sameData({ row, index: 1, update }, { row, index: 1, update: () => {} }), true, 'a new handler alone renders nothing');
  assert.equal(sameData({ row, index: 1 }, { row, index: 2 }), false, 'a moved row renders again');
  assert.equal(sameData({ row }, { row: { id: 'a' } }), false, 'a changed row object renders again');
  const pointRow = read('features/tools/gateway/PointRow.tsx');
  assert.match(pointRow, /export const PointRow = memo\(PointRowView, sameData\);/);
  const table = read('features/tools/gateway/PointTable.tsx');
  assert.match(table, /<PointRow row=\{row\} index=\{index\} protocol=\{protocol\} rows=\{ownRows\}/);
  assert.match(table, /togglePicked=\{\(\) => setPicked\(current => current && togglePick\(current, key\)\)\}/);
  assert.match(table, /toggleChange=\{\(\) => setChanging\(current => current === key \? null : key\)\}/);
  assert.match(table, /const now = latest\(\);  \/\/ a memoised row may call an older copy of this handler/);
  assert.match(read('features/tools/gateway/usePointList.ts'), /const now = latest\(\);\n    if \(now\) edit\(now\.map\(/);
  const master = read('features/tools/gateway/master/MasterPointTable.tsx');
  assert.match(master, /<MasterPointRow key=\{row\.key\} row=\{row\}/);
  assert.match(master, /untick\(latest\.current\.rows\.filter\(/);
  assert.match(read('features/tools/gateway/master/MasterPointRow.tsx'), /\}, sameData\);/);
});

test('U5: the lab status and the GOOSE message lists come from the push stream, not a 2-3 s poll', async () => {
  // api/gooseMessages.ts imports the fetch client, so its one-line path is checked as text and used here.
  assert.ok(read('api/gooseMessages.ts').includes(
    'export const messagesPath = (iedId: string) => `/goose/messages?ied=${encodeURIComponent(iedId)}`;'));
  const messagesPath = (iedId: string) => `/goose/messages?ied=${encodeURIComponent(iedId)}`;
  const paths = { '/network/lab': { t: 'network.lab', params: [] }, '/goose/messages': { t: 'goose.messages', params: ['ied'] } };
  assert.deepEqual(pathTopic(paths, messagesPath('ied 1')), { t: 'goose.messages', params: { ied: 'ied 1' }, key: '[["ied","ied 1"]]' });
  assert.equal(messagesPath('a/b'), '/goose/messages?ied=a%2Fb');
  assert.deepEqual(pathTopic(paths, '/network/lab')?.t, 'network.lab');
  const clock = fakeTimers();
  let asked = 0;
  const feed = createLiveFeed(async () => { asked += 1; return '{"active":false}'; }, { timers: clock.timers });
  const { source } = handSource(path => pathTopic(paths, path) !== null);
  feed.setPush(source);
  feed.subscribe('/network/lab', 3000, () => {});
  feed.subscribe(messagesPath('x'), 2000, () => {});
  await settle();
  assert.deepEqual(clock.intervals(), [], 'no 3 s or 2 s timer while the stream carries them');
  assert.equal(asked, 2, 'one read each when the view opens');
  assert.match(read('features/network/LabNetwork.tsx'), /liveFeed\.subscribe<LabStatus>\(LAB, 3000,/, 'the page keeps 3 s for polling engines');
});
