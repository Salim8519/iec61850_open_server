import assert from 'node:assert/strict';
import test from 'node:test';
import { adoptProject, forgetHistory, NO_HISTORY, projectIdentity, recordEdit, stepHistory,
  type HistorySession } from '../../frontend/src/state/historySession.ts';
import { classifyProjectChange, READ_MARGIN_MS, seenInRead } from '../../frontend/src/state/projectChange.ts';
import type { Diagram } from '../../frontend/src/api/types.ts';

// What useStudio does with the project and the stream, in the order it does it.
const READ_AT = 1_800_000_000_000; // when the page's project read started (ms)
const THIS_TAB = 'this-tab';
const empty: Diagram = { nodes: [], edges: [] };
const breaker = { id: 'cb', type: 'equipment' as const, position: { x: 20, y: 20 },
  data: { kind: 'CB' as const, label: 'CB1', rotation: 0 } };
const drawn: Diagram = { nodes: [breaker], edges: [] };
const project = (diagram: Diagram, ieds = ['ied-1'], id = 'p1') => ({
  session_id: 's1', id, ieds: ieds.map(item => ({ id: item })), diagram,
}) as unknown as Parameters<typeof adoptProject>[1];
const secondsBefore = (ms: number) => (READ_AT - ms) / 1000;
const history = (session: HistorySession) => session.history!;

/** The reaction of useStudio's stream handler to one project_changed event. */
function onProjectChanged(session: HistorySession, event: { source?: unknown; timestamp: number }): HistorySession {
  return classifyProjectChange(event, THIS_TAB, READ_AT, true) === 'other' ? forgetHistory(session) : session;
}

test('the first edit after opening the page can be undone although the stream replayed older changes', () => {
  let session = adoptProject(NO_HISTORY, project(empty));
  // The push stream replays the bus on every connect: changes of earlier tabs and of the backend.
  for (const event of [{ source: 'earlier-tab-1', timestamp: secondsBefore(60_000) },
    { source: 'earlier-tab-2', timestamp: secondsBefore(30_000) }, { source: null, timestamp: secondsBefore(5_000) }]) {
    assert.equal(classifyProjectChange(event, THIS_TAB, READ_AT, true), 'replayed');
    session = onProjectChanged(session, event);
  }
  session = recordEdit(session, drawn);
  assert.equal(history(session).past.length, 1, 'the first edit is a step');
  session = stepHistory(session, 'undo');
  assert.equal(history(session).present.nodes.length, 0);
  assert.equal(history(session).future.length, 1);
});

test('a project read after a local edit keeps the history, so Replace my drawing can be undone', () => {
  let session = adoptProject(NO_HISTORY, project(drawn));
  const replacement: Diagram = { nodes: [{ ...breaker, id: 'from-scd' }], edges: [] };
  session = recordEdit(session, replacement);
  // The save answers and the project is read again: same project, so nothing starts over.
  session = adoptProject(session, project(replacement));
  assert.equal(history(session).past.length, 1);
  assert.equal(history(stepHistory(session, 'undo')).present.nodes[0].id, 'cb');
});

test('a really new change by another tab forgets the history and the next read starts a new one', () => {
  let session = recordEdit(adoptProject(NO_HISTORY, project(empty)), drawn);
  const event = { source: 'other-tab', timestamp: (READ_AT + 2_000) / 1000 };
  assert.equal(classifyProjectChange(event, THIS_TAB, READ_AT, true), 'other');
  session = onProjectChanged(session, event);
  assert.equal(session.identity, '');
  const reread = { nodes: [{ ...breaker, id: 'from-other-tab' }], edges: [] };
  session = adoptProject(session, project(reread));
  assert.equal(history(session).past.length, 0);
  assert.equal(history(session).present.nodes[0].id, 'from-other-tab');
});

test('a local edit never starts a new history, even while the history waits for a new read', () => {
  let session = forgetHistory(adoptProject(NO_HISTORY, project(empty)));
  session = recordEdit(session, drawn);
  assert.equal(history(session).past.length, 1);
  assert.equal(history(session).present.nodes.length, 1);
  assert.equal(recordEdit(NO_HISTORY, drawn), NO_HISTORY, 'no history yet: nothing to record on');
});

test('another project, or the same one with other IEDs, starts a new history', () => {
  const session = recordEdit(adoptProject(NO_HISTORY, project(empty)), drawn);
  assert.equal(adoptProject(session, project(drawn)), session);
  assert.equal(history(adoptProject(session, project(drawn, ['ied-1', 'ied-2']))).past.length, 0);
  assert.equal(history(adoptProject(session, project(drawn, ['ied-1'], 'other-project'))).past.length, 0);
  assert.notEqual(projectIdentity(project(empty, ['a'])), projectIdentity(project(empty, ['a', 'b'])));
});

test('own saves coming back and unknown owners are told apart from replays', () => {
  const recent = { source: THIS_TAB, timestamp: (READ_AT + 500) / 1000 };
  assert.equal(classifyProjectChange(recent, THIS_TAB, READ_AT, true), 'own');
  // Without a project there is nothing to update: the event is judged by its age like any other.
  assert.equal(classifyProjectChange(recent, THIS_TAB, READ_AT, false), 'other');
  assert.equal(classifyProjectChange({ source: THIS_TAB, timestamp: secondsBefore(9_000) }, THIS_TAB, READ_AT, false), 'replayed');
});

test('an event is old enough only when it came well before the read started', () => {
  assert.equal(seenInRead({ timestamp: secondsBefore(READ_MARGIN_MS + 50) }, READ_AT), true);
  assert.equal(seenInRead({ timestamp: secondsBefore(READ_MARGIN_MS - 50) }, READ_AT), false);
  assert.equal(seenInRead({ timestamp: (READ_AT + 10) / 1000 }, READ_AT), false);
});
