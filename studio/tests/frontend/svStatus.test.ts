import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { SV_PROBLEMS, SV_STATES, svStatus } from '../../frontend/src/copy/sv.ts';
import { TONES } from '../../frontend/src/copy/status.ts';

const domain = readFileSync(new URL('../../backend/studio_app/domain/sv_supervision.py', import.meta.url), 'utf8');
const native = ['sv_traffic.cpp', 'sv_traffic.hpp']
  .map(name => readFileSync(new URL(`../../native/live/${name}`, import.meta.url), 'utf8')).join('\n');

test('every sv state has text icon and hint', () => {
  const problems = domain.slice(domain.indexOf('PROBLEM_ORDER'), domain.indexOf(')', domain.indexOf('PROBLEM_ORDER')));
  const names = [...problems.matchAll(/"([a-z_]+)"/g)].map(match => match[1]);
  assert.equal(names.length, 7);
  const states = new Set(['stopped', 'problem', ...[...native.matchAll(/"(waiting|receiving|timeout)"/g)].map(match => match[1])]);
  assert.deepEqual([...states].sort(), ['problem', 'receiving', 'stopped', 'timeout', 'waiting']);
  for (const state of states) {
    const status = svStatus(state);
    assert.equal(status, SV_STATES[state], state);
    assert.ok(status.label && status.hint.length > 20 && TONES[status.tone].icon, state);
  }
  for (const name of names) {
    const status = svStatus('problem', [name]);
    assert.equal(status, SV_PROBLEMS[name], `problem ${name} needs its own wording`);
    assert.ok(status.hint.length > 30 && TONES[status.tone].icon, name);
  }
  const labels = Object.values(SV_PROBLEMS).map(item => item.label);
  assert.equal(new Set(labels).size, labels.length);
});
