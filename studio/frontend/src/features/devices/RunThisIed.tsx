import { Play, Plug, Square, Unplug } from 'lucide-react';
import { api } from '../../api/client';
import type { Ied, Project } from '../../api/types';
import { RUN_ONE_COPY } from '../../copy/running';

/** Run or connect only this IED (ADR-0067). The top bar Run still starts everything not set to Off.
 * Simulated: its MMS server, reports, polling answers, GOOSE and all its points.
 * Real: a client read (free report blocks, else polling) that sends nothing unless you operate. */
export function RunThisIed({ ied, project, busy, perform }: {
  ied: Ied; project: Project; busy: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  if (ied.role === 'off') return null;
  const inRun = (project.running_ieds || []).includes(ied.id);
  const copy = RUN_ONE_COPY[ied.role === 'real' ? 'real' : 'simulated'];
  const Icon = ied.role === 'real' ? (inRun ? Unplug : Plug) : (inRun ? Square : Play);
  return <button className={inRun ? undefined : 'primary'} disabled={busy} title={inRun ? copy.stopHint : copy.startHint}
    onClick={() => perform(() => api(`/runtime/ieds/${encodeURIComponent(ied.id)}/${inRun ? 'stop' : 'start'}`, 'POST'))}>
    <Icon size={16} />{inRun ? copy.stop : copy.start}</button>;
}
