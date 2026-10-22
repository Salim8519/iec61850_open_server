// "Save always" for the gateway point list: the helpers the table uses around state/autoSave.ts.
import type { GatewayPoint, Protocol } from '../../../api/gateway';

/** A whole list waiting to be saved, with the number of the edit it came from (see PointTable). */
export type PointSave = { protocol: Protocol; rows: GatewayPoint[]; edit: number };

/** The engine's refusal while the gateway runs. The list then waits for the gateway to stop. */
export const isLockedAnswer = (error: unknown) =>
  error instanceof Error && error.message.includes('Stop the gateway before changing its point list');

/** The problems to show when a save fails: the engine's per-point list, else its one sentence. */
export const saveProblems = (error: unknown, listed: string[]): string[] =>
  listed.length ? listed : [error instanceof Error ? error.message : String(error)];

// A list that had not been saved when the Points tab was left while the gateway ran (it cannot be saved
// until the gateway stops). It comes back when the tab opens again and stays kept until it is saved or
// discarded (so a page that mounts twice, as React does in development, still finds it).
const kept = new Map<Protocol, GatewayPoint[]>();
export const keepDraft = (protocol: Protocol, rows: GatewayPoint[]) => { kept.set(protocol, rows); };
export const keptDraft = (protocol: Protocol): GatewayPoint[] | undefined => kept.get(protocol);
export const clearDraft = (protocol: Protocol) => { kept.delete(protocol); };
