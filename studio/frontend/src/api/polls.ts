import { apiText } from './client';
import { createLiveFeed } from './liveFeed';
import type { Visibility } from './sharedPoll';

/** The page is hidden (minimised window, background tab): polls pause until it shows again. */
const pageVisibility: Visibility = {
  hidden: () => document.visibilityState === 'hidden',
  subscribe: changed => {
    document.addEventListener('visibilitychange', changed);
    return () => document.removeEventListener('visibilitychange', changed);
  },
};

/** The app's one live feed: every view that polls the backend goes through it (speed lane F). */
export const liveFeed = createLiveFeed(apiText, { visibility: pageVisibility });
export const HEALTH = '/health';
export const LINK_MAP = '/goose/links';
export const ACTIVITY = '/activity';
