import type { GooseMembers } from '../api/gooseMessages';
import { leafIndex, type LeafIndex } from './gooseLeaves.ts';
import type { GooseLinkMap, MmsClient } from '../api/signalLayers';
import { TRIGGER_MS, triggerOf, type Trigger } from './gooseTriggers.ts';
import { changeText, expectedLinks, LINK_FADE_MS, observedLinks, PulseGate, pulseLabel, receivedLabel,
  type GooseState, type PulseKind, type SignalChange } from './signalPulses.ts';

/** `cause`: key of the publisher signal change that raised it (ADR-0103), when known. */
export type Pulse = { key: number; linkId: string; kind: PulseKind; label: string; short: string; change: string;
  start: number; cause?: number };
export type ControlPulse = { key: number; iedId: string; peer: string; label: string; start: number };
export type Layers = { goose: boolean; readers: boolean };
export type SignalLayerState = {
  layers: Layers; map: GooseLinkMap; goose: Record<string, GooseState>;
  listening: boolean; pulses: Pulse[]; controls: ControlPulse[]; clients: Record<string, MmsClient[]>;
  /** Recent publisher changes that belong to a sent dataset, and the backend's latest equipment states
   * (`equipment_state`), so one change can be worded "CB Y130 open" (ADR-0103). View data only. */
  signals: SignalChange[]; equipment: Record<string, { value: unknown; at: number }>;
  /** Recent user actions on components (ADR-0112): where a change starts. View data only. */
  triggers: Trigger[];
  /** Exact leaves of every sent message, from Go (GET /goose/members); empty with an older engine. */
  members: LeafIndex;
};
type Event = { kind: string; ied_id?: string; ref?: string; value?: unknown; [key: string]: unknown };

export const EMPTY_MAP: GooseLinkMap = { ieds: [], blocks: [], links: [], diagnostics: [], summary: {}, running: false };
const EMPTY: SignalLayerState = { layers: { goose: false, readers: false }, map: EMPTY_MAP, goose: {},
  listening: false, pulses: [], controls: [], clients: {}, signals: [], equipment: {}, triggers: [], members: new Map() };

/** One snapshot for the SLD signal layers, fed by live events. Only the overlay subscribes (and the
 * IED cards through a keyed store of their own words), so GOOSE and MMS traffic never re-renders
 * equipment nodes (interaction contract). */
export class SignalLayerStore {
  private value: SignalLayerState = EMPTY;
  private listeners = new Set<() => void>();
  private gate = new PulseGate();
  private counter = 0;
  private recent = new Map<string, { text: string; at: number; cause: number }>();

