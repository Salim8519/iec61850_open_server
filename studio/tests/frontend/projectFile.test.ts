// ADR-0096: project files are .scadabench; an old .subsim file is refused in plain words.
import assert from 'node:assert/strict';
import test from 'node:test';
import {
  fileKind, OLD_PROJECT_MESSAGE, OPEN_FILE_ACCEPT, PROJECT_EXTENSION,
} from '../../frontend/src/state/projectFile.ts';

test('a .scadabench file is a project, whatever its letter case', () => {
  for (const name of ['station.scadabench', 'STATION.ScadaBench', 'my station v2.scadabench']) {
    assert.equal(fileKind(name), 'project', name);
  }
});

test('a .subsim file is an old project that is refused, not read as SCL', () => {
  for (const name of ['station.subsim', 'STATION.SubSim']) assert.equal(fileKind(name), 'old-project', name);
});

test('SCL files and anything else are imported as before', () => {
  for (const name of ['a.cid', 'b.icd', 'c.scd', 'd.iid', 'scadabench', 'notes.txt', 'a.scadabench.txt']) {
    assert.equal(fileKind(name), 'scl', name);
  }
});

test('the refusal is plain English and names both extensions', () => {
  assert.equal(OLD_PROJECT_MESSAGE,
    'This is an old project file (.subsim). ScadaBench now uses .scadabench project files.');
});

test('the file chooser offers the new extension and lists the old one only so it can explain itself', () => {
  const accepted = OPEN_FILE_ACCEPT.split(',');
  assert.ok(accepted.includes(PROJECT_EXTENSION) && accepted.includes('.subsim'));
  assert.deepEqual(['.cid', '.icd', '.scd', '.iid'].filter(extension => !accepted.includes(extension)), []);
});
