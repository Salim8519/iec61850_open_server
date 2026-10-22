import type { GooseLink } from '../api/signalLayers';
import type { Pulse } from './signalLayerStore.ts';
import type { Origin, OriginLookup } from './signalOrigin.ts';
import type { PulseKind, SignalChange } from './signalPulses.ts';

export type { SignalChange };

/** One real change on the SLD (ADR-0103): publisher signal changes that land on the same symbol
 * within COALESCE_MS (a breaker's open and closed contacts) are one change with one line per
 * receiving IED, not one line per link and per contact. Pure grouping of store data. */
export const COALESCE_MS = 400;

export type OriginSignal = SignalChange & { origin: Origin | null };
export type Coalesced = { id: string; publisher: string; iedId: string | null; origin: Origin | null; start: number;
  signals: OriginSignal[] };
export type ChangeLine = { pulse: Pulse; link: GooseLink };
/** `receivers`: every in-project IED the change is sent to; `got`: those with a real stNum increase. */
export type ChangeGroup = Coalesced & { kind: PulseKind; lines: ChangeLine[]; receivers: string[]; got: string[] };

const nodeOf = (origin: Origin | null) => origin?.nodeId ?? null;

/** Signals from one publisher that map to the same symbol (or to no symbol) within the window. */
export function coalesceSignals(signals: SignalChange[], originOf: OriginLookup, window = COALESCE_MS): Coalesced[] {
  const groups: Coalesced[] = [];
  for (const signal of [...signals].sort((a, b) => a.start - b.start || a.key - b.key)) {
    const origin = originOf(signal.iedId, signal.ref);
    const group = groups.find(item => item.publisher === signal.publisher && nodeOf(item.origin) === nodeOf(origin)
      && signal.start - item.start <= window);
    const entry = { ...signal, origin };
    if (group) group.signals.push(entry);
    else groups.push({ id: `s${signal.key}`, publisher: signal.publisher, iedId: signal.iedId, origin, start: signal.start, signals: [entry] });
  }
  return groups;
}

const rank = (line: ChangeLine) => (line.pulse.kind === 'observed' ? 0 : 1);

/** Pulses joined to the change that caused them; a pulse with no known cause (only a new stNum)
 * groups with its publisher's other pulses inside the window. One line per receiving IED. */
export function changeGroups(pulses: Pulse[], signals: SignalChange[], links: Map<string, GooseLink>,
  originOf: OriginLookup, window = COALESCE_MS): ChangeGroup[] {
  const coalesced = coalesceSignals(signals, originOf, window);
  const bySignal = new Map(coalesced.flatMap(group => group.signals.map(signal => [signal.key, group] as const)));
  const lines = new Map<Coalesced, ChangeLine[]>();
  const loose: Coalesced[] = [];
  for (const pulse of [...pulses].sort((a, b) => a.start - b.start || a.key - b.key)) {
    const link = links.get(pulse.linkId);
    if (!link) continue;
    let group = pulse.cause === undefined ? undefined : bySignal.get(pulse.cause);
    if (!group) {
      group = loose.find(item => item.publisher === link.publisher && pulse.start - item.start <= window);
      if (!group) loose.push(group = { id: `p${pulse.key}`, publisher: link.publisher, iedId: null, origin: null, start: pulse.start, signals: [] });
    }
    lines.set(group, [...(lines.get(group) || []), { pulse, link }]);
  }
  const result: ChangeGroup[] = [];
  for (const [group, all] of lines) {
    const best = new Map<string, ChangeLine>();
    for (const line of all) {
      const known = best.get(line.link.subscriber);
      if (!known || rank(line) < rank(known)) best.set(line.link.subscriber, line);
    }
    const chosen = [...best.values()].sort((a, b) => a.link.subscriber.localeCompare(b.link.subscriber));
    const sent = group.signals.flatMap(signal => signal.linkIds).map(id => links.get(id)?.subscriber)
      .filter((name): name is string => !!name);
    const receivers = [...new Set([...sent, ...chosen.map(line => line.link.subscriber)])].sort();
    const got = chosen.filter(line => line.pulse.kind === 'observed').map(line => line.link.subscriber);
    result.push({ ...group, kind: got.length ? 'observed' : 'expected', lines: chosen, receivers, got,
      start: Math.min(group.start, ...all.map(line => line.pulse.start)) });
  }
  return result.sort((a, b) => a.start - b.start);
}
