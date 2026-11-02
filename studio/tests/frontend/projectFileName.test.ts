import assert from 'node:assert/strict';
import test from 'node:test';
import { projectFileName } from '../../frontend/src/state/saveProjectAs.ts';

test('Save as suggests a Windows-safe file name from the project name (ADR-0135)', () => {
  assert.equal(projectFileName('North yard'), 'North yard.scadabench');
  assert.equal(projectFileName('Bay 1/2: "test"?'), 'Bay 1 2 test.scadabench');
  assert.equal(projectFileName('  trailing dots... '), 'trailing dots.scadabench');
  assert.equal(projectFileName(''), 'substation.scadabench');
});
