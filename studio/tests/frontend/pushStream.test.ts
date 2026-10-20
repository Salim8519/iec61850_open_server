import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { createTopicStates, fullText } from '../../frontend/src/api/pushStream.ts';
import { deltaFrame, fullFrame, keyedText, type Keyed } from './pushFakes.ts';

const sha = (text: string) => createHash('sha256').update(text, 'utf8').digest('hex');
type Step = { frame: string; sha256?: string };
const recorded = JSON.parse(readFileSync(new URL('./fixtures/push/push-frames.json', import.meta.url), 'utf8')) as Step[];

test('push stream: frames recorded from the Go hub rebuild the exact GET text every time', () => {
  const states = createTopicStates();
  ['log', 'pts', 'health', 'err'].forEach(t => states.track(t));
  let deltas = 0;
  let statuses = 0;
  for (const [index, step] of recorded.entries()) {
    const outcome = states.apply(step.frame);
    if (index === 0) {
      assert.deepEqual(outcome, { kind: 'hello', epoch: 'boot-1', client: 'tab1' });
      continue;
    }
    if (/"base":/.test(step.frame.slice(0, 40))) deltas += 1;
    if (outcome.kind === 'status') {
      statuses += 1;
      assert.equal(outcome.status, 500);
      assert.equal(sha(states.text(outcome.t) as string), step.sha256, 'the error body is held as sent');
      continue;
    }
    assert.equal(outcome.kind, 'text', `frame ${index}: ${step.frame.slice(0, 80)}`);
    if (outcome.kind === 'text') assert.equal(sha(outcome.text), step.sha256, `frame ${index} rebuilds the GET text`);
  }
  assert.equal(statuses, 1);
  assert.ok(deltas >= 300, `most changes travel as deltas (${deltas} of ${recorded.length})`);
});

test('push stream: the full body is cut out of the frame, never re-encoded', () => {
  const body = '{"a":1.0,"b":"\\u00e9 \\"q\\" </x>","c":[1,2.50],"z":{"full":0}}';
  const frame = `{"t":"health","rev":4,"full":${body}}`;
  assert.equal(fullText(frame), body);
  const states = createTopicStates();
  states.track('health');
  assert.deepEqual(states.apply(frame), { kind: 'text', t: 'health', text: body });
  assert.notEqual(JSON.stringify(JSON.parse(body)), body, 'JSON.stringify would have changed the text');
});

const table = (rows: [string, string][], tail = ']}'): Keyed =>
  ({ head: '{"points":[', keys: rows.map(([key]) => key), rows: new Map(rows), tail });

test('push stream: base == held rev applies; an old rev is ignored; a gap asks for resync once', () => {
  const states = createTopicStates();
  states.track('pts');
  const one = table([['a', '{"id":"a","v":1.0}'], ['b', '{"id":"b","v":2}']]);
  const two = table([['a', '{"id":"a","v":1.5}'], ['b', '{"id":"b","v":2}'], ['c', '{"id":"c","v":3}']]);
  const three = table([['b', '{"id":"b","v":2}'], ['c', '{"id":"c","v":4}']], '],"n":3}');
  assert.equal(states.apply(deltaFrame('pts', 2, 1, one, two) as string).kind, 'skip', 'nothing held yet');
  assert.deepEqual(states.apply(fullFrame('pts', 1, one)), { kind: 'text', t: 'pts', text: keyedText(one) });
  assert.deepEqual(states.apply(deltaFrame('pts', 2, 1, one, two) as string), { kind: 'text', t: 'pts', text: keyedText(two) });
  assert.equal(states.apply(deltaFrame('pts', 2, 1, one, two) as string).kind, 'skip', 'a repeat is ignored');
  assert.deepEqual(states.apply(deltaFrame('pts', 3, 2, two, three) as string), { kind: 'text', t: 'pts', text: keyedText(three) });
  assert.equal(states.rev('pts'), 3);
  // rev 4 was lost: rev 5 (base 4) cannot be applied.
  const four = table([['b', '{"id":"b","v":2}'], ['c', '{"id":"c","v":5}']], '],"n":3}');
  assert.deepEqual(states.apply(deltaFrame('pts', 5, 4, three, four) as string), { kind: 'resync', t: 'pts' });
  assert.equal(states.apply(deltaFrame('pts', 6, 5, four, three) as string).kind, 'skip', 'no second resync before the full frame');
  assert.deepEqual(states.apply(fullFrame('pts', 6, two)), { kind: 'text', t: 'pts', text: keyedText(two) });
  assert.equal(states.apply(deltaFrame('pts', 7, 6, two, three) as string).kind, 'text');
});

test('push stream: a full frame is always applied, also with a lower rev (restart, new params)', () => {
  const states = createTopicStates();
  states.track('pts');
  const one = table([['a', '{"id":"a"}']]);
  const two = table([['b', '{"id":"b"}']]);
  states.apply(fullFrame('pts', 40, one));
  assert.deepEqual(states.apply(fullFrame('pts', 1, two)), { kind: 'text', t: 'pts', text: keyedText(two) });
  assert.equal(states.rev('pts'), 1);
  assert.equal(states.apply(deltaFrame('pts', 41, 40, one, two) as string).kind, 'resync', 'base 40 is not held any more');
});

