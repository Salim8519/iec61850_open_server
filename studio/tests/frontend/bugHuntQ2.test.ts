// Lane Q2 bug hunt (2026-10-20): defects found by testing every function on the Go and Python engines.
import assert from 'node:assert/strict';
import test from 'node:test';
import { ROLE_HINTS, ROLE_RUNNING, roleTitle } from '../../frontend/src/copy/master.ts';

test('a greyed-out role button says why it cannot be picked, not only what it does', () => {
  for (const role of ['gateway', 'master', 'converter'] as const) {
    assert.equal(roleTitle(role), ROLE_HINTS[role]);
    // The engine's reason (ADR-0143 role_locks) is said on the button itself.
    assert.equal(roleTitle(role, 'Stop the protocol converter before switching roles'),
      `${ROLE_HINTS[role]} Stop the protocol converter before switching roles.`);
    assert.equal(roleTitle(role, undefined, true), `${ROLE_HINTS[role]} ${ROLE_RUNNING}`);
  }
});
