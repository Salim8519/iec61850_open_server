import type { GooseLink, GooseLinkMap } from '../api/signalLayers';

/** When the SLD shows GOOSE activity (spec signal-layers, LAYER-11..14). Pure rules, no timers. */
export const PULSE_MS = 700;
export const LINK_FADE_MS = 2000;
export const PULSE_GAP_MS = 1000;
/** A dot from an IED outside the project travels from the canvas edge for ~2 s, then nothing remains. */
export const EDGE_PULSE_MS = 2000;

export type GooseState = { state: string; problem?: string; st_num?: number };
export type PulseKind = 'observed' | 'expected';

/** At most one pulse per link per second; later changes inside that second are coalesced. */
export class PulseGate {
  private last = new Map<string, number>();

  allow(linkId: string, now: number, gap = PULSE_GAP_MS): boolean {
    const previous = this.last.get(linkId);
    if (previous !== undefined && now - previous < gap) return false;
    this.last.set(linkId, now);
    return true;
  }

  clear() { this.last.clear(); }
}

/** Links that pulse for one listener frame: only a real stNum increase, never the first frame seen. */
export function observedLinks(map: GooseLinkMap, gocbRef: string, previous?: number, next?: number): GooseLink[] {
  if (previous === undefined || next === undefined || next <= previous) return [];
  return map.links.filter(link => link.gocb_ref === gocbRef && (link.status === 'declared' || link.status === 'remote'));
}

/** Without a listener: a changed publisher value that belongs to a sent block's dataset is
 * expected to raise stNum. These pulses are hollow and say "expected", never "received". */
export function expectedLinks(map: GooseLinkMap, publisher: string, ref: string): GooseLink[] {
  const refs = new Set(map.blocks.filter(block => block.publisher === publisher && block.status === 'sendable'
    && block.member_refs.some(member => ref === member || ref.startsWith(`${member}.`))).map(block => block.gocb_ref));
  return map.links.filter(link => link.gocb_ref !== null && refs.has(link.gocb_ref) && link.status === 'declared');
}

const PROBLEM_WORDS: Record<string, (publisher: string) => string> = {
  timeout: publisher => `GOOSE lost from ${publisher}`,
  conf_rev_mismatch: publisher => `${publisher}: configuration revision differs`,
  dataset_mismatch: publisher => `${publisher}: dataset differs`,
  go_id_mismatch: publisher => `${publisher}: GOOSE ID differs`,
  invalid_frame: publisher => `${publisher}: unreadable GOOSE`,
  needs_commission: publisher => `${publisher}: needs commissioning`,
};

/** Persistent words for a subscriber tag; null when nothing is wrong. */
export function problemWords(publisher: string, state?: GooseState): { text: string; tone: 'problem' | 'waiting' } | null {
  if (!state) return null;
  if (state.state === 'timeout') return { text: `⚠ ${PROBLEM_WORDS.timeout(publisher)}`, tone: 'problem' };
  if (state.state === 'problem') {
    const words = PROBLEM_WORDS[state.problem || ''];
    return { text: `⚠ ${words ? words(publisher) : `GOOSE problem from ${publisher}`}`, tone: 'problem' };
  }
  if (state.state === 'waiting') return { text: `… waiting for ${publisher}`, tone: 'waiting' };
  return null;
}

const DBPOS = ['Intermediate', 'Open', 'Closed', 'Invalid'];

/** Plain words for a changed value: Dbpos as Open/Closed, Booleans as on/off. */
export function valueWords(value: unknown): string {
  if (typeof value === 'boolean') return value ? 'on' : 'off';
  if (typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= 3) return String(value);
  return value === null || value === undefined ? '' : String(value).slice(0, 16);
}

export function changeText(name: string, value: unknown, dbpos: boolean): string {
  const words = dbpos && typeof value === 'number' && DBPOS[value] ? DBPOS[value] : valueWords(value);
  return `${name} ${words}`.trim();
}

export function pulseLabel(publisher: string, subscriber: string, change: string, kind: PulseKind): string {
  return `${publisher} → ${subscriber} · ${change || 'new state'}${kind === 'expected' ? ' (expected)' : ''}`;
}

/** True when the operating system asks for reduced motion; the overlay then shows a static highlight. */
export function reducedMotion(match?: (query: string) => { matches: boolean }): boolean {
  const media = match ?? (typeof window !== 'undefined' && window.matchMedia ? window.matchMedia.bind(window) : undefined);
  return !!media && media('(prefers-reduced-motion: reduce)').matches;
}
