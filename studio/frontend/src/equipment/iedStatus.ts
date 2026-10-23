import { clientLine, externalClients, readerBadge } from '../copy/signalLayers.ts';
import type { KeyedStore } from './keyedStore.ts';
import type { SignalLayerState, SignalLayerStore } from './signalLayerStore.ts';
import { problemWords } from './signalPulses.ts';

/** Persistent words an IED component card shows (ADR-0112): its GOOSE problems while the GOOSE
 * layer is on (as words with a dashed border, never colour alone) and "◉ 2 read" while *Who is
 * reading* is on. They moved here from the old IED tags; transient pulses stay in the overlay. */
export type IedProblem = { text: string; tone: 'problem' | 'waiting' };
export type IedLive = { problems: IedProblem[]; readers: string; readersTitle: string };

/** IED id -> the card's words as JSON text (the keyed store compares text, so unchanged words
 * notify nobody). IEDs with nothing to say are left out. */
export function iedLiveWords(state: SignalLayerState): Record<string, string> {
  const { goose, readers } = state.layers;
  const names = new Map(state.map.ieds.filter(ied => ied.in_project && ied.ied_id).map(ied => [ied.ied_id!, ied.name]));
  const ids = new Set([...names.keys(), ...(readers ? Object.keys(state.clients) : [])]);
  const result: Record<string, string> = {};
  for (const id of ids) {
    const name = names.get(id);
    const problems = goose && state.map.running && name ? state.map.links
      .filter(link => link.subscriber === name && link.status === 'declared' && link.gocb_ref)
      .map(link => problemWords(link.publisher, state.goose[link.gocb_ref!] ?? link.live ?? undefined))
      .filter((item): item is IedProblem => !!item && (item.tone === 'problem' || state.listening)) : [];
    const clients = readers ? state.clients[id] : undefined;
    const badge = readerBadge(clients);
    if (!problems.length && !badge) continue;
    result[id] = JSON.stringify({ problems, readers: badge,
      readersTitle: externalClients(clients).map(clientLine).join('\n') } satisfies IedLive);
  }
  return result;
}

/** Feeds the cards' keyed store from the signal layer store. Pulses and dots change other parts
 * of the snapshot; the words are worked out again only when what they depend on changed. */
export function watchIedLive(store: SignalLayerStore, keyed: KeyedStore<string>): () => void {
  let seen: unknown[] = [];
  const update = () => {
    const state = store.get();
    const inputs = [state.map, state.goose, state.clients, state.layers, state.listening];
    if (inputs.every((value, index) => value === seen[index])) return;
    seen = inputs;
    keyed.replace(iedLiveWords(state));
  };
  update();
  return store.subscribe(update);
}
