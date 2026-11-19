// Fix plan A16: overlapping actions keep the buttons disabled until the last one ends.
import assert from 'node:assert/strict';
import test from 'node:test';
import { createBusyCount } from '../../frontend/src/state/busyCount.ts';

function screen() {
  const shown = { busy: false, working: '', error: '' };
  const view = { busy: (on: boolean) => { shown.busy = on; }, working: (label: string) => { shown.working = label; },
    error: (message: string) => { shown.error = message; } };
  return { shown, view };
}
/** A task the test finishes by hand. */
function handTask() {
  let finish: (failure?: Error) => void = () => {};
  const task = () => new Promise<void>((resolve, reject) => { finish = failure => (failure ? reject(failure) : resolve()); });
  return { task, finish: (failure?: Error) => finish(failure) };
}
const tick = () => new Promise<void>(resolve => setImmediate(resolve));

test('two overlapping actions: busy stays true until both end', async () => {
  const { shown, view } = screen();
  const run = createBusyCount(view);
  const first = handTask();
  const second = handTask();
  const a = run(first.task, 'Starting the IEDs…');
  const b = run(second.task, 'Adding a sample IED…');
  first.finish();
  await a;
  assert.equal(shown.busy, true, 'the second action still runs');
  assert.equal(shown.working, 'Adding a sample IED…');
  second.finish();
  await b;
  assert.equal(shown.busy, false);
  assert.equal(shown.working, '');
});

test('the first error of overlapping actions is kept; a new action later clears it', async () => {
  const { shown, view } = screen();
  const run = createBusyCount(view);
  const first = handTask();
  const second = handTask();
  const a = run(first.task, 'one');
  const b = run(second.task, 'two');
  first.finish(new Error('first failure'));
  await a;
  second.finish(new Error('second failure'));
  await b;
  assert.equal(shown.error, 'first failure');
  const third = handTask();
  const c = run(third.task, 'three');
  await tick();
  assert.equal(shown.error, '', 'a fresh action starts with no old error');
  third.finish();
  await c;
});
