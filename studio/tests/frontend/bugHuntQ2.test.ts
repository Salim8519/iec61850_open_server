// Lane Q2 bug hunt (2026-10-20): defects found by testing every function on the Go and Python engines.
import assert from 'node:assert/strict';
import test from 'node:test';
import { ROLE_HINTS, ROLE_LOCKED, roleTitle } from '../../frontend/src/copy/master.ts';

test('a greyed-out role button says why it cannot be picked, not only what it does', () => {
  for (const role of ['gateway', 'master', 'converter'] as const) {
    assert.equal(roleTitle(role, false), ROLE_HINTS[role]);
    assert.equal(roleTitle(role, true), `${ROLE_HINTS[role]} ${ROLE_LOCKED}`);
  }
  assert.match(ROLE_LOCKED, /^Stop or disconnect/);
});
