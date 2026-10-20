import assert from 'node:assert/strict';
import test from 'node:test';
import { createLiveFeed, type PushSource } from '../../frontend/src/api/liveFeed.ts';
import { fakeTimers, settle } from './fakeClock.ts';

/** A push source the test drives by hand (speed lane F4 wiring of the live feed). */
function handSource() {
  const senders = new Map<string, { text: (text: string) => void; ask?: () => void; release?: () => void }>();
  const source: PushSource = {
    covers: path => path.startsWith('/health') || path === '/sv',
    watch: (path, text, more) => { senders.set(path, { text, ...more }); return () => senders.delete(path); },
  };
  return { source, senders };
}

test('live feed + push: the pushed text equals the poll answer without Flask\'s newline', async () => {
  const clock = fakeTimers();
  const feed = createLiveFeed(async () => '{"v":1}\n', { timers: clock.timers });
  const seen: unknown[] = [];
  const { source, senders } = handSource();
  feed.setPush(source);
  feed.subscribe('/sv', 2000, value => seen.push(value));
  await settle();
  senders.get('/sv')?.text('{"v":1}');
  assert.equal(seen.length, 1, 'a frame with the same body is not shown again');
  senders.get('/sv')?.text('{"v":2}');
  assert.deepEqual(seen, [{ v: 1 }, { v: 2 }]);
});

test('live feed + push: a request answered after a frame does not replace the frame', async () => {
  const clock = fakeTimers();
  let release = (_text: string) => {};
  const feed = createLiveFeed(() => new Promise<string>(resolve => { release = resolve; }), { timers: clock.timers });
  const seen: unknown[] = [];
  const { source, senders } = handSource();
  feed.setPush(source);
  feed.subscribe('/sv', 2000, value => seen.push(value));
  senders.get('/sv')?.text('{"v":2}');
  release('{"v":1}\n'); // read before the change the frame carries
  await settle();
  assert.deepEqual(seen, [{ v: 2 }], 'the older HTTP answer is not shown over the pushed one');
});

test('live feed + push: release gives the path back to polling; ask reads once over HTTP', async () => {
  const clock = fakeTimers();
  let asked = 0;
  const feed = createLiveFeed(async () => { asked += 1; return '{}'; }, { timers: clock.timers });
  const { source, senders } = handSource();
  feed.setPush(source);
  feed.subscribe('/health?x=1', 3000, () => {});
  await settle();
  assert.deepEqual(clock.intervals(), []);
  senders.get('/health?x=1')?.ask?.();
  await settle();
  assert.equal(asked, 2);
  senders.get('/health?x=1')?.release?.();
  assert.deepEqual(feed.pushed(), []);
  assert.deepEqual(clock.intervals(), [3000]);
});
