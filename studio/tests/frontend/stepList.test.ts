// Change 0150: long waits show their steps (network address check, opening and importing files).
import assert from 'node:assert/strict';
import test from 'node:test';
import type { CheckProgress } from '../../frontend/src/api/network.ts';
import { checkSteps, ROW_STATE } from '../../frontend/src/features/network/checkSteps.ts';
import { workSteps } from '../../frontend/src/state/workSteps.ts';

const progress = (running: boolean): CheckProgress => ({ run: 2, running, asked: 30, answered: 12, rows: [],
  steps: [
    { key: 'cards', label: 'Read the network card', state: 'done', ms: 3 },
    { key: 'addresses', label: 'Check each address on the network', state: 'running', ms: 900 },
    { key: 'firewall', label: 'Read Windows Firewall rules', state: 'waiting', ms: 0 },
    { key: 'plan', label: 'Prepare the exact Windows changes', state: 'waiting', ms: 0 },
  ] });

test('before the engine reports, the four steps show with the first one running', () => {
  for (const value of [null, progress(false)]) {
    const steps = checkSteps(value);
    assert.deepEqual(steps.map(step => step.state), ['running', 'waiting', 'waiting', 'waiting']);
  }
});

test('the address step counts answers ("12 of 30 addresses")', () => {
  const step = checkSteps(progress(true))[1];
  assert.equal(step.state, 'running');
  assert.deepEqual([step.done, step.total, step.unit], [12, 30, 'addresses']);
  const one = checkSteps({ ...progress(true), asked: 1, answered: 0 })[1];
  assert.equal(one.unit, 'address');
});

test('every row state has words, so colour is never the only signal', () => {
  for (const state of ['checking', 'free', 'in_use', 'not_checked'] as const) assert.ok(ROW_STATE[state].length > 5);
});

test('opening a file ticks finished steps with their time and turns the current one', () => {
  const steps = workSteps({ title: 'Opening a.scadabench', labels: ['Save', 'Open', 'Show'], at: 1, times: [12] });
  assert.deepEqual(steps.map(step => [step.state, step.ms]), [['done', 12], ['running', undefined], ['waiting', undefined]]);
});
