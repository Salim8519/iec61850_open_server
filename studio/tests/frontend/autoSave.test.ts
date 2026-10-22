// "Save always": one request in flight, the latest value wins, nothing is lost while held or failing.
import assert from 'node:assert/strict';
import test from 'node:test';
import { createAutoSaver, SaveLater, type SaveState } from '../../frontend/src/state/autoSave.ts';

/** A save function the test finishes by hand, so "in flight" is something it can look at. */
function handSave() {
  const calls: Array<{ value: number; ok: () => void; fail: (error: Error) => void }> = [];
  let parallel = 0;
  let most = 0;
  const run = (value: number) => new Promise<void>((resolve, reject) => {
    parallel += 1; most = Math.max(most, parallel);
    calls.push({ value, ok: () => { parallel -= 1; resolve(); }, fail: error => { parallel -= 1; reject(error); } });
  });
  return { run, calls, most: () => most };
}
const tick = () => new Promise<void>(resolve => setImmediate(resolve));
const phases = () => { const seen: SaveState[] = []; return { seen, on: (state: SaveState) => seen.push(state) }; };

test('the first value is sent at once, with no timer, and the status goes Saving then Saved', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.push(1);
  assert.deepEqual(save.calls.map(call => call.value), [1], 'sent in the same tick');
  assert.equal(log.seen.at(-1)?.phase, 'saving');
  save.calls[0].ok();
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'saved');
  assert.equal(saver.pending(), false);
});

test('values pushed while a request runs become one trailing save: the latest wins, never two at once', async () => {
  const save = handSave();
  const saver = createAutoSaver(save.run, () => {});
  saver.push(1);
  saver.push(2);
  saver.push(3);
  assert.equal(save.calls.length, 1, 'only one request in flight');
  save.calls[0].ok();
  await tick();
  assert.deepEqual(save.calls.map(call => call.value), [1, 3], 'the value 2 was replaced by 3');
  save.calls[1].ok();
  await tick();
  assert.equal(save.most(), 1, 'two requests were never in flight together');
  await saver.settled();
});

test('while a trailing save waits, the status stays Saving; Saved appears only after the last one', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.push(1);
  saver.push(2);
  save.calls[0].ok();
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'saving');
  save.calls[1].ok();
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'saved');
});

test('a failed save reports its words, keeps nothing, and the next push tries again', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.push(1);
  save.calls[0].fail(new Error('The port must be 1 to 65535'));
  await tick();
  assert.deepEqual(log.seen.at(-1), { phase: 'failed', message: 'The port must be 1 to 65535' });
  saver.push(2);
  assert.equal(save.calls.length, 2, 'a new value is sent again');
  save.calls[1].ok();
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'saved');
});

test('a failure that a newer value already replaced is not shown', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.push(1);
  saver.push(2);
  save.calls[0].fail(new Error('old value was wrong'));
  await tick();
  assert.ok(!log.seen.some(state => state.phase === 'failed'), 'the older failure is out of date');
  assert.equal(save.calls[1].value, 2);
  save.calls[1].ok();
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'saved');
});

test('held (the gateway runs): nothing is sent, the latest list waits and is sent on release', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.hold(true);
  saver.push(1);
  saver.push(2);
  assert.equal(save.calls.length, 0);
  assert.equal(log.seen.at(-1)?.phase, 'waiting');
  assert.equal(saver.pending(), true);
  saver.hold(false);
  assert.deepEqual(save.calls.map(call => call.value), [2], 'only the latest list is sent');
  save.calls[0].ok();
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'saved');
});

test('a save the engine refuses for now (SaveLater) keeps its value and sends it when released', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.push(1);
  save.calls[0].fail(new SaveLater());
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'waiting');
  assert.equal(save.calls.length, 1, 'no retry by itself');
  saver.hold(true); // the page now knows that the gateway runs
  saver.push(2);
  assert.equal(save.calls.length, 1, 'a newer value waits too');
  saver.hold(false);
  assert.deepEqual(save.calls.map(call => call.value), [1, 2]);
  save.calls[1].ok();
  await tick();
});

test('a refused value is replaced by a newer one that arrived while it was in flight', async () => {
  const save = handSave();
  const saver = createAutoSaver(save.run, () => {});
  saver.push(1);
  saver.push(2);
  save.calls[0].fail(new SaveLater());
  await tick();
  saver.hold(false);
  assert.deepEqual(save.calls.map(call => call.value), [1, 2]);
  save.calls[1].ok();
  await tick();
});

test('a refusal only waits for the gateway: a newer value, or retry, is a new attempt', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.push(1);
  save.calls[0].fail(new SaveLater());
  await tick();
  saver.retry();
  assert.deepEqual(save.calls.map(call => call.value), [1, 1], 'retry sends the refused value again');
  save.calls[1].fail(new SaveLater());
  await tick();
  saver.push(2);
  assert.deepEqual(save.calls.map(call => call.value), [1, 1, 2], 'a new edit is tried at once');
  save.calls[2].ok();
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'saved');
});

test('drop clears the status so Discard does not leave "Not saved yet" behind', async () => {
  const save = handSave();
  const log = phases();
  const saver = createAutoSaver(save.run, log.on);
  saver.push(1);
  save.calls[0].fail(new Error('nope'));
  await tick();
  assert.equal(log.seen.at(-1)?.phase, 'failed');
  saver.drop();
  assert.equal(log.seen.at(-1)?.phase, 'idle');
});

test('drop forgets a waiting value (Discard) but not the request already sent', async () => {
  const save = handSave();
  const saver = createAutoSaver(save.run, () => {});
  saver.hold(true);
  saver.push(1);
  saver.drop();
  saver.hold(false);
  assert.equal(save.calls.length, 0);
  assert.equal(saver.pending(), false);
});

test('settled resolves when the last request is done, also after the page was left', async () => {
  const save = handSave();
  const saver = createAutoSaver(save.run, () => {});
  saver.push(1);
  let done = false;
  const waiting = saver.settled().then(() => { done = true; });
  await tick();
  assert.equal(done, false);
  save.calls[0].ok();
  await waiting;
  assert.equal(done, true);
});