test('push stream: an empty list, a status answer and a whole topic', () => {
  const states = createTopicStates();
  states.track('log');
  states.track('activity');
  const empty = table([], '],"last":0}');
  const filled = table([['1', '{"seq":1}'], ['2', '{"seq":2}']], '],"last":2}');
  assert.equal(states.apply(fullFrame('log', 1, empty)).kind, 'text');
  assert.deepEqual(states.apply(deltaFrame('log', 2, 1, empty, filled) as string), { kind: 'text', t: 'log', text: keyedText(filled) });
  assert.deepEqual(states.apply(deltaFrame('log', 3, 2, filled, empty) as string), { kind: 'text', t: 'log', text: keyedText(empty) });
  assert.deepEqual(states.apply(fullFrame('log', 4, '{"error":"Gateway stopped"}', 500)), { kind: 'status', t: 'log', status: 500 });
  assert.equal(states.apply(deltaFrame('log', 5, 4, empty, filled) as string).kind, 'resync', 'no rows to apply a delta to');
  assert.deepEqual(states.apply(fullFrame('activity', 9, '{"items":[]}')), { kind: 'text', t: 'activity', text: '{"items":[]}' });
  assert.equal(states.apply('not json').kind, 'bad');
  assert.equal(states.apply(fullFrame('other', 1, '{}')).kind, 'skip', 'a topic this tab does not hold');
});

test('push stream: a new epoch drops every held text; a gate drops old-parameter frames', () => {
  const states = createTopicStates();
  states.track('pts');
  const one = table([['a', '{"id":"a"}']]);
  const two = table([['a', '{"id":"a","v":2}']]);
  states.apply(fullFrame('pts', 3, one));
  states.resetAll();
  assert.equal(states.text('pts'), null);
  assert.equal(states.apply(deltaFrame('pts', 4, 3, one, two) as string).kind, 'skip', 'waits for the new full frame');
  // Parameters changed: PUT number 2 carries them; frames before its answer may be of the old ones.
  states.track('pts', 2);
  assert.equal(states.apply(fullFrame('pts', 7, one)).kind, 'skip');
  assert.deepEqual(states.acked(1), [], 'an older PUT does not open the gate');
  assert.deepEqual(states.acked(2), ['pts'], 'a full frame was dropped and none held: resync');
  assert.equal(states.apply(fullFrame('pts', 1, two)).kind, 'text');
  states.track('pts', 3);
  assert.deepEqual(states.acked(3), [], 'nothing dropped: the full frame is still to come');
});

test('push stream: 1000 random edits with lost frames always end on the GET text (mirror_test port)', () => {
  let seed = 61850;
  const random = (n: number) => { seed = (seed * 1103515245 + 12345) % 2147483648; return seed % n; };
  let server: Keyed = table([], '],"last":0}');
  let rev = 1;
  const states = createTopicStates();
  states.track('log');
  states.apply(fullFrame('log', rev, server));
  let next = 1;
  let resyncs = 0;
  let applied = 0;
  for (let round = 0; round < 1000; round += 1) {
    const keys = server.keys.slice();
    const rows = new Map(server.rows);
    for (let count = random(3); count > 0; count -= 1, next += 1) {
      keys.push(String(next));
      rows.set(String(next), `{"seq":${next},"text":"line ${next} \\u00e9 \\"q\\"","value":1.0}`);
    }
    if (keys.length && random(3) === 0) { const key = keys[random(keys.length)]; rows.set(key, `${rows.get(key)?.slice(0, -1)},"ok":true}`); }
    if (keys.length > 3 && random(4) === 0) rows.delete(keys.splice(random(keys.length), 1)[0]);
    if (keys.length > 20) keys.splice(0, keys.length - 20).forEach(key => rows.delete(key));
    if (keys.length > 2 && random(25) === 0) [keys[0], keys[1]] = [keys[1], keys[0]];
    const after: Keyed = { head: '{"entries":[', keys, rows, tail: `],"last":${next - 1}}` };
    const delta = deltaFrame('log', rev + 1, rev, server, after);
    rev += 1;
    server = after;
    if (random(10) === 0) continue; // this frame is lost (collapsed or dropped)
    const outcome = states.apply(delta ?? fullFrame('log', rev, server));
    if (outcome.kind === 'resync') {
      resyncs += 1;
      assert.equal(states.apply(fullFrame('log', rev, server)).kind, 'text', 'the server answers a resync with a full frame');
    } else if (outcome.kind === 'text') applied += 1;
    assert.equal(states.text('log'), keyedText(server), `round ${round}`);
  }
  assert.ok(resyncs > 50 && applied > 600, `${applied} applied, ${resyncs} resyncs`);
});
