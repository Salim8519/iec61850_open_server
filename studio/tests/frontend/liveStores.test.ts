import assert from 'node:assert/strict';
import test from 'node:test';
import { createFrameBatch } from '../../frontend/src/state/frameBatch.ts';
import { createValueStore, eventValues } from '../../frontend/src/state/valueStore.ts';
import { createEventLogStore } from '../../frontend/src/state/eventLogStore.ts';
import { createKeyedStore } from '../../frontend/src/equipment/keyedStore.ts';
import { fakeFrames } from './fakeClock.ts';

test('frame batch: one frame request for many tasks, the same task runs once', () => {
  const frames = fakeFrames();
  const batch = createFrameBatch(frames.clock);
  let runs = 0;
  const task = () => { runs += 1; };
  batch.schedule(task); batch.schedule(task); batch.schedule(() => { runs += 10; });
  assert.equal(frames.waiting(), 1);
  assert.equal(runs, 0, 'nothing runs before the frame');
  frames.frame();
  assert.equal(runs, 11);
  assert.equal(batch.pending(), 0);
});

test('frame batch: work asked during a flush waits for the next frame; visibility flushes at once', () => {
  const frames = fakeFrames();
  let changed = () => {};
  const batch = createFrameBatch(frames.clock, listener => { changed = listener; });
  const order: string[] = [];
  batch.schedule(() => { order.push('a'); batch.schedule(() => order.push('b')); });
  frames.frame();
  assert.deepEqual(order, ['a']);
  changed();  // the page was hidden: no frames come, so a visibility change flushes now
  assert.deepEqual(order, ['a', 'b']);
  assert.equal(frames.waiting(), 0, 'the cancelled frame request is gone');
});

test('value store: last value wins and listeners hear once per frame, only for changed keys', () => {
  const frames = fakeFrames();
  const store = createValueStore(createFrameBatch(frames.clock));
  const heard: string[] = [];
  store.subscribe('ied:a', () => heard.push('a'));
  store.subscribe('ied:b', () => heard.push('b'));
  let any = 0;
  store.subscribeAll(() => { any += 1; });
  for (let index = 0; index < 100; index++) store.set('ied:a', index);
  assert.equal(store.get('ied:a'), undefined, 'not visible before the frame');
  frames.frame();
  assert.equal(store.get('ied:a'), 99);
  assert.deepEqual(heard, ['a'], '100 writes, one notification, b untouched');
  assert.equal(any, 1);
  store.set('ied:a', 99);
  frames.frame();
  assert.deepEqual(heard, ['a'], 'the same value is not news');
  store.clear();
  frames.frame();
  assert.equal(store.get('ied:a'), undefined);
  assert.deepEqual(heard, ['a', 'a']);
  assert.equal(store.size(), 0);
});

test('value store: signal and control events write their reference and their updates', () => {
  assert.deepEqual(eventValues({ kind: 'signal', ied_id: 'I', ref: 'LD/X.stVal', value: 1, updates: { 'LD/X.q': 0 } }),
    [['I:LD/X.stVal', 1], ['I:LD/X.q', 0]]);
  assert.deepEqual(eventValues({ kind: 'control_applied', ied_id: 'I', ref: 'LD/C.Oper', status_ref: 'LD/C.stVal', value: true }),
    [['I:LD/C.stVal', true]]);
  assert.deepEqual(eventValues({ kind: 'runtime' }), []);
});

test('event log: a 500-entry ring, newest first, time made once per second, counts per kind', () => {
  const frames = fakeFrames();
  let formatted = 0;
  const log = createEventLogStore<{ id: number; kind: string; timestamp: number }>(createFrameBatch(frames.clock), {
    formatTime: second => { formatted += 1; return `t${second}`; }, describe: event => `#${event.id}`,
  });
  let notified = 0;
  log.subscribe(() => { notified += 1; });
  for (let id = 1; id <= 1200; id++) log.add({ id, kind: id % 100 === 0 ? 'goose_conflict' : 'signal', timestamp: 1000 + Math.floor(id / 400) });
  frames.frame();
  assert.equal(notified, 1, '1200 events, one notification');
  assert.equal(log.size(), 500);
  assert.equal(log.at(0)?.id, 1200);
  assert.equal(log.at(499)?.id, 701);
  assert.equal(log.at(500), undefined);
  assert.equal(log.at(0)?.label, 'goose conflict', 'kind words made once, without underscores');
  assert.equal(log.at(1)?.label, 'signal');
  assert.equal(log.at(0)?.time, 't1003');
  assert.equal(formatted, 4, 'one clock text per second, not per event');
  assert.equal(log.count('goose_conflict'), 12, 'counts every event received, not only the kept ones');
  const entry = log.at(1)!;
  assert.equal(entry.details, undefined, 'details are worked out only when shown');
  assert.equal(log.details(entry), '#1199');
});

test('keyed store: a whole new record tells only the keys that changed', () => {
  const store = createKeyedStore<string>();
  const heard: string[] = [];
  for (const key of ['a', 'b', 'c']) store.subscribe(key, () => heard.push(key));
  assert.equal(store.replace({ a: 'test', b: 'blocked' }), 2);
  assert.equal(store.replace({ a: 'test', b: 'blocked' }), 0, 'the same answer every 3 s re-renders nothing');
  assert.equal(store.replace({ a: 'test', c: 'off' }), 2);
  assert.deepEqual(heard, ['a', 'b', 'b', 'c']);
  assert.equal(store.get('b'), undefined);
});
