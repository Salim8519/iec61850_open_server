import type { StudioEvent } from '../api/types';
import type { EventLogStore } from './eventLogStore';
import { eventValues, type ValueStore } from './valueStore.ts';

/** The page's one live event stream (speed lane F).
 *
 * With the Python engine it is the SSE bus `GET /api/v1/events`. The Go engine's push stream carries
 * the same bus events plus topic frames; `api/pushClient.ts` plugs it in with `setStreamFactory` at
 * start-up (it falls back to `sseStream` when the backend has no push) and the rest of the app does
 * not change: `open` fires when the stream is (re)connected, `event` for every bus event. */
export type StreamHandlers = { open: () => void; event: (event: StudioEvent) => void };
export type StudioStream = { close: () => void };
export type StreamFactory = (handlers: StreamHandlers) => StudioStream;

export const sseStream: StreamFactory = ({ open, event }) => {
  const source = new EventSource('/api/v1/events');
  source.onopen = open;
  source.onmessage = message => event(JSON.parse(message.data) as StudioEvent);
  return { close: () => source.close() };
};

let factory: StreamFactory = sseStream;

/** Replace the transport (for example with the push stream when the backend offers it). */
export function setStreamFactory(next: StreamFactory) { factory = next; }

export function openEventStream(handlers: StreamHandlers): StudioStream { return factory(handlers); }

type Layers = { handle: (event: StudioEvent, now: number) => void };
type Equipment = { set: (key: string, value: unknown) => void };
export type LiveTargets<E extends StudioEvent> = { values: ValueStore; log: EventLogStore<E>; layers: Layers;
  equipment: Equipment; energize: (event: StudioEvent) => void };

/** Hand one event to the live stores; none of this touches React state.
 * Per-second MMS client summaries feed the SLD layer only, not Activity (spec signal-layers). */
export function deliverLive(event: StudioEvent, targets: LiveTargets<StudioEvent>, now: number) {
  if (event.kind !== 'mms_access') targets.log.add(event);
  targets.layers.handle(event, now);
  for (const [key, value] of eventValues(event)) targets.values.set(key, value);
  if (event.kind === 'equipment_state' && event.node_id) targets.equipment.set(event.node_id, event.value);
  if (event.kind === 'energization') targets.energize(event);
}

/** Events after which the project must be read again. */
export const RELOADS = new Set(['runtime', 'resync', 'worker_failed', 'live_status']);