  get = () => this.value;

  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  };

  /** `quiet`: keep the data current without telling anyone (its layer is off, nothing shows it). */
  private update(next: Partial<SignalLayerState>, quiet = false) {
    this.value = { ...this.value, ...next };
    if (!quiet) this.listeners.forEach(listener => listener());
  }

  /** The two switches only; the same object until they change, so the toolbar is not re-rendered. */
  layers = () => this.value.layers;

  setLayers(layers: Layers) {
    const now = this.value.layers;
    if (now.goose === layers.goose && now.readers === layers.readers) return;
    this.update({ layers });
  }

  setMap(map: GooseLinkMap) {
    if (map === this.value.map) return;
    const goose = { ...this.value.goose };
    for (const link of map.links) {
      if (link.gocb_ref && link.live && link.live.state !== 'stopped' && !goose[link.gocb_ref]) goose[link.gocb_ref] = link.live;
    }
    const listening = map.running && (this.value.listening || map.links.some(link => link.live && link.live.state !== 'stopped'));
    this.update({ map, goose, listening }, !this.value.layers.goose && !this.value.layers.readers);
  }

  /** Go's exact message leaves; indexed once per answer. */
  setMembers(members: GooseMembers) { this.update({ members: leafIndex(members) }, true); }

  setClients(clients: Record<string, MmsClient[]>) { this.update({ clients }, !this.value.layers.readers); }

  /** Stop clears everything live; the declared map stays. */
  stopped() {
    this.gate.clear(); this.recent.clear();
    this.update({ goose: {}, listening: false, pulses: [], controls: [], clients: {}, signals: [], equipment: {}, triggers: [] });
  }

  private pulse(links: { id: string; publisher: string; subscriber: string; gocb_ref: string | null }[],
    kind: PulseKind, change: string, now: number, cause?: number) {
    // Full IED names everywhere on the SLD (change 0097): "C03_Y1105", never "C03".
    const fresh = links.filter(link => this.gate.allow(link.id, now)).map(link => ({
      key: ++this.counter, linkId: link.id, kind, start: now, change, ...(cause === undefined ? {} : { cause }),
      label: pulseLabel(link.publisher, link.subscriber, change, kind),
      short: receivedLabel(link.publisher, change, kind) }));
    if (fresh.length) this.update({ pulses: [...this.value.pulses.filter(item => now - item.start < LINK_FADE_MS), ...fresh] },
      !this.value.layers.goose);
  }

  handle(event: Event, now: number, describe?: (iedId: string, ref: string, value: unknown) => string) {
    const trigger = triggerOf(event, now, this.counter + 1);
    if (trigger) {
      this.counter = trigger.key;
      const triggers = [...this.value.triggers.filter(item => now - item.at < 2 * TRIGGER_MS), trigger];
      // A trigger after its own change (an operate's feedback) re-plans what is already drawn.
      this.update({ triggers }, !this.value.layers.goose || (!this.value.pulses.length && !this.value.signals.length));
      return;
    }
    if (event.kind === 'goose_status' && typeof event.gocb_ref === 'string') {
      const previous = this.value.goose[event.gocb_ref];
      const state = { state: String(event.state), problem: String(event.problem || ''),
        st_num: typeof event.st_num === 'number' ? event.st_num : undefined };
      this.update({ goose: { ...this.value.goose, [event.gocb_ref]: state }, listening: true }, !this.value.layers.goose);
      const recent = this.recent.get(event.gocb_ref);
      const known = recent && now - recent.at < 3000 ? recent : undefined;
      const change = known ? known.text : state.st_num !== undefined ? `stNum ${state.st_num}` : '';
      this.pulse(observedLinks(this.value.map, event.gocb_ref, previous?.st_num, state.st_num), 'observed', change, now, known?.cause);
    } else if (event.kind === 'signal' && event.ied_id && event.ref) {
      const publisher = this.value.map.ieds.find(ied => ied.ied_id === event.ied_id)?.name;
      if (!publisher) return;
      // Go's exact leaves when it listed this IED (a leaf it does not list is in no message).
      const exact = this.value.members.get(event.ied_id);
      const links = expectedLinks(this.value.map, publisher, event.ref, exact ? exact.get(event.ref) ?? [] : undefined);
      if (!links.length) return;
      const node = event.ref.split('/').pop() || event.ref;
      const text = describe?.(event.ied_id, event.ref, event.value) || changeText(node.replace(/\.stVal$/, ''), event.value,
        /\.Pos\.stVal$/.test(event.ref));
      // Every change is kept, even one the per-link gate silences, so its partner contact joins it.
      const signal: SignalChange = { key: ++this.counter, iedId: event.ied_id, publisher, ref: event.ref, value: event.value,
        text, start: now, linkIds: links.map(link => link.id) };
      this.update({ signals: [...this.value.signals.filter(item => now - item.start < LINK_FADE_MS), signal] },
        !this.value.layers.goose || !this.value.pulses.length);
      for (const link of links) if (link.gocb_ref) this.recent.set(link.gocb_ref, { text, at: now, cause: signal.key });
      if (!this.value.listening) this.pulse(links, 'expected', text, now, signal.key);
    } else if (event.kind === 'equipment_state' && typeof event.node_id === 'string') {
      const equipment = { ...this.value.equipment, [event.node_id]: { value: event.value, at: now } };
      this.update({ equipment }, !this.value.layers.goose || !this.value.pulses.length);
    } else if (event.kind === 'mms_access' && event.ied_id && Array.isArray(event.clients)) {
      this.update({ clients: { ...this.value.clients, [event.ied_id]: event.clients as MmsClient[] } }, !this.value.layers.readers);
    } else if (event.kind === 'control_applied' && event.external === true && event.ied_id) {
      const name = this.value.map.ieds.find(ied => ied.ied_id === event.ied_id)?.name || '';
      const peer = String(event.peer || '');
      const control = { key: ++this.counter, iedId: event.ied_id, peer, start: now,
        label: `${peer.split(':')[0] || 'A client'} operated ${String(event.ref || '').split('/').pop()}${name ? ` on ${name}` : ''}` };
      this.update({ controls: [...this.value.controls.filter(item => now - item.start < LINK_FADE_MS), control] });
    }
  }

  /** Drop pulses whose link has faded; returns ms until the next expiry, or null. */
  prune(now: number): number | null {
    const pulses = this.value.pulses.filter(item => now - item.start < LINK_FADE_MS);
    const controls = this.value.controls.filter(item => now - item.start < LINK_FADE_MS);
    const signals = this.value.signals.filter(item => now - item.start < LINK_FADE_MS);
    const triggers = this.value.triggers.filter(item => now - item.at < 2 * TRIGGER_MS);
    if (pulses.length !== this.value.pulses.length || controls.length !== this.value.controls.length) {
      this.update({ pulses, controls, signals, triggers });
    } else if (signals.length !== this.value.signals.length || triggers.length !== this.value.triggers.length) {
      this.update({ signals, triggers }, true);
    }
    const starts = [...pulses, ...controls].map(item => item.start + LINK_FADE_MS - now);
    return starts.length ? Math.max(16, Math.min(...starts)) : null;
  }
}
