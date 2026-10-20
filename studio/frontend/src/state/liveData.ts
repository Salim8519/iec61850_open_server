import type { StudioEvent } from '../api/types';
import { eventDetails } from '../features/event-log/eventDetails';
import { createEventLogStore } from './eventLogStore';
import { browserFrameBatch } from './frameBatch';
import { createValueStore } from './valueStore';

/** The page's live stores, made on first use (no work at import time): data point values and
 * the Activity log, both flushed once per animation frame. */
function make() {
  const batch = browserFrameBatch();
  return {
    batch,
    values: createValueStore(batch),
    log: createEventLogStore<StudioEvent>(batch, { describe: eventDetails }),
  };
}

let data: ReturnType<typeof make> | null = null;

export function liveData() {
  if (!data) data = make();
  return data;
}

export type LiveData = ReturnType<typeof liveData>;
