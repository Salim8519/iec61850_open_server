import type { CSSProperties } from 'react';

/** Section states computed by the backend (spec energization); the browser only displays them. */
export type EnergyState = 'live' | 'maybe_live' | 'dead' | 'earthed' | 'earth_fault';
export type SwitchHint = { code: 'load' | 'unknown' | 'earth'; text: string; related: string[] };
export type SwitchHints = { open: SwitchHint[]; close: SwitchHint[] };
export type Energization = {
  nodes: Record<string, EnergyState[]>; edges: Record<string, EnergyState>; hints?: Record<string, SwitchHints>;
};
export const NO_ENERGY: Energization = { nodes: {}, edges: {}, hints: {} };

type Look = { word: string; meaning: string; colour: string; width: number; dash?: string };

/** Each state differs by line pattern and width as well as colour (never colour alone). */
export const ENERGY_LOOK: Record<EnergyState, Look> = {
  live: { word: 'Live', meaning: 'Power reaches here through closed switches', colour: '#b42318', width: 4 },
  maybe_live: { word: 'Maybe live', meaning: 'Live if a switch with an unknown position is closed',
    colour: '#a15c07', width: 3, dash: '2 6' },
  dead: { word: 'Dead', meaning: 'No power reaches here', colour: '#5b6472', width: 2, dash: '7 5' },
  earthed: { word: 'Earthed', meaning: 'Connected to earth by a closed earth switch',
    colour: '#15803d', width: 3, dash: '12 4 2 4' },
  earth_fault: { word: 'Earth fault', meaning: 'Live and earthed at the same time',
    colour: '#86198f', width: 5, dash: '14 4' },
};
export const ENERGY_ORDER: EnergyState[] = ['live', 'maybe_live', 'dead', 'earthed', 'earth_fault'];

export function wireStyle(state: EnergyState): CSSProperties {
  const look = ENERGY_LOOK[state];
  return { stroke: look.colour, strokeWidth: look.width, strokeDasharray: look.dash, strokeLinecap: 'round' };
}

/** Plain sentence for the properties sheet: one section, or the two sides of an open switch. */
export function sectionSentence(states: EnergyState[] | undefined): string {
  if (!states?.length) return '';
  const words = states.map(state => ENERGY_LOOK[state].word.toLowerCase());
  if (words.length === 1) return `This part is ${words[0]}.`;
  return `One side is ${words[0]}, the other side is ${words.slice(1).join(' / ')}.`;
}

/** One shared snapshot; the canvas re-renders only when the backend reports a change. */
export class EnergizationStore {
  private value: Energization = NO_ENERGY;
  private listeners = new Set<() => void>();

  get = () => this.value;

  set(value: Energization | null | undefined) {
    this.value = value && (Object.keys(value.edges).length || Object.keys(value.nodes).length) ? value : NO_ENERGY;
    this.listeners.forEach(listener => listener());
  }

  clear() { this.set(NO_ENERGY); }

  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  };
}

type Edge = { id: string; className?: string; style?: CSSProperties };
export type EdgeCache = Map<string, { source: unknown; state: EnergyState | undefined; projected: unknown }>;

/** Wires with their section style; unchanged wires keep their object so React Flow skips them. */
export function energizedEdges<T extends Edge>(edges: T[], states: Energization['edges'], previous: EdgeCache) {
  const cache: EdgeCache = new Map();
  const result = edges.map(edge => {
    const state = states[edge.id];
    const hit = previous.get(edge.id);
    if (hit && hit.source === edge && hit.state === state) { cache.set(edge.id, hit); return hit.projected as T; }
    const projected = state ? { ...edge, className: `wire-${state}`, style: wireStyle(state) } : edge;
    cache.set(edge.id, { source: edge, state, projected });
    return projected;
  });
  return { edges: result, cache };
}
