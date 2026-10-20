import assert from 'node:assert/strict';
import test from 'node:test';
import { createLiveFeed } from '../../frontend/src/api/liveFeed.ts';
import { fakeTimers, fakeVisibility, settle } from './fakeClock.ts';

function backend(answers: Record<string, string>) {
  const asked: string[] = [];
  return { asked, fetchText: async (path: string) => { asked.push(path); return answers[path]; }, answers };
}

test('live feed: one request per path for every view, at the fastest interval asked for', async () => {
  const server = backend({ '/activity': '{"n":1}' });
  const clock = fakeTimers();
  const feed = createLiveFeed(server.fetchText, { timers: clock.timers });
  const a: unknown[] = [];
  const b: unknown[] = [];
  const stopA = feed.subscribe('/activity', 2000, value => a.push(value));
  const stopB = feed.subscribe('/activity', 5000, value => b.push(value));
  await settle();
  assert.equal(server.asked.length, 1, 'the second view shares the first request');
  assert.deepEqual(clock.intervals(), [2000], 'same interval as before, one timer');
  assert.deepEqual(a, [{ n: 1 }]);
  assert.deepEqual(b, [{ n: 1 }]);
  stopA();
  assert.deepEqual(clock.intervals(), [5000]);
  stopB();
  assert.deepEqual(clock.intervals(), []);
  assert.deepEqual(feed.active(), []);
});

test('live feed: an unchanged answer calls nobody; a changed one does', async () => {
  const server = backend({ '/health': '{"ok":true}' });
  const clock = fakeTimers();
  const feed = createLiveFeed(server.fetchText, { timers: clock.timers });
  const seen: unknown[] = [];
  feed.subscribe('/health', 3000, value => seen.push(value));
  await settle();
  clock.tick(); await settle();
  clock.tick(); await settle();
  assert.equal(server.asked.length, 3);
  assert.equal(seen.length, 1, 'three answers, the same text: one call');
  server.answers['/health'] = '{"ok":false}';
  clock.tick(); await settle();
  assert.deepEqual(seen, [{ ok: true }, { ok: false }]);
  feed.refresh('/health'); await settle();
  assert.equal(seen.length, 3, 'an explicit refresh is always shown');
});

test('live feed: no overlap while an answer is pending', async () => {
  let release = () => {};
  let calls = 0;
  const clock = fakeTimers();
  const feed = createLiveFeed(() => { calls += 1; return new Promise<string>(resolve => { release = () => resolve('{}'); }); },
    { timers: clock.timers });
  feed.subscribe('/sv', 1000, () => {});
  clock.tick(); clock.tick(); clock.tick();
  assert.equal(calls, 1, 'a slow backend gets no pile-up');
  release(); await settle();
  assert.equal(calls, 2, 'asked once more, because ticks came while busy');
});

test('live feed: paused while the page is hidden, asks once when it shows again', async () => {
  const server = backend({ '/activity': '{}', '/app/update': '{}' });
  const clock = fakeTimers();
  const page = fakeVisibility();
  const feed = createLiveFeed(server.fetchText, { timers: clock.timers, visibility: page.visibility });
  feed.subscribe('/activity', 2000, () => {});
  feed.subscribe('/app/update', null, () => {});
  await settle();
  assert.equal(server.asked.length, 2);
  page.set(true);
  assert.deepEqual(clock.intervals(), [], 'hidden: no timers');
  clock.tick(); await settle();
  assert.equal(server.asked.length, 2);
  page.set(false);
  await settle();
  assert.deepEqual(clock.intervals(), [2000]);
  assert.deepEqual(server.asked.slice(2), ['/activity'], 'visible again: polled paths ask at once, one-shot paths do not');
});

test('live feed: after an error the next answer is shown even when it equals the old one', async () => {
  let fail = false;
  const clock = fakeTimers();
  const feed = createLiveFeed(async () => { if (fail) throw new Error('busy'); return '{"a":1}'; }, { timers: clock.timers });
  const seen: unknown[] = [];
  const errors: string[] = [];
  feed.subscribe('/goose', 2000, value => seen.push(value), error => errors.push(error.message));
  await settle();
  fail = true; clock.tick(); await settle();
  fail = false; clock.tick(); await settle();
  assert.deepEqual(errors, ['busy']);
  assert.equal(seen.length, 2, 'the view clears its error with the answer');
});

test('live feed: a push source replaces the timer of the paths it covers', async () => {
  const server = backend({ '/health': '{"v":1}', '/activity': '{}' });
  const clock = fakeTimers();
  const feed = createLiveFeed(server.fetchText, { timers: clock.timers });
  const seen: unknown[] = [];
  feed.subscribe('/health', 3000, value => seen.push(value));
  feed.subscribe('/activity', 2000, () => {});
  await settle();
  const senders = new Map<string, (text: string) => void>();
  feed.setPush({ covers: path => path === '/health', watch: (path, onText) => { senders.set(path, onText); return () => senders.delete(path); } });
  assert.deepEqual(clock.intervals(), [2000], 'only /activity is still polled');
  assert.deepEqual(feed.pushed(), ['/health']);
  senders.get('/health')!('{"v":1}');
  senders.get('/health')!('{"v":2}');
  assert.deepEqual(seen, [{ v: 1 }, { v: 2 }], 'a pushed frame equal to the last answer is ignored');
  feed.setPush(null);
  assert.deepEqual(clock.intervals().sort(), [2000, 3000], 'without push the timer comes back');
  assert.equal(senders.size, 0);
});

test('live feed: an incremental task answers null when nothing is new', async () => {
  const clock = fakeTimers();
  const feed = createLiveFeed(async () => '{}', { timers: clock.timers });
  let after = 0;
  const pages: number[][] = [];
  const stop = feed.subscribeTask<{ entries: number[]; last: number }>('gateway-traffic', 500,
    async () => after >= 3 ? null : { entries: [after + 1], last: after + 1 },
    page => { after = page.last; pages.push(page.entries); });
  await settle();
  clock.tick(); await settle();
  clock.tick(); await settle();
  clock.tick(); await settle();
  clock.tick(); await settle();
  assert.deepEqual(pages, [[1], [2], [3]]);
  assert.deepEqual(clock.intervals(), [500]);
  stop();
  assert.deepEqual(clock.intervals(), []);
});
