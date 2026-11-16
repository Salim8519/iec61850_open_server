import assert from 'node:assert/strict';
import test from 'node:test';

// ADR-0139: with auto save off the top bar says whether changes are waiting for Save.
test('Top bar words follow the auto save switch', async () => {
  const { saveWords, AUTOSAVE_COPY } = await import('../../frontend/src/copy/projectFile.ts');
  const on = { on: true, unsaved: false };
  assert.equal(saveWords('All changes saved', on).text, 'All changes saved');
  assert.equal(saveWords('Saving...', on).text, 'Saving...');
  assert.equal(saveWords('All changes saved', { on: false, unsaved: false }).text, AUTOSAVE_COPY.off);
  assert.equal(saveWords('All changes saved', { on: false, unsaved: true }).text, AUTOSAVE_COPY.unsaved);
  // The drawing is still on its way to the engine: that is not saved either.
  assert.equal(saveWords('Saving...', { on: false, unsaved: false }).text, AUTOSAVE_COPY.unsaved);
  // A failed diagram send keeps its own words in both modes.
  assert.equal(saveWords('Not saved', { on: false, unsaved: true }).text, 'Not saved');
  assert.ok(saveWords('x', { on: false, unsaved: true }).tip!.includes('Ctrl+S'));
  assert.ok(AUTOSAVE_COPY.unsaved.length <= 'All changes saved'.length + 2, 'fits where "All changes saved" fits at 1366 px');
});
