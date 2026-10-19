import { useEffect, useState } from 'react';
import { HEALTH, polls } from '../../api/polls';

export type ModeSummary = { devices: Record<string, string>; simulation: boolean | null; worst: string | null };
export type Modes = { ieds: Record<string, ModeSummary>; equipment: Record<string, string> };

/** Mode badges and simulation sources from the backend health result (display only). */
export function useModes(running: boolean, revision: number) {
  const [state, setState] = useState<{ modes: Modes; simulation: string[] }>({ modes: { ieds: {}, equipment: {} }, simulation: [] });
  useEffect(() => polls.subscribe<{ modes: Modes; simulation_sources: string[] }>(HEALTH, running ? 3000 : null,
    value => setState({ modes: value.modes, simulation: value.simulation_sources })), [running, revision]);
  return state;
}
