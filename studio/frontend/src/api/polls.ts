import { api } from './client';
import { createPoller } from './sharedPoll';

/** The app's shared pollers: views showing the same backend state share one request. */
export const polls = createPoller(path => api(path));
export const HEALTH = '/health';
export const LINK_MAP = '/goose/links';
