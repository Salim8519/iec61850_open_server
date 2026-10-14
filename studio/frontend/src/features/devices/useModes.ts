import { useEffect, useState } from 'react';
import { api } from '../../api/client';

export type ModeSummary = { devices: Record<string, string>; simulation: boolean | null; worst: string | null };
export type Modes = { ieds: Record<string, ModeSummary>; equipment: Record<string, string> };

/** Mode badges and simulation sources from the backend health result (display only). */
export function useModes(running: boolean, revision: number) {
  const [state, setState] = useState<{ modes: Modes; simulation: string[] }>({ modes: { ieds: {}, equipment: {} }, simulation: [] });
  useEffect(() => {
    let cancelled = false;
    const load = () => api<{ modes: Modes; simulation_sources: string[] }>('/health')
      .then(value => { if (!cancelled) setState({ modes: value.modes, simulation: value.simulation_sources }); }).catch(() => {});
    load();
    const timer = running ? setInterval(load, 3000) : undefined;
    return () => { cancelled = true; if (timer) clearInterval(timer); };
  }, [running, revision]);
  return state;
}
