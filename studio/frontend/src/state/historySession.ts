import type { Diagram, Project } from '../api/types';
import { createHistory, recordDiagram, redoDiagram, undoDiagram, type DiagramHistory } from './diagramHistory.ts';

/** The Undo history of the open project, and which project it belongs to. */
export type HistorySession = { identity: string; history: DiagramHistory | null };
type Known = Pick<Project, 'session_id' | 'id' | 'ieds'>;

export const NO_HISTORY: HistorySession = { identity: '', history: null };

/** What a history belongs to: the app session, the project and its IEDs (adding one changes the bindings). */
export function projectIdentity(project: Known): string {
  return `${project.session_id}:${project.id}:${project.ieds.map(ied => ied.id).join(',')}`;
}

/** A project was read. The same project keeps its history; another one (or one that was forgotten)
 * starts a new history from what was read. */
export function adoptProject(session: HistorySession, project: Known & Pick<Project, 'diagram'>): HistorySession {
  const identity = projectIdentity(project);
  return identity === session.identity ? session : { identity, history: createHistory(project.diagram) };
}

/** What is saved no longer matches the history (a new change by another tab, a failed save, another
 * file opened): the next project read starts a new history. */
export function forgetHistory(session: HistorySession): HistorySession {
  return { ...session, identity: '' };
}

/** A local edit is one more step on the history in hand. It never looks at the identity, so an edit
 * can never start a new history, whatever the push stream replayed before it. */
export function recordEdit(session: HistorySession, diagram: Diagram, group?: string): HistorySession {
  return session.history ? { ...session, history: recordDiagram(session.history, diagram, group) } : session;
}

export function stepHistory(session: HistorySession, direction: 'undo' | 'redo'): HistorySession {
  if (!session.history) return session;
  return { ...session, history: (direction === 'undo' ? undoDiagram : redoDiagram)(session.history) };
}
