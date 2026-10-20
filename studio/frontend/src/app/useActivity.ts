import { ACTIVITY } from '../api/polls';
import type { Activity } from '../api/activity';
import { useLiveFeed } from '../api/useLiveFeed';

const EVERY_MS = 2000;

/** Polls what runs in the tools so every place can show its own running dot. Through the shared
 * live feed: an unchanged answer re-renders nothing, and a hidden window does not poll. */
export function useActivity(enabled: boolean) {
  return useLiveFeed<Activity>(enabled ? ACTIVITY : null, EVERY_MS).value;
}
