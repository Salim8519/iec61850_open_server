import type { StudioEvent } from '../api/types';

/** Events this much older than a project read are certainly in it (same PC clock; generous margin). */
export const READ_MARGIN_MS = 250;

/** True when the backend published `event` well before the last project read started: that read
 * (the backend publishes after applying a change) already holds the change. */
export function seenInRead(event: Pick<StudioEvent, 'timestamp'>, readAt: number): boolean {
  return event.timestamp * 1000 < readAt - READ_MARGIN_MS;
}

/** What a `project_changed` event means for this tab.
 * - own: this tab's own save came back; only the revision moves.
 * - replayed: an older change that the project read already holds. The push stream replays the whole
 *   bus on every connect, so every page load sees earlier tabs' changes; they change nothing here.
 * - other: a really new change by someone else; the pending save and the Undo history no longer fit.
 */
export type ProjectChange = 'own' | 'replayed' | 'other';

export function classifyProjectChange(event: { timestamp: number; source?: unknown }, tab: string,
  readAt: number, hasProject: boolean): ProjectChange {
  if (event.source === tab && hasProject) return 'own';
  return seenInRead(event, readAt) ? 'replayed' : 'other';
}
