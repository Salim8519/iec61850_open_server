import assert from 'node:assert/strict';
import test from 'node:test';
import { createPoller } from '../../frontend/src/api/sharedPoll.ts';

const settle = () => new Promise(resolve => setTimeout(resolve, 0));

function slowBackend() {
  const calls: { path: string; resolve: (value: unknown) => void }[] = [];
  const fetcher = (path: string) => new Promise(resolve => calls.push({ path, resolve }));
  return { calls, fetcher };
}

test('views showing the same path share one request and never overlap', async () => {
  const backend = slowBackend();
  const polls = createPoller(backend.fetcher);
  const seen: unknown[] = [];
  const stopA = polls.subscribe('/health', 3000, value => seen.push(['a', value]));
  const stopB = polls.subscribe('/health', 3000, value => seen.push(['b', value]));
  polls.refresh('/health');
  assert.equal(backend.calls.length, 1, 'a second view or a refresh waits for the answer in flight');
  backend.calls[0].resolve(1);
  await settle();
  assert.deepEqual(seen, [['a', 1], ['b', 1]]);
  assert.equal(backend.calls.length, 2, 'asked again once, because a view asked while busy');
  backend.calls[1].resolve(2);
  await settle();
  assert.equal(backend.calls.length, 2, 'no pile-up: nothing more until the timer');
  stopA(); stopB();
  assert.deepEqual(polls.active(), []);
});

test('a view that opens later gets the last answer at once', async () => {
  const backend = slowBackend();
  const polls = createPoller(backend.fetcher);
  const stopA = polls.subscribe('/goose/links', 5000, () => {});
  backend.calls[0].resolve({ links: 3 });
  await settle();
  const late: unknown[] = [];
  const stopB = polls.subscribe('/goose/links', null, value => late.push(value));
  assert.deepEqual(late, [{ links: 3 }]);
  stopA(); stopB();
});

test('errors reach only the views that listen for them', async () => {
  const polls = createPoller(() => Promise.reject(new Error('backend busy')));
  const errors: string[] = [];
  const stop = polls.subscribe('/health', null, () => assert.fail('no value'), error => errors.push(error.message));
  await settle();
  assert.deepEqual(errors, ['backend busy']);
  stop();
});

test('the timer runs at the fastest interval asked for and stops with the last view', async t => {
  t.mock.timers.enable({ apis: ['setInterval'] });
  let count = 0;
  const polls = createPoller(() => { count += 1; return Promise.resolve(count); });
  const slow = polls.subscribe('/health', 5000, () => {});
  const fast = polls.subscribe('/health', 3000, () => {});
  await settle();
  const start = count;
  t.mock.timers.tick(3000);
  await settle();
  assert.equal(count, start + 1);
  fast();
  t.mock.timers.tick(3000);
  await settle();
  assert.equal(count, start + 1, 'back to 5 s once the 3 s view closed');
  slow();
  t.mock.timers.tick(10000);
  await settle();
  assert.equal(count, start + 1, 'no requests after every view closed');
});
