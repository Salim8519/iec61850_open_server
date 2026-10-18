import type { GooseLinkMap, MmsClient } from '../api/signalLayers';
import { changeText, expectedLinks, LINK_FADE_MS, observedLinks, PulseGate, pulseLabel,
  type GooseState, type PulseKind } from './signalPulses.ts';

export type Pulse = { key: number; linkId: string; kind: PulseKind; label: string; start: number };
export type ControlPulse = { key: number; iedId: string; peer: string; label: string; start: number };
export type Layers = { goose: boolean; readers: boolean };
export type SignalLayerState = {
  layers: Layers; map: GooseLinkMap; goose: Record<string, GooseState>;
  listening: boolean; pulses: Pulse[]; controls: ControlPulse[]; clients: Record<string, MmsClient[]>;
};
type Event = { kind: string; ied_id?: string; ref?: string; value?: unknown; [key: string]: unknown };

export const EMPTY_MAP: GooseLinkMap = { ieds: [], blocks: [], links: [], diagnostics: [], summary: {}, running: false };
const EMPTY: SignalLayerState = { layers: { goose: false, readers: false }, map: EMPTY_MAP, goose: {},
  listening: false, pulses: [], controls: [], clients: {} };

/** One snapshot for the SLD signal layers, fed by live events. Only the overlay subscribes,
 * so GOOSE and MMS traffic never re-renders the equipment nodes (interaction contract). */
export class SignalLayerStore {
  private value: SignalLayerState = EMPTY;
  private listeners = new Set<() => void>();
  private gate = new PulseGate();
  private counter = 0;
  private recent = new Map<string, { text: string; at: number }>();

  get = () => this.value;

  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  };

  private update(next: Partial<SignalLayerState>) {
    this.value = { ...this.value, ...next };
    this.listeners.forEach(listener => listener());
  }

  setLayers(layers: Layers) { this.update({ layers }); }

  setMap(map: GooseLinkMap) {
    const goose = { ...this.value.goose };
    for (const link of map.links) {
      if (link.gocb_ref && link.live && link.live.state !== 'stopped' && !goose[link.gocb_ref]) goose[link.gocb_ref] = link.live;
    }
    const listening = map.running && (this.value.listening || map.links.some(link => link.live && link.live.state !== 'stopped'));
    this.update({ map, goose, listening });
  }

  setClients(clients: Record<string, MmsClient[]>) { this.update({ clients }); }

  /** Stop clears everything live; the declared map stays. */
  stopped() {
    this.gate.clear(); this.recent.clear();
    this.update({ goose: {}, listening: false, pulses: [], controls: [], clients: {} });
  }

  private pulse(links: { id: string; publisher: string; subscriber: string; gocb_ref: string | null }[],
    kind: PulseKind, change: string, now: number) {
    // Full IED names everywhere on the SLD (change 0097): "C03_Y1105", never "C03".
    const fresh = links.filter(link => this.gate.allow(link.id, now)).map(link => ({
      key: ++this.counter, linkId: link.id, kind, start: now,
      label: pulseLabel(link.publisher, link.subscriber, change, kind) }));
    if (fresh.length) this.update({ pulses: [...this.value.pulses.filter(item => now - item.start < LINK_FADE_MS), ...fresh] });
  }

  handle(event: Event, now: number, describe?: (iedId: string, ref: string, value: unknown) => string) {
    if (event.kind === 'goose_status' && typeof event.gocb_ref === 'string') {
      const previous = this.value.goose[event.gocb_ref];
      const state = { state: String(event.state), problem: String(event.problem || ''),
        st_num: typeof event.st_num === 'number' ? event.st_num : undefined };
      this.update({ goose: { ...this.value.goose, [event.gocb_ref]: state }, listening: true });
      const recent = this.recent.get(event.gocb_ref);
      const change = recent && now - recent.at < 3000 ? recent.text : state.st_num !== undefined ? `stNum ${state.st_num}` : '';
      this.pulse(observedLinks(this.value.map, event.gocb_ref, previous?.st_num, state.st_num), 'observed', change, now);
    } else if (event.kind === 'signal' && event.ied_id && event.ref) {
      const publisher = this.value.map.ieds.find(ied => ied.ied_id === event.ied_id)?.name;
      if (!publisher) return;
      const links = expectedLinks(this.value.map, publisher, event.ref);
      if (!links.length) return;
      const node = event.ref.split('/').pop() || event.ref;
      const text = describe?.(event.ied_id, event.ref, event.value) || changeText(node.replace(/\.stVal$/, ''), event.value,
        /\.Pos\.stVal$/.test(event.ref));
      for (const link of links) if (link.gocb_ref) this.recent.set(link.gocb_ref, { text, at: now });
      if (!this.value.listening) this.pulse(links, 'expected', text, now);
    } else if (event.kind === 'mms_access' && event.ied_id && Array.isArray(event.clients)) {
      this.update({ clients: { ...this.value.clients, [event.ied_id]: event.clients as MmsClient[] } });
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
    if (pulses.length !== this.value.pulses.length || controls.length !== this.value.controls.length) this.update({ pulses, controls });
    const starts = [...pulses, ...controls].map(item => item.start + LINK_FADE_MS - now);
    return starts.length ? Math.max(16, Math.min(...starts)) : null;
  }
}
