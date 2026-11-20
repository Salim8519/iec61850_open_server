// Fix plan A1 and A2: a project read or a failed save never throws a drawing edit away.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import type { Diagram, Project } from '../../frontend/src/api/types.ts';
import { createDiagramSave, keepLocalEdit, NOT_SAVED, SAVED } from '../../frontend/src/state/diagramSave.ts';

const node = (id: string, x: number) => ({ id, type: 'equipment', position: { x, y: 0 }, data: {} }) as unknown as Diagram['nodes'][number];
const drawing = (x: number): Diagram => ({ nodes: [node('cb1', x)], edges: [] });
const project = (x: number, revision = 1): Project =>
  ({ id: 'p', name: 'P', session_id: 's1', revision, ieds: [], diagram: drawing(x), runtime: 'stopped' }) as unknown as Project;
const wait = (ms = 5) => new Promise<void>(resolve => setTimeout(resolve, ms));

/** One tab wired the way useStudio wires it: `read` is GET /project, `put` is PUT /project/diagram. */
function tab(server: { project: Project }, put: (body: { diagram: Diagram; revision: number }) => Promise<{ revision: number }>) {
  let current: Project | null = server.project;
  const seen = { status: SAVED, error: '' };
  const save = createDiagramSave({
    project: () => current, show: value => { current = value; }, put, tabId: 't1', delay: 1,
    status: text => { seen.status = text; }, error: message => { seen.error = message; },
    reload: async () => { await refresh(); },
  });
  async function refresh() {
    current = keepLocalEdit(structuredClone(server.project), current, save.pending());
  }
  return { save, refresh, seen, now: () => current! };
}

test('A1: a project read before the save timer fires keeps the edit, and the PUT carries it', async () => {
  const server = { project: project(10) };
  const sent: Diagram[] = [];
  const view = tab(server, async body => { sent.push(body.diagram); return { revision: body.revision + 1 }; });
  view.save.edit(drawing(99));
  await view.refresh();  // e.g. a runtime or resync event, or Ctrl+S's own refresh
  assert.equal(view.now().diagram.nodes[0].position.x, 99, 'the drawing on screen is still the edit');
  await wait();
  assert.equal(sent.length, 1);
  assert.equal(sent[0].nodes[0].position.x, 99, 'the PUT carries the edited drawing');
  assert.equal(view.seen.status, SAVED);
});

test('A1: with nothing waiting to be saved, a read shows the engine drawing', () => {
  assert.equal(keepLocalEdit(project(10, 4), project(99, 3), false).diagram.nodes[0].position.x, 10);
  const other = { ...project(10, 4), session_id: 's2' };
  assert.equal(keepLocalEdit(other, project(99, 3), true).diagram.nodes[0].position.x, 10, 'another project');
  const kept = keepLocalEdit(project(10, 2), project(99, 3), true);
  assert.deepEqual([kept.diagram.nodes[0].position.x, kept.revision], [99, 3], 'an older read keeps the base revision');
});

test('A2: a network error keeps the drawing and says Not saved; the next edit tries again', async () => {
  const server = { project: project(10) };
  let down = true;
  const sent: Diagram[] = [];
  const view = tab(server, async body => {
    if (down) throw new TypeError('Failed to fetch');
    sent.push(body.diagram); return { revision: body.revision + 1 };
  });
  view.save.edit(drawing(50));
  await wait();
  assert.equal(view.now().diagram.nodes[0].position.x, 50, 'the edit is still on screen');
  assert.equal(view.seen.status, NOT_SAVED);
  assert.equal(view.seen.error, 'Failed to fetch', 'the real reason is shown');
  await view.refresh();
  assert.equal(view.now().diagram.nodes[0].position.x, 50, 'a later read keeps the unsaved edit');
  down = false;
  view.save.edit(drawing(60));
  await wait();
  assert.equal(sent.at(-1)?.nodes[0].position.x, 60);
  assert.equal(view.seen.status, SAVED);
});

test('A2: Save after a failed save sends the drawing again', async () => {
  const server = { project: project(10) };
  let down = true;
  const sent: Diagram[] = [];
  const view = tab(server, async body => {
    if (down) throw new Error('The engine did not answer');
    sent.push(body.diagram); return { revision: body.revision + 1 };
  });
  view.save.edit(drawing(70));
  await wait();
  down = false;
  await view.save.save();
  assert.equal(sent.at(-1)?.nodes[0].position.x, 70);
  assert.equal(view.seen.status, SAVED);
});

test('A2: only a revision conflict reads the engine drawing back', async () => {
  const server = { project: project(10, 5) };
  const conflict = Object.assign(new Error('Another tab changed this project.'), { data: { code: 'conflict' } });
  const view = tab(server, async () => { throw conflict; });
  view.save.edit(drawing(80));
  await wait();
  assert.equal(view.now().diagram.nodes[0].position.x, 10, 'the other tab\'s drawing is shown');
  assert.equal(view.now().revision, 5);
  assert.equal(view.seen.status, NOT_SAVED);
  assert.equal(view.seen.error, 'Another tab changed this project.');
});

// Change 0224 (found in trial U2): Undo first saves what is on screen. Sending an unchanged drawing
// again made the engine record an empty step, so the first Undo after a drag did nothing visible.
test('a save with nothing waiting sends nothing, but still waits for a save on its way', async () => {
  const server = { project: project(10) };
  const sent: Diagram[] = [];
  let release = () => {};
  const view = tab(server, body => new Promise(resolve => {
    sent.push(body.diagram); release = () => resolve({ revision: body.revision + 1 });
  }));
  await view.save.save();
  assert.equal(sent.length, 0, 'nothing edited: no PUT');
  view.save.edit(drawing(60));
  await wait();
  assert.equal(sent.length, 1, 'the edit is on its way');
  let settled = false;
  const prepare = view.save.save().then(() => { settled = true; });  // e.g. Undo right after the drag
  await wait();
  assert.equal(settled, false, 'Undo waits for the drag to reach the engine');
  release();
  await prepare;
  assert.equal(sent.length, 1, 'and does not send the same drawing again');
  await view.save.save();
  assert.equal(sent.length, 1);
});

test('useStudio reads the project through keepLocalEdit and saves through createDiagramSave', () => {
  const hook = readFileSync(new URL('../../frontend/src/state/useStudio.ts', import.meta.url), 'utf8');
  assert.match(hook, /apply\(keepLocalEdit\(read, current\.current, diagramSave\.pending\(\)\)\)/);
  assert.match(hook, /createDiagramSave\(/);
  assert.doesNotMatch(hook, /catch \(reason\) \{\s*saveGeneration\.current \+= 1;\s*await refresh\(\);/);
});
