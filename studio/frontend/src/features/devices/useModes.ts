import { useEffect, useState } from 'react';
import { HEALTH, liveFeed } from '../../api/polls';
import type { KeyedStore } from '../../equipment/keyedStore';

export type ModeSummary = { devices: Record<string, string>; simulation: boolean | null; worst: string | null };
export type Modes = { ieds: Record<string, ModeSummary>; equipment: Record<string, string> };
type HealthModes = { modes: Modes; simulation_sources: string[] };

/** Mode badges and simulation sources from the backend health result (display only). */
export function useModes(running: boolean, revision: number) {
  const [state, setState] = useState<{ modes: Modes; simulation: string[] }>({ modes: { ieds: {}, equipment: {} }, simulation: [] });
  useEffect(() => {
    // /health also carries check times that change on every answer; re-render only for new modes.
    let last = '';
    return liveFeed.subscribe<HealthModes>(HEALTH, running ? 3000 : null, value => {
      const next = { modes: value.modes, simulation: value.simulation_sources };
      const text = JSON.stringify(next);
      if (text !== last) { last = text; setState(next); }
    });
  }, [running, revision]);
  return state;
}

/** The SLD's equipment mode badges go into `badges` key by key: only an equipment whose badge
 * changed re-renders, and the diagram page itself does not (speed lane F). */
export function useModeBadges(badges: KeyedStore<string>, running: boolean, revision: number) {
  useEffect(() => liveFeed.subscribe<HealthModes>(HEALTH, running ? 3000 : null,
    value => { badges.replace(value.modes.equipment); }), [badges, running, revision]);
}
