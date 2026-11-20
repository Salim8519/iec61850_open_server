import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// Change 0224 (screen trial U1, docs/testing/speed-trials-plan-2026-11-20.md): wiring that the
// browser trials measured, kept as source checks because these are React hooks. U3 (lazy pages) was
// dropped: the pages stay in the first load, so the stylesheet order and the look are unchanged.
const SRC = resolve(dirname(fileURLToPath(import.meta.url)), '../../frontend/src');
// Checkouts may use CRLF line ends; the patterns below use LF.
const read = (path: string) => readFileSync(join(SRC, path), 'utf8').replace(/\r\n/g, '\n');

test('U1: runtime-only actions run without reading the whole project again', () => {
  const studio = read('state/useStudio.ts');
  assert.match(studio, /work\(async \(\) => \{ await task\(\); await refresh\(\); \}, label\)/, 'perform still reads the project');
  assert.match(studio, /const command = useCallback\(\(task: \(\) => Promise<unknown>, label = 'Working…'\) => work\(task, label\), \[work\]\)/);
  assert.match(studio, /perform, command, refresh/);
  assert.match(read('app/App.tsx'), /<CommandContext\.Provider value=\{studio\.command\}>/);
  const users: Record<string, RegExp> = {
    'features/test/ValueControl.tsx': /await command\(async \(\) => \{/,  // set value, pulse, resume
    'features/test/SequenceEditor.tsx': /command\(\(\) => api\(`\/scenarios\/\$\{scenario\.id\}\/start`, 'POST'\)\)/,
    'features/diagram/ied/GooseLinkSection.tsx': /=> command\(async \(\) => \{\n    setNote\(''\);\n    await sendGoose/,
    'features/diagram/ied/IedSheet.tsx': /=> command\(async \(\) => \{\n    setNote\(''\);\n    await sendGoose/,
    'features/diagram/DiagramPage.tsx': /studio\.command\(async \(\) => \{[^\n]*\n      await api\(`\/equipment\/\$\{encodeURIComponent\(nodeId\)\}\/test`/,
  };
  for (const [file, pattern] of Object.entries(users)) {
    assert.match(read(file), pattern, `${file} sends its runtime-only action through command`);
    if (file !== 'features/diagram/DiagramPage.tsx') assert.match(read(file), /useCommand\(perform\)/, file);
  }
  // Operate is not runtime-only: the engine remembers the last position and options in the project.
  const diagram = read('features/diagram/DiagramPage.tsx');
  assert.match(diagram, /return studio\.perform\(async \(\) => \{\n      const result = await api<[^>]*>\(\n        `\/equipment\/\$\{encodeURIComponent\(nodeId\)\}\/operate`/);
  assert.match(diagram, /useTapCommands\(studio\.perform, report\)/, 'tap commands go through operate too');
  const context = read('state/runCommand.ts');
  assert.match(context, /return useContext\(CommandContext\) \?\? fallback;/, 'outside the app a page keeps its perform');
});

test('kept as before: a second tab, the stream events that read the project, and the 2 s Explorer refresh', async () => {
  const { classifyProjectChange } = await import('../../frontend/src/state/projectChange.ts');
  const { RELOADS } = await import('../../frontend/src/state/eventStream.ts');
  // Another tab's edit is new here: this tab drops its pending drawing and reads the project again.
  assert.equal(classifyProjectChange({ timestamp: 10, source: 'tab-b' }, 'tab-a', 9000, true), 'other');
  assert.equal(classifyProjectChange({ timestamp: 10, source: 'tab-a' }, 'tab-a', 9000, true), 'own');
  const studio = read('state/useStudio.ts');
  assert.match(studio, /\} else if \(change === 'other'\) \{\n            diagramSave\.drop\(\);\n            setSaveState\(SAVED\);\n            refresh\(\)\.catch\(fail\);/);
  assert.match(studio, /if \(!seen\(event\)\) refresh\(\)\.catch\(fail\);/, 'run, resync, worker and live events still read it');
  assert.deepEqual([...RELOADS].sort(), ['live_status', 'resync', 'runtime', 'worker_failed']);
  // ADR-0135: open data objects are read every 2 s while shown and the window is visible (trial U5 kept it).
  const reads = read('features/tools/explorer/useNodeReads.ts');
  assert.match(reads, /export const REFRESH_MS = 2000;/);
  assert.match(reads, /const timer = window\.setInterval\(\(\) => \{\n      if \(document\.visibilityState === 'visible'\) latest\.current\(\);\n    \}, REFRESH_MS\);/);
});
